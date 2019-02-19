package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	rps     int
	names   string
	logging bool
	timeout time.Duration
)

func main() {
	flag.IntVar(&rps, "rps", 10, "DNS loopups per second.")
	flag.StringVar(&names, "names", "google.com", "Comma separated list of hostnames")
	flag.BoolVar(&logging, "enable-logging", true, "Whether to enable logging or not")
	flag.DurationVar(&timeout, "timeout", 1*time.Second, "Timeout for DNS queries")
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
	}

	for {
		select {
		case <-ticker.C:
			go dnsResolver.Lookup(hostNames[i], timeout)
		case <-sigs:
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
}

func (r *CustomResolver) Lookup(name string, timeout time.Duration) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	names, err := r.resolver.LookupHost(ctx, name)
	if err != nil {
		if r.logging {
			log.Printf("[ERROR] %s", err)
		}
		return
	}
	if r.logging {
		log.Printf("%s - %s", name, names)
	}
}
