.PHONY: all build test vet run bench web-install web-build web-test

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

web-install:
	cd web && pnpm install

web-build:
	cd web && pnpm run build

web-test:
	cd web && pnpm test
