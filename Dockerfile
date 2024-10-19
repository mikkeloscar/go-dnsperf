FROM golang AS builder

# RUN apk add -U git gcc bind-dev musl-dev

WORKDIR /go-dnsperf

COPY . .
RUN CGO_ENABLED=1 go build

FROM ubuntu:latest

RUN apt-get update && apt-get install --yes \
    dnsutils \
    iputils-ping \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go-dnsperf/go-dnsperf /go-dnsperf

ENTRYPOINT ["/go-dnsperf"]
