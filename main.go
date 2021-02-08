package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	config struct {
		RPS        int
		Names      string
		Logging    bool
		Timeout    time.Duration
		Duration   time.Duration
		MetricAddr string
	}

	dnsLookupHisto = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "dnsperf",
		Subsystem: "lookup",
		Name:      "duration_seconds",
		Help:      "Duration for DNS lookup measured by the client",
		Buckets:   []float64{0.001, 0.01, 0.02, 0.03, 0.04, 0.05, 0.06, 0.07, 0.08, 0.09, 0.1, 0.25, 0.5, 1, 2, 3, 4, 5},
	})
	dnsLookupErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "dnsperf",
			Subsystem: "lookup",
			Name:      "errors_total",
			Help:      "Number of lookup errors.",
		},
	)
	dnsLookupSuccessTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "dnsperf",
			Subsystem: "lookup",
			Name:      "success_total",
			Help:      "Number of successful lookups.",
		},
	)
)

func main() {
	flag.IntVar(&config.RPS, "rps", 10, "DNS lookups per second.")
	flag.StringVar(&config.Names, "names", "google.com", "Comma separated list of hostnames")
	flag.BoolVar(&config.Logging, "enable-logging", true, "Whether to enable logging or not")
	flag.DurationVar(&config.Timeout, "timeout", 1*time.Second, "Timeout for DNS queries")
	flag.DurationVar(&config.Duration, "duration", -1, "Duration of the test, defaults to forever")
	flag.StringVar(&config.MetricAddr, "metric-addr", ":9090", "Metric address to listen on")
	flag.Parse()

	hostNames := strings.Split(config.Names, ",")

	log.Print(config.RPS)

	ticker := time.NewTicker(1 * time.Second / time.Duration(config.RPS))
	defer ticker.Stop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	i := 0

	dnsResolver := &CustomResolver{
		resolver: &net.Resolver{},
		logging:  config.Logging,
		hg:       newHG(),
	}

	var done <-chan time.Time
	if config.Duration > 0 {
		done = time.After(config.Duration)
	}

	server := setupServer(config.MetricAddr)

	ctx, cancel := context.WithCancel(context.Background())
	go handleSigterm(server, cancel)
	go func(server *http.Server) {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metric server failed: %v", err)
		}
	}(server)

	for {
		select {
		case <-ticker.C:
			go dnsResolver.Lookup(hostNames[i], config.Timeout)
		case <-sigs:
			dnsResolver.hg.printStats()
			return
		case <-done:
			dnsResolver.hg.printStats()
			return
		case <-ctx.Done():
			dnsResolver.hg.printStats()
			return
		}

		if i < len(hostNames)-1 {
			i++
		} else {
			i = 0
		}
	}
}

type CustomResolver struct {
	resolver *net.Resolver
	logging  bool
	hg       *hg
}

func (r *CustomResolver) Lookup(name string, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	start := time.Now()
	names, err := r.resolver.LookupHost(ctx, name)
	cancel()
	duration := time.Since(start)
	dnsLookupHisto.Observe(duration.Seconds())
	if err != nil {
		dnsLookupErrorsTotal.Inc()
		r.hg.record(false, duration)
		if r.logging {
			log.Printf("[ERROR] %s", err)
		} else {
			fmt.Fprintf(
				os.Stderr,
				"\rsuccess=%d, failure=%d",
				len(r.hg.success),
				len(r.hg.failure),
			)
		}

		return
	}

	dnsLookupSuccessTotal.Inc()
	r.hg.record(true, duration)
	if r.logging {
		log.Printf("%s - %s", name, names)
	} else {
		fmt.Fprintf(
			os.Stderr,
			"\rsuccess=%d, failure=%d         ",
			r.hg.successCount,
			r.hg.failureCount,
		)
	}
}

type hg struct {
	mx           *sync.Mutex
	success      map[time.Duration]int
	failure      map[time.Duration]int
	successCount int
	failureCount int
}

func newHG() *hg {
	return &hg{
		mx:      &sync.Mutex{},
		success: make(map[time.Duration]int),
		failure: make(map[time.Duration]int),
	}
}

func (hg *hg) record(success bool, d time.Duration) {
	var bucket time.Duration
	nextBucket := 100 * time.Microsecond
	for d > nextBucket {
		bucket = nextBucket
		nextBucket <<= 1
	}

	hg.mx.Lock()
	defer hg.mx.Unlock()

	if success {
		hg.success[bucket]++
		hg.successCount++
	} else {
		hg.failure[bucket]++
		hg.failureCount++
	}
}

func sortBuckets(b map[time.Duration]int) []time.Duration {
	var i []int
	for bucket := range b {
		i = append(i, int(bucket))
	}

	sort.Ints(i)
	d := make([]time.Duration, len(i))
	for ii := range i {
		d[ii] = time.Duration(i[ii])
	}

	return d
}

func printBuckets(title string, b map[time.Duration]int) {
	if len(b) == 0 {
		fmt.Printf("%s: none.\n", strings.Title(title))
		return
	}

	buckets := sortBuckets(b)
	fmt.Printf("%s:\n", strings.Title(title))
	for i := range buckets {
		fmt.Printf("%v, %d\n", buckets[i], b[buckets[i]])
	}
}

func (hg *hg) printStats() {
	printBuckets("success", hg.success)
	printBuckets("failure", hg.failure)
}

// handleSigterm handles SIGTERM signal sent to the process.
func handleSigterm(server *http.Server, cancelFunc func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	log.Println("Received Term signal. Terminating...")
	server.Close()
	cancelFunc()
}

func setupServer(listenAddr string) *http.Server {
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.Handler())
	handler.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return &http.Server{
		Addr:    listenAddr,
		Handler: handler,
	}
}
