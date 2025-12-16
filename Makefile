# =============================================================================
# LuminetIQ Makefile
# =============================================================================
#
# Build, test, and deploy automation for LuminetIQ network diagnostics tool.
#
# Quick Start:
#   make build          - Build production binary (frontend + backend)
#   make dev            - Run in development mode (hot reload frontend)
#   make deploy         - Deploy to remote Ubuntu server
#   make test           - Run all tests
#   make help           - Show all available targets
#
# Requirements:
#   - Go 1.25.5+
#   - Node.js 25.2.1+ (for frontend)
#   - libpcap-dev (for packet capture)
#   - Docker (optional, for cross-compilation)
#
# Environment Variables:
#   DEPLOY_HOST  - Target server IP (default: 192.168.64.7)
#   DEPLOY_USER  - SSH user (default: krisarmstrong)
#   DEPLOY_PATH  - Remote installation path (default: /home/$USER/luminetiq)
#   DEPLOY_PORT  - HTTPS port for smoke tests (default: 8443)
#   REBUILD      - Set to 1 to force rebuild during deploy
#
# =============================================================================

.PHONY: all build build-frontend frontend-deps build-backend build-backend-dev \
        build-iperf3 build-iperf3-linux build-iperf3-linux-amd64 build-iperf3-linux-arm64 build-iperf3-all \
        build-linux-amd64 build-linux-arm64 build-linux-docker \
        docker docker-build docker-test docker-push \
        clean clean-all test test-backend test-frontend test-coverage test-integration test-e2e \
        lint lint-backend lint-frontend fmt fmt-frontend fmt-all \
        security security-backend security-frontend security-secrets \
        run dev dev-frontend \
        deploy smoke-test smoke-test-local \
        deps deps-update logs logs-100 help \
        verify release-check iso-info pre-commit

# =============================================================================
# Configuration Variables
# =============================================================================

# Application name - used for binary output and process management
BINARY_NAME=luminetiq

