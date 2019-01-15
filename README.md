```
CGO_ENABLED=1 go build
docker build --rm -t mikkeloscar/go-dnsperf:latest .
docker push mikkeloscar/go-dnsperf:latest
```
