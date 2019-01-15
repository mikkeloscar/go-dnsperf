package main

import (
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
	rps   int
	names string
)

func main() {
	flag.IntVar(&rps, "rps", 10, "DNS loopups per second.")
	flag.StringVar(&names, "names", "google.com", "Comma separated list of hostnames")
	flag.Parse()

	hostNames := strings.Split(names, ",")

	log.Print(rps)

	ticker := time.NewTicker(1 * time.Second / time.Duration(rps))
	defer ticker.Stop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	i := 0

	for {
		select {
		case <-ticker.C:
			go lookup(hostNames[i])
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

func lookup(name string) {
	names, err := net.LookupHost(name)
	if err != nil {
		log.Printf("[ERROR] %s", err)
		return
	}
	log.Printf("%s - %s", name, names)
}