# Version information extracted from git
# Falls back to "dev" if not in a git repository
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go build flags for reproducibility (closes #465, #550)
# -trimpath: Remove file system paths from compiled binary
# -buildvcs=false: Disable VCS stamping for deterministic builds
GOFLAGS=-trimpath -buildvcs=false

# Linker flags to embed version info and strip debug symbols
# -s: Omit symbol table and debug info
# -w: Omit DWARF symbol table
# -X: Set package variables at link time
LDFLAGS=-s -w \
    -X github.com/krisarmstrong/luminetiq/internal/version.Version=$(VERSION) \
    -X github.com/krisarmstrong/luminetiq/internal/version.Commit=$(COMMIT) \
    -X github.com/krisarmstrong/luminetiq/internal/version.BuildTime=$(BUILD_TIME)

# =============================================================================
# Deployment Configuration
# =============================================================================

# Remote server settings (override with environment variables)
DEPLOY_HOST?=192.168.64.7
DEPLOY_USER?=krisarmstrong
DEPLOY_PATH?=/home/$(DEPLOY_USER)/luminetiq
DEPLOY_PORT?=8443

# Docker image settings
DOCKER_IMAGE?=luminetiq
DOCKER_TAG?=$(VERSION)

# =============================================================================
# Default Target
# =============================================================================

all: verify ## Full build and validation (recommended before release)

# =============================================================================
# Build Targets
# =============================================================================

# Build complete application (frontend embedded in Go binary)
build: build-frontend build-backend ## Build everything (frontend embedded in binary)
	@echo "Build complete: $(BINARY_NAME) ($(VERSION))"

# -----------------------------------------------------------------------------
# iperf3 Builds - Network performance testing tool
# -----------------------------------------------------------------------------

build-iperf3: ## Build iperf3 from source (native platform)
	@echo "Building iperf3 for native platform..."
	@./scripts/build-iperf3.sh

build-iperf3-linux: ## Build iperf3 for Linux (AMD64 + ARM64 via Docker)
	@echo "Cross-compiling iperf3 for Linux (all architectures)..."
	@./scripts/build-iperf3-linux.sh all

build-iperf3-linux-amd64: ## Build iperf3 for Linux AMD64 only
	@./scripts/build-iperf3-linux.sh amd64

build-iperf3-linux-arm64: ## Build iperf3 for Linux ARM64 only
	@./scripts/build-iperf3-linux.sh arm64

build-iperf3-all: build-iperf3 build-iperf3-linux ## Build iperf3 for all platforms

# -----------------------------------------------------------------------------
# Backend Builds
# -----------------------------------------------------------------------------

# Production build with embedded frontend assets
# CGO is required for libpcap (packet capture) and ethtool bindings
build-backend: ## Build Go backend with embedded frontend
	@echo "🔨 Building backend..."
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/luminetiq
	@echo "✅ Backend build complete: $(BINARY_NAME)"

# Development build that reads frontend from disk instead of embedding
# Allows frontend hot-reload without rebuilding Go binary
build-backend-dev: ## Build Go backend in dev mode (reads frontend from disk)
	CGO_ENABLED=1 go build -tags dev $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/luminetiq

# -----------------------------------------------------------------------------
# Frontend Build (closes #464, #467)
# -----------------------------------------------------------------------------

# Install frontend dependencies (separate from build for caching)
# Only runs npm ci if node_modules is missing or package-lock.json changed
frontend-deps: ## Install frontend dependencies (cached)
	@if [ ! -d web/node_modules ] || [ web/package-lock.json -nt web/node_modules/.package-lock.json ]; then \
		echo "Installing frontend dependencies..."; \
		cd web && npm ci; \
	else \
		echo "Frontend dependencies up to date"; \
	fi

# Build React/TypeScript frontend
# Output goes to web/dist/ which is embedded in the Go binary
build-frontend: frontend-deps ## Build React frontend
	@echo "🔨 Building frontend..."
	@cd web && npm run build
	@echo "✅ Frontend build complete"

# -----------------------------------------------------------------------------
# Cross-Compilation for Linux
# -----------------------------------------------------------------------------

# Native cross-compilation (requires cross-compiler toolchain)
build-linux-amd64: build-frontend ## Build for Linux AMD64 (requires cross-compiler)
	@echo "Building for Linux AMD64..."
	@echo "Note: Requires CGO. Run on Linux or use 'make build-linux-docker'"
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-gnu-gcc \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 ./cmd/luminetiq

# ARM64 build for Raspberry Pi and ARM servers
build-linux-arm64: build-frontend ## Build for Linux ARM64 (Raspberry Pi, ARM servers)
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-arm64 ./cmd/luminetiq

# Docker-based cross-compilation (works from any platform)
# Uses official Go image with libpcap installed
build-linux-docker: build-frontend ## Build for Linux AMD64 using Docker (cross-platform)
	@echo "Building for Linux AMD64 using Docker..."
	docker run --rm -v "$(PWD):/build" -w /build golang:1.25.5 bash -c "\
		apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
		go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-amd64 ./cmd/luminetiq"
	@echo "Built: $(BINARY_NAME)-linux-amd64"

# =============================================================================
# Docker Targets
# =============================================================================

# Build Docker image (multi-stage build)
docker-build: ## Build Docker image
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

# Test Docker image starts correctly
docker-test: docker-build ## Test Docker image starts and responds
	@echo "Testing Docker image..."
	@docker rm -f luminetiq-test 2>/dev/null || true
	@docker run -d --name luminetiq-test -p 18443:8443 $(DOCKER_IMAGE):$(DOCKER_TAG)
	@sleep 5
	@curl -sf -k "https://localhost:18443/api/health" > /dev/null && \
		echo "PASS: Docker container health check" || \
		(echo "FAIL: Container not responding" && docker logs luminetiq-test && docker rm -f luminetiq-test && exit 1)
	@docker rm -f luminetiq-test > /dev/null 2>&1

# Full Docker target
docker: docker-test ## Build and test Docker image

# Push to registry (requires DOCKER_REGISTRY env var)
docker-push: docker-build ## Push Docker image to registry
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "Error: DOCKER_REGISTRY not set"; \
		exit 1; \
	fi
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker tag $(DOCKER_IMAGE):latest $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest

# =============================================================================
# Deployment
# =============================================================================

# Full deployment pipeline:
# 1. Build Linux binary (native or Docker)
# 2. Rsync binary, iperf3, and configs to remote
# 3. Set capabilities and restart service
# 4. Run smoke tests to verify deployment
deploy: ## Deploy to Ubuntu server (builds via Docker if needed)
	@# Build Linux binary - try direct first, fallback to Docker
	@if [ ! -f $(BINARY_NAME)-linux-amd64 ] || [ "$(REBUILD)" = "1" ]; then \
		if command -v x86_64-linux-gnu-gcc > /dev/null 2>&1; then \
			$(MAKE) build-linux-amd64; \
		else \
			echo "Cross-compiler not found, using Docker..."; \
			$(MAKE) build-linux-docker; \
		fi; \
	fi
	@echo "Deploying to $(DEPLOY_USER)@$(DEPLOY_HOST):$(DEPLOY_PATH)..."
	@# Check for iperf3-linux-amd64
	@if [ ! -f bin/iperf3-linux-amd64 ]; then \
		echo "Warning: bin/iperf3-linux-amd64 not found."; \
		echo "Run 'make build-iperf3-linux' first for iperf3 support."; \
	fi
	@# Create remote directory if needed
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "mkdir -p $(DEPLOY_PATH)/bin"
	@# Rsync binary
	rsync -avz --progress $(BINARY_NAME)-linux-amd64 $(DEPLOY_USER)@$(DEPLOY_HOST):$(DEPLOY_PATH)/$(BINARY_NAME)
	@# Rsync iperf3 if it exists
	@if [ -f bin/iperf3-linux-amd64 ]; then \
		rsync -avz --progress bin/iperf3-linux-amd64 $(DEPLOY_USER)@$(DEPLOY_HOST):$(DEPLOY_PATH)/bin/iperf3; \
	fi
	@# Rsync configs if they exist
	@if [ -d configs ]; then \
		rsync -avz --progress configs/ $(DEPLOY_USER)@$(DEPLOY_HOST):$(DEPLOY_PATH)/configs/; \
	fi
	@# Set capabilities and restart
	@# cap_net_raw is required for ICMP ping and packet capture (LLDP/CDP)
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "\
		cd $(DEPLOY_PATH) && \
		chmod +x $(BINARY_NAME) && \
		if [ -f bin/iperf3 ]; then chmod +x bin/iperf3; fi && \
		pkill -9 $(BINARY_NAME) 2>/dev/null || true && \
		sleep 1 && \
		sudo setcap cap_net_raw=+ep ./$(BINARY_NAME) && \
		nohup ./$(BINARY_NAME) > $(BINARY_NAME).log 2>&1 & \
		sleep 3 && \
		ps aux | grep '[l]uminetiq' || echo 'Warning: Server may not have started'"
	@echo ""
	@echo "Deployment complete. Running smoke tests..."
	@$(MAKE) smoke-test

# -----------------------------------------------------------------------------
# Smoke Tests - Quick verification of deployed application
# -----------------------------------------------------------------------------

# Verify deployed server is operational
smoke-test: ## Run smoke tests against deployed server
	@echo ""
	@echo "=== Smoke Tests for $(DEPLOY_HOST):$(DEPLOY_PORT) ==="
	@echo ""
	@echo "1. Checking server process..."
	@ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "pgrep -x $(BINARY_NAME) > /dev/null" && \
		echo "   PASS: Server process running" || \
		(echo "   FAIL: Server process not found" && exit 1)
	@echo ""
	@echo "2. Checking API /api/status..."
	@curl -sf -k "https://$(DEPLOY_HOST):$(DEPLOY_PORT)/api/status" > /dev/null && \
		echo "   PASS: API responding" || \
		(echo "   FAIL: API not responding" && exit 1)
	@echo ""
	@echo "3. Checking frontend loads..."
	@curl -sf -k "https://$(DEPLOY_HOST):$(DEPLOY_PORT)/" | grep -q "<!DOCTYPE" && \
		echo "   PASS: Frontend HTML served" || \
		(echo "   FAIL: Frontend not loading" && exit 1)
	@echo ""
	@echo "4. Checking iperf3 availability..."
	@curl -sf -k "https://$(DEPLOY_HOST):$(DEPLOY_PORT)/api/iperf/info" | grep -q '"installed":true' && \
		echo "   PASS: iperf3 available" || \
		echo "   WARN: iperf3 not available (optional)"
	@echo ""
	@echo "=== All smoke tests passed! ==="
	@echo "Server running at: https://$(DEPLOY_HOST):$(DEPLOY_PORT)"

# Local smoke tests for development
smoke-test-local: ## Run smoke tests against local server (port 8443)
	@echo "Running local smoke tests..."
	@echo "1. API /api/status..."
	@curl -sf -k "https://localhost:8443/api/status" > /dev/null && \
		echo "   PASS" || (echo "   FAIL" && exit 1)
	@echo "2. Frontend loads..."
	@curl -sf -k "https://localhost:8443/" | grep -q "<!DOCTYPE" && \
		echo "   PASS" || (echo "   FAIL" && exit 1)
	@echo "All local tests passed!"

# =============================================================================
# Development
# =============================================================================

# Run with elevated privileges (required for raw socket operations)
run: ## Run the application (requires sudo for packet capture)
	sudo go run ./cmd/luminetiq

# Development mode: backend reads frontend from disk
# Run alongside 'make dev-frontend' for full hot-reload experience
dev: ## Run backend in development mode (reads frontend from disk)
	go run -tags dev ./cmd/luminetiq

# Vite dev server with hot module replacement
# Proxies API requests to backend on :8443
dev-frontend: ## Run frontend in development mode
	cd web && npm run dev

# =============================================================================
# Testing
# =============================================================================

# Run complete test suite
test: test-backend test-frontend ## Run all tests
	@echo "✅ All tests complete"

# Go tests with race detection and coverage
test-backend: ## Run Go tests
	@echo "🧪 Running backend tests..."
	@go test -race -coverprofile=coverage.out ./...
	@echo "✅ Backend tests complete"

# Frontend unit tests via Vitest
test-frontend: ## Run frontend tests
	@echo "🧪 Running frontend tests..."
	@cd web && npm test
	@echo "✅ Frontend tests complete"

# Frontend E2E tests via Playwright (closes #482, #309)
# Requires backend to be running on port 8443
test-e2e: ## Run frontend E2E tests (requires backend running)
	cd web && npm run test:e2e

# Run E2E tests with UI mode for debugging
test-e2e-ui: ## Run E2E tests with Playwright UI
	cd web && npm run test:e2e:ui

# Install Playwright browsers (required before first E2E run)
test-e2e-install: ## Install Playwright browsers
	cd web && npx playwright install --with-deps chromium

# Generate HTML coverage report
test-coverage: ## Generate coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Full integration test: deploy, install as systemd service, run tests
test-integration: build-linux-docker ## Full integration test on Ubuntu server via systemd
	@echo "Running integration tests on $(DEPLOY_HOST)..."
	rsync -avz $(BINARY_NAME)-linux-amd64 $(DEPLOY_USER)@$(DEPLOY_HOST):/tmp/luminetiq-test
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "\
		sudo systemctl stop luminetiq-test 2>/dev/null || true && \
		sudo cp /tmp/luminetiq-test /usr/local/bin/luminetiq-test && \
		sudo chmod +x /usr/local/bin/luminetiq-test && \
		sudo setcap cap_net_raw=+ep /usr/local/bin/luminetiq-test && \
		echo '[Unit]' | sudo tee /etc/systemd/system/luminetiq-test.service > /dev/null && \
		echo 'Description=LuminetIQ Integration Test' | sudo tee -a /etc/systemd/system/luminetiq-test.service > /dev/null && \
		echo '[Service]' | sudo tee -a /etc/systemd/system/luminetiq-test.service > /dev/null && \
		echo 'Type=simple' | sudo tee -a /etc/systemd/system/luminetiq-test.service > /dev/null && \
		echo 'ExecStart=/usr/local/bin/luminetiq-test' | sudo tee -a /etc/systemd/system/luminetiq-test.service > /dev/null && \
		echo 'WorkingDirectory=/tmp' | sudo tee -a /etc/systemd/system/luminetiq-test.service > /dev/null && \
		echo '[Install]' | sudo tee -a /etc/systemd/system/luminetiq-test.service > /dev/null && \
		echo 'WantedBy=multi-user.target' | sudo tee -a /etc/systemd/system/luminetiq-test.service > /dev/null && \
		sudo systemctl daemon-reload && \
		sudo systemctl start luminetiq-test && \
		sleep 5 && \
		sudo systemctl is-active luminetiq-test && \
		curl -sf -k https://localhost:8443/api/health && \
		echo 'PASS: Integration test passed' && \
		sudo systemctl stop luminetiq-test && \
		sudo rm -f /etc/systemd/system/luminetiq-test.service /usr/local/bin/luminetiq-test"
	@echo "Integration tests completed successfully"

# =============================================================================
# Linting & Formatting
# =============================================================================

# Run all linters
lint: lint-backend lint-frontend ## Run all linters
	@echo "✅ All linters complete"

# golangci-lint with project configuration (.golangci.yml)
lint-backend: ## Run Go linter
	@echo "🔍 Running backend linter..."
	@golangci-lint run
	@echo "✅ Backend lint complete"

# ESLint with TypeScript rules
lint-frontend: ## Run frontend linter
	@echo "🔍 Running frontend linter..."
	@cd web && npm run lint
	@echo "✅ Frontend lint complete"

# Format Go code
fmt: ## Format Go code with gofmt
	gofmt -w -s .
	@echo "Go code formatted"

# Format frontend code
fmt-frontend: ## Format frontend code with Prettier
	cd web && npx prettier --write .
	@echo "Frontend code formatted"

# Format all code
fmt-all: fmt fmt-frontend ## Format all code (Go + frontend)

# =============================================================================
# Security Scanning
# =============================================================================

# Run all security scans
security: security-backend security-frontend security-secrets ## Run all security scans

# Go security scanning (gosec + govulncheck)
security-backend: ## Run Go security scans (gosec, govulncheck)
	@echo "Running Go security scans..."
	@if command -v gosec > /dev/null 2>&1; then \
		gosec -quiet ./...; \
	else \
		echo "SKIP: gosec not installed (go install github.com/securego/gosec/v2/cmd/gosec@latest)"; \
	fi
	@if command -v govulncheck > /dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "SKIP: govulncheck not installed (go install golang.org/x/vuln/cmd/govulncheck@latest)"; \
	fi
	@echo "Go security scans complete"

# Frontend security scanning (npm audit)
security-frontend: ## Run frontend security scan (npm audit)
	@echo "Running npm audit..."
	cd web && npm audit --audit-level=high
	@echo "npm audit complete"

# Secret scanning (gitleaks)
security-secrets: ## Scan for secrets in codebase (gitleaks)
	@echo "Running gitleaks..."
	@if command -v gitleaks > /dev/null 2>&1; then \
		gitleaks detect --source . --config .gitleaks.toml --verbose; \
	else \
		echo "SKIP: gitleaks not installed (brew install gitleaks or go install github.com/gitleaks/gitleaks/v8@latest)"; \
	fi
	@echo "Secret scan complete"

# Trivy filesystem scan (if available)
security-trivy: ## Run Trivy vulnerability scan
	@echo "Running Trivy filesystem scan..."
	@if command -v trivy > /dev/null 2>&1; then \
		trivy fs --severity HIGH,CRITICAL .; \
	else \
		echo "SKIP: trivy not installed (brew install trivy)"; \
	fi

# =============================================================================
# Cleanup
# =============================================================================

# Remove build artifacts
clean: ## Clean build artifacts
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	rm -rf web/dist

# Full clean including dependencies
clean-all: clean ## Clean everything including dependencies
	rm -rf web/node_modules
	rm -rf build/iperf3 bin/iperf3*

# =============================================================================
# Dependencies
# =============================================================================

# Install all dependencies (Go modules + npm packages)
deps: ## Install dependencies
	go mod download
	cd web && npm ci

# Update dependencies to latest versions
deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy
	cd web && npm update

# =============================================================================
# Remote Logs
# =============================================================================

# Stream live logs from deployed server
logs: ## Tail server logs on remote
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "tail -f $(DEPLOY_PATH)/$(BINARY_NAME).log"

# View recent log entries
logs-100: ## Show last 100 lines of remote server logs
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "tail -100 $(DEPLOY_PATH)/$(BINARY_NAME).log"

# =============================================================================
# Help
# =============================================================================

# Auto-generate help from ## comments
help: ## Show this help
	@echo "LuminetIQ - Network Diagnostics Tool"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build                    Build production binary"
	@echo "  make dev & make dev-frontend  Full development environment"
	@echo "  make deploy REBUILD=1         Force rebuild and deploy"
	@echo "  make deploy DEPLOY_HOST=x.x.x.x  Deploy to different server"

# =============================================================================
# Verification & Release Checks
# =============================================================================

# Full verification: builds, tests, lints, security scans, and validates everything
# This is what 'make all' runs - use before releases
# Docker tests are skipped if Docker is not available (e.g., on macOS without Docker)
verify: lint test security build ## Full verification (lint, test, security, build, docker if available)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		$(MAKE) docker-test; \
	else \
		echo "SKIP: Docker not available - skipping docker-test"; \
		echo "      Run 'make docker-test' manually when Docker is running"; \
	fi
	@echo ""
	@echo "=============================================="
	@echo "  VERIFICATION COMPLETE"
	@echo "=============================================="
	@echo "  Version:     $(VERSION)"
	@echo "  Commit:      $(COMMIT)"
	@echo "  Binary:      $(BINARY_NAME)"
	@echo "  Docker:      $(DOCKER_IMAGE):$(DOCKER_TAG) (if Docker available)"
	@echo "=============================================="
	@echo ""
	@echo "Ready for deployment. Run 'make deploy' to deploy."

# Pre-commit hook setup and manual run
pre-commit: ## Run pre-commit hooks manually
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "pre-commit not installed. Install with: pip install pre-commit"; \
		echo "Then run: pre-commit install"; \
		exit 1; \
	fi

# Install pre-commit hooks
pre-commit-install: ## Install pre-commit hooks
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit install; \
		pre-commit install --hook-type pre-push; \
		echo "Pre-commit hooks installed successfully"; \
	else \
		echo "pre-commit not installed. Install with: pip install pre-commit"; \
		exit 1; \
	fi

# =============================================================================
# ISO Creation (Manual Process)
# =============================================================================

# Custom Ubuntu ISO creation is a separate, manual process
iso-info: ## Show instructions for creating custom Ubuntu ISO
	@echo ""
	@echo "=== Custom Ubuntu ISO Creation ==="
	@echo ""
	@echo "Creating a custom Ubuntu ISO with LuminetIQ pre-installed requires:"
	@echo "  1. Ubuntu host system (or VM)"
	@echo "  2. cubic or live-build package"
	@echo "  3. Built Linux binary (luminetiq-linux-amd64)"
	@echo ""
	@echo "Steps:"
	@echo "  1. Install cubic: sudo apt install cubic"
	@echo "  2. Download Ubuntu Server ISO"
	@echo "  3. Run cubic and customize:"
	@echo "     - Copy luminetiq-linux-amd64 to /usr/local/bin/luminetiq"
	@echo "     - Copy deploy/systemd/luminetiq.service to /etc/systemd/system/"
	@echo "     - Enable service: systemctl enable luminetiq"
	@echo "  4. Generate ISO"

# =============================================================================
# Release Checks
# =============================================================================

# Pre-release checklist validation
release-check: verify ## Full release validation (verify + install script check)
	@echo "Running release checks..."
	@bash -n deploy/systemd/install.sh && echo "PASS: install.sh syntax valid"
	@bash -n deploy/systemd/uninstall.sh 2>/dev/null && echo "PASS: uninstall.sh syntax valid" || true
	@if echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+'; then \
		echo "PASS: Version format valid ($(VERSION))"; \
	else \
		echo "WARN: Version $(VERSION) doesn't match vX.Y.Z format"; \
	fi
	@if git diff --quiet && git diff --staged --quiet; then \
		echo "PASS: No uncommitted changes"; \
	else \
		echo "WARN: Uncommitted changes detected"; \
	fi
	@echo ""
	@echo "Release checks complete. Ready for 'git tag $(VERSION) && git push --tags'"
