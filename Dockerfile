FROM ubuntu:18.04

ADD go-dnsperf /

ENTRYPOINT ["/go-dnsperf"]
