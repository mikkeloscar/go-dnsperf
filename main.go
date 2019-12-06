package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	rps      int
	names    string
	logging  bool
	timeout  time.Duration
	duration time.Duration
)

func main() {
	flag.IntVar(&rps, "rps", 10, "DNS lookups per second.")
	flag.StringVar(&names, "names", "google.com", "Comma separated list of hostnames")
	flag.BoolVar(&logging, "enable-logging", true, "Whether to enable logging or not")
	flag.DurationVar(&timeout, "timeout", 1*time.Second, "Timeout for DNS queries")
	flag.DurationVar(&duration, "duration", -1, "Duration of the test, defaults to forever")
	flag.Parse()

	hostNames := strings.Split(names, ",")

	log.Print(rps)

	ticker := time.NewTicker(1 * time.Second / time.Duration(rps))
	defer ticker.Stop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	i := 0

	dnsResolver := &CustomResolver{
		resolver: &net.Resolver{},
		logging:  logging,
		hg:       newHG(),
	}

	var done <-chan time.Time
	if duration > 0 {
		done = time.After(duration)
	}

	for {
		select {
		case <-ticker.C:
			go dnsResolver.Lookup(hostNames[i], timeout)
		case <-sigs:
			dnsResolver.hg.printStats()
			return
		case <-done:
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
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	start := time.Now()
	names, err := r.resolver.LookupHost(ctx, name)
	duration := time.Since(start)
	if err != nil {
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
