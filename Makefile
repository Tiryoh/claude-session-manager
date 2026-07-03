.PHONY: build test vet install

build:
	go build -o bin/csm ./cmd/csm

test:
	go test ./...

vet:
	go vet ./...

install:
	go install ./cmd/csm
