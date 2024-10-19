.PHONY: clean build.local build.linux build.docker build.push

BINARY        		 ?= go-dnsperf
VERSION       		 ?= $(shell git describe --tags --always --dirty)
IMAGE         		 ?= mikkeloscar/$(BINARY)
TAG           		 ?= $(VERSION)
SOURCES       		 = $(shell find . -name '*.go')
DOCKERFILE    		 ?= Dockerfile

default: build.local

clean:
	rm -rf build

build.local: build/$(BINARY)
build.linux: build/linux/$(BINARY)
build.linux.amd64: build/linux/amd64/$(BINARY)-godns
build.linux.arm64: build/linux/arm64/$(BINARY)-godns

build/$(BINARY): go.mod $(SOURCES)
	CGO_ENABLED=1 go build -o build/$(BINARY)
	CGO_ENABLED=0 go build -o build/$(BINARY)-godns

build/linux/amd64/$(BINARY)-godns: go.mod $(SOURCES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/linux/amd64/$(BINARY)-godns

build/linux/arm64/$(BINARY)-godns: go.mod $(SOURCES)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o build/linux/arm64/$(BINARY)-godns

build.docker: build.linux
	docker build --rm -t "$(IMAGE):$(TAG)" -f $(DOCKERFILE) .
	docker build --rm -t "$(IMAGE)-alpine:$(TAG)" -f $(DOCKERFILE).alpine .
	docker build --rm -t "$(IMAGE)-godns:$(TAG)" -f $(DOCKERFILE).godns .

build.push: build.docker
	docker push "$(IMAGE):$(TAG)"
	docker push "$(IMAGE)-alpine:$(TAG)"
	docker push "$(IMAGE)-godns:$(TAG)"

build.push.multiarch: build.linux.amd64 build.linux.arm64
	docker buildx create --driver-opt network=host --bootstrap --use
	# docker buildx build --rm -t "$(IMAGE):$(TAG)" --platform linux/amd64,linux/arm64 --push .
	docker buildx build --rm -t "$(IMAGE)-alpine:$(TAG)" --platform linux/amd64,linux/arm64 --push -f $(DOCKERFILE).alpine .
	docker buildx build --rm -t "$(IMAGE)-godns:$(TAG)" --platform linux/amd64,linux/arm64 --push -f $(DOCKERFILE).godns .
