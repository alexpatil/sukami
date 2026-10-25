.PHONY: all build test vet run bench

all: vet test build

build:
	go build -o bin/sukami ./cmd/sukami

bench:
	go run ./cmd/sukami-bench

test:
	go test -v ./pkg/...

vet:
	go vet ./...

run:
	go run ./cmd/sukami
