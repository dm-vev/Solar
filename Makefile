.SHELL := /bin/bash
.PHONY: all build test test-race vet lint fmt ci clean docker

all: build

build:
	go build -o bin/solar ./cmd/solar

test:
	go test -count=1 ./internal/...

test-race:
	go test -race -count=1 ./internal/...

vet:
	go vet ./internal/... ./cmd/...

lint:
	golangci-lint run --timeout 5m ./internal/... ./cmd/...

fmt:
	gofmt -w ./internal/... ./cmd/...
	goimports -w -local github.com/solar-mc/solar ./internal/... ./cmd/...

ci: vet test-race lint build

clean:
	rm -rf bin/

docker:
	docker build -t solar-mc/solar:latest .
