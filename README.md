# go-dnsperf

Very trivial tool for stress testing Kubernetes DNS

## Build

```
# Ubuntu based
CGO_ENABLED=1 go build
docker build --rm -t mikkeloscar/go-dnsperf:latest .
docker push mikkeloscar/go-dnsperf:latest

# alpine based
docker build --rm -t mikkeloscar/go-dnsperf:alpine-latest -f Dockerfile.alpine .
docker push mikkeloscar/go-dnsperf:alpine-latest

# golang dns
CGO_ENABLED=0 go build
docker build --rm -t mikkeloscar/go-dnsperf:godns-latest -f Dockerfile.godns .
docker push mikkeloscar/go-dnsperf:godns-latest
```

### Notes

#### Run test on dedicated node

```sh
$ kubectl label nodes <node-name> dns-load-test=true
```

Exlcude coredns pod from node:

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        # ...
        # add this expression
        - key: dns-load-test
          operator: NotIn
          values:
          - "true"
```
