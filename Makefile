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

build/$(BINARY): go.mod $(SOURCES)
	CGO_ENABLED=1 go build -o build/$(BINARY)
	CGO_ENABLED=0 go build -o build/$(BINARY)-godns

build/linux/$(BINARY): go.mod $(SOURCES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -o build/$(BINARY)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/$(BINARY)-godns

build.docker: build.linux
	docker build --rm -t "$(IMAGE):$(TAG)" -f $(DOCKERFILE) .
	docker build --rm -t "$(IMAGE)-alpine:$(TAG)" -f $(DOCKERFILE).alpine .
	docker build --rm -t "$(IMAGE)-godns:$(TAG)" -f $(DOCKERFILE).godns .

build.push: build.docker
	docker push "$(IMAGE):$(TAG)"
	docker push "$(IMAGE)-alpine:$(TAG)"
	docker push "$(IMAGE)-godns:$(TAG)"
