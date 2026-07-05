VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test vet fmt lint install

build:
	go build -ldflags "$(LDFLAGS)" -o bin/csm ./cmd/csm

test:
	go test ./...

vet:
	go vet ./...

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

lint:
	golangci-lint run ./...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/csm
