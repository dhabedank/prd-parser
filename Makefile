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

# Install to /usr/local/bin (uses sudo if needed)
install: build
	@if [ -w /usr/local/bin ]; then \
		cp prd-parser /usr/local/bin/; \
		echo "Installed to /usr/local/bin/prd-parser"; \
	elif command -v sudo >/dev/null 2>&1; then \
		sudo cp prd-parser /usr/local/bin/; \
		echo "Installed to /usr/local/bin/prd-parser"; \
	else \
		mkdir -p ~/go/bin; \
		cp prd-parser ~/go/bin/; \
		echo "Installed to ~/go/bin/prd-parser"; \
		echo "Run: echo 'export PATH=\"\$$HOME/go/bin:\$$PATH\"' >> ~/.zshrc && source ~/.zshrc"; \
	fi
