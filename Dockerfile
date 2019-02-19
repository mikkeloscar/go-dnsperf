FROM ubuntu:18.04

RUN apt-get update && apt-get install --yes \
    dnsutils \
    iputils-ping \
    && rm -rf /var/lib/apt/lists/*

ADD go-dnsperf /

ENTRYPOINT ["/go-dnsperf"]
