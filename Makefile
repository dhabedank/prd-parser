.PHONY: build test lint clean install

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o prd-parser .

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

lint:
	golangci-lint run

clean:
	rm -f prd-parser coverage.out

install: build
	cp prd-parser $(GOPATH)/bin/
