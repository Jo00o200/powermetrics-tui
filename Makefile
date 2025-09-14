# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=powermetrics-tui
BINARY_UNIX=$(BINARY_NAME)_unix

# Build info
VERSION?=$(shell git describe --tags --always --dirty)
BUILD_TIME?=$(shell date +%Y-%m-%d_%H:%M:%S)
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test coverage deps help

all: test build

build: ## Build the application
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_NAME) .

build-prod: ## Build for production (same as build but explicit)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_NAME) .

build-all: build-linux build-darwin build-windows ## Build for multiple platforms

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_NAME)-linux-arm64 .

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_NAME)-darwin-arm64 .

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -trimpath -o $(BINARY_NAME)-windows-amd64.exe .

test: ## Run tests
	$(GOTEST) -v ./...

coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

deps: ## Download dependencies
	$(GOGET) -d ./...
	$(GOCMD) mod tidy

build-dev: ## Development build (with debug symbols)
	$(GOBUILD) -race -o $(BINARY_NAME)-dev .

install: ## Install the application
	CGO_ENABLED=0 $(GOCMD) install $(LDFLAGS) -trimpath .

run: ## Run the application
	$(GOBUILD) -o $(BINARY_NAME) . && ./$(BINARY_NAME)

## Show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)