FROM registry.opensource.zalan.do/library/ubuntu-20.04:latest

RUN apt-get update && apt-get install --yes \
    dnsutils \
    iputils-ping \
    && rm -rf /var/lib/apt/lists/*

ADD build/go-dnsperf /

ENTRYPOINT ["/go-dnsperf"]
