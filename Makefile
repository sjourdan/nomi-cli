# Default target builds the CLI
all: build

VERSION := $(shell git describe --tags --always --dirty)

# Build the CLI
build:
	go build -ldflags "-X main.Version=$(VERSION)" -o nomi-cli
	
# Install the CLI to $GOPATH/bin
install: build
	go install

# Clean build artifacts
clean:
	rm -f nomi-cli

# Install dependencies
deps:
	go mod download

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests and show coverage in terminal
test-coverage-text:
	go test -v -cover ./...

.PHONY: all build clean deps test test-coverage test-coverage-text install
