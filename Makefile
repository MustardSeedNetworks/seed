.PHONY: all build build-frontend build-backend build-iperf3 clean test lint run dev help

# Variables
BINARY_NAME=netscope
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

# Default target
all: build

## Build

build: build-iperf3 build-frontend build-backend ## Build everything

build-iperf3: ## Build iperf3 from source
	@echo "Building iperf3..."
	@./scripts/build-iperf3.sh

build-backend: ## Build Go backend
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/netscope

build-frontend: ## Build React frontend
	cd web && npm ci && npm run build

build-linux-amd64: ## Build for Linux AMD64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/netscope

build-linux-arm64: ## Build for Linux ARM64 (Raspberry Pi)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/netscope

## Development

run: ## Run the application (requires sudo for packet capture)
	sudo go run ./cmd/netscope

dev: ## Run backend in development mode
	go run ./cmd/netscope -dev

dev-frontend: ## Run frontend in development mode
	cd web && npm run dev

## Testing

test: test-backend test-frontend ## Run all tests

test-backend: ## Run Go tests
	go test -race -coverprofile=coverage.out ./...

test-frontend: ## Run frontend tests
	cd web && npm test

test-coverage: ## Generate coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Linting

lint: lint-backend lint-frontend ## Run all linters

lint-backend: ## Run Go linter
	golangci-lint run

lint-frontend: ## Run frontend linter
	cd web && npm run lint

## Cleanup

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	rm -rf web/dist web/node_modules
	rm -rf build/iperf3 bin/iperf3*

## Dependencies

deps: ## Install dependencies
	go mod download
	cd web && npm ci

deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy
	cd web && npm update

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
