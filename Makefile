# =============================================================================
# Seed Makefile
# =============================================================================
#
# Build, test, and deploy automation for Seed network diagnostics tool.
#
# QUICK START
# -----------
#   make build          Build production binary (frontend + backend)
#   make test           Run all unit tests (backend + frontend)
#   make verify         Full CI pipeline (lint, test, security, build)
#   make dev            Run backend in dev mode (hot reload frontend)
#   make deploy         Deploy to remote Ubuntu server
#   make install        Install on current system (macOS/Linux)
#   make help           Show all available targets
#
# COMMON WORKFLOWS
# ----------------
#   Development:        make dev & make dev-frontend
#   Before commit:      make verify
#   Release:            make release VERSION=v1.0.0
#   Cross-compile:      make build-all
#
# REQUIREMENTS
# ------------
#   - Go 1.25.5+ (with CGO for libpcap)
#   - Node.js 25.2.1+ and npm
#   - libpcap-dev (Linux) or libpcap (macOS via Homebrew)
#   - Docker (optional, for cross-compilation and container builds)
#
# ENVIRONMENT VARIABLES
# ---------------------
#   DEPLOY_HOST     Target server IP (default: 192.168.64.7)
#   DEPLOY_USER     SSH user (default: krisarmstrong)
#   DEPLOY_PATH     Remote path (default: /home/$USER/seed)
#   DEPLOY_PORT     HTTPS port for smoke tests (default: 443, fallback: 8443)
#   REBUILD         Set to 1 to force rebuild during deploy
#   DOCKER_REGISTRY Container registry for docker-push
#
# PORT CONFIGURATION
# ------------------
#   The application prefers port 443 (standard HTTPS) if available.
#   If 443 requires root/elevated permissions, it falls back to 8443.
#   Set DEPLOY_PORT to override for smoke tests.
#
# =============================================================================

.PHONY: all build build-frontend frontend-deps generate-types build-backend build-backend-dev \
        build-darwin build-all \
        build-iperf3 build-iperf3-linux build-iperf3-linux-amd64 build-iperf3-linux-arm64 build-iperf3-all \
        build-linux-amd64 build-linux-arm64 build-linux-docker \
        docker docker-build docker-test docker-push \
        clean clean-all \
        test test-all test-backend test-frontend test-coverage test-integration \
        test-e2e test-e2e-ui test-e2e-install \
        lint lint-backend lint-frontend lint-md \
        fmt fmt-frontend fmt-all fmt-md fmt-check \
        fix fix-backend fix-frontend fix-md fix-all \
        security security-backend security-frontend security-secrets security-trivy \
        storybook build-storybook test-storybook \
        run dev dev-frontend \
        install uninstall status \
        deploy smoke-test smoke-test-local \
        deps update update-go update-npm outdated \
        tools tools-go tools-frontend version version-check \
        logs logs-100 help \
        verify release release-check pre-commit pre-commit-install \
        license-check license-report \
        i18n-sync i18n-check i18n-list \
        iso-info

# =============================================================================
# Configuration Variables
# =============================================================================

# Application name - used for binary output and process management
BINARY_NAME=seed

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
    -X github.com/krisarmstrong/seed/internal/version.Version=$(VERSION) \
    -X github.com/krisarmstrong/seed/internal/version.Commit=$(COMMIT) \
    -X github.com/krisarmstrong/seed/internal/version.BuildTime=$(BUILD_TIME)

# =============================================================================
# Deployment Configuration
# =============================================================================

# Remote server settings (override with environment variables)
DEPLOY_HOST?=192.168.64.7
DEPLOY_USER?=krisarmstrong
DEPLOY_PATH?=/home/$(DEPLOY_USER)/seed
DEPLOY_PORT?=443

# Docker image settings
DOCKER_IMAGE?=seed
DOCKER_TAG?=$(VERSION)

# =============================================================================
# Progress Display Helpers
# =============================================================================

# ANSI color codes for terminal output
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
BOLD := \033[1m
RESET := \033[0m

# Progress step display: $(call step,current,total,message)
define step
	@printf "\n$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)\n"
	@printf "$(BOLD)$(CYAN)[$(1)/$(2)]$(RESET) $(BOLD)$(3)$(RESET)\n"
	@printf "$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)\n"
endef

# Success message
define success
	@printf "$(GREEN)✓ $(1) complete$(RESET)\n"
endef

# Timer: Start a named timer
# Usage: $(call timer-start,name)
define timer-start
	@date +%s > /tmp/make-timer-$(1)
endef

# Timer: Show elapsed time for a named timer
# Usage: $(call timer-end,name,description)
define timer-end
	@if [ -f /tmp/make-timer-$(1) ]; then \
		START=$$(cat /tmp/make-timer-$(1)); \
		END=$$(date +%s); \
		ELAPSED=$$((END - START)); \
		MINS=$$((ELAPSED / 60)); \
		SECS=$$((ELAPSED % 60)); \
		if [ $$MINS -gt 0 ]; then \
			printf "$(GREEN)✓ $(2) complete $(YELLOW)($$MINS min $$SECS sec)$(RESET)\n"; \
		else \
			printf "$(GREEN)✓ $(2) complete $(YELLOW)($$SECS sec)$(RESET)\n"; \
		fi; \
		rm -f /tmp/make-timer-$(1); \
	fi
endef

# =============================================================================
# Default Target
# =============================================================================

all: verify ## Full build and validation (recommended before release)

# =============================================================================
# Build Targets
# =============================================================================

# Build complete application (frontend embedded in Go binary)
build: ## Build everything (frontend embedded in binary)
	@printf "$(BOLD)$(CYAN)┌─ Building Application ───────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Frontend (React/TypeScript)                                           $(CYAN)│$(RESET)\n"
	$(call timer-start,build-frontend)
	@$(MAKE) --no-print-directory build-frontend-quiet
	$(call timer-end,build-frontend,Frontend build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) Backend (Go)                                                          $(CYAN)│$(RESET)\n"
	$(call timer-start,build-backend)
	@$(MAKE) --no-print-directory build-backend-quiet
	$(call timer-end,build-backend,Backend build)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "$(GREEN)✓ Build complete: $(BINARY_NAME) ($(VERSION))$(RESET)\n"

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
	@printf "$(BOLD)🔨 Building backend...$(RESET)\n"
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/seed
	@printf "$(GREEN)✓ Backend build complete: $(BINARY_NAME)$(RESET)\n"

# Backend build (quiet mode)
build-backend-quiet:
	@printf "   Compiling Go binary...\n"
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/seed
	@SIZE=$$(ls -lh $(BINARY_NAME) | awk '{print $$5}'); \
	printf "   Output: $(BINARY_NAME) ($$SIZE)\n"

# Development build that reads frontend from disk instead of embedding
# Allows frontend hot-reload without rebuilding Go binary
build-backend-dev: ## Build Go backend in dev mode (reads frontend from disk)
	CGO_ENABLED=1 go build -tags dev $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/seed

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

# Generate TypeScript types from JSON Schema
# Ensures frontend types match backend config structure
generate-types: frontend-deps ## Generate TypeScript types from JSON Schema
	@echo "🔧 Generating TypeScript types from schema..."
	@./scripts/generate-types.sh
	@echo "✅ TypeScript types generated"

# Build React/TypeScript frontend
# Output goes to web/dist/ which is embedded in the Go binary
build-frontend: frontend-deps ## Build React frontend
	@printf "$(BOLD)🔨 Building frontend...$(RESET)\n"
	@cd web && npm run build
	@printf "$(GREEN)✓ Frontend build complete$(RESET)\n"

# Frontend build (quiet mode)
build-frontend-quiet: frontend-deps
	@printf "   Bundling React application...\n"
	@cd web && npm run build 2>&1 | tail -3
	@SIZE=$$(du -sh web/dist 2>/dev/null | cut -f1 || echo "unknown"); \
	printf "   Output: web/dist ($$SIZE)\n"

# -----------------------------------------------------------------------------
# Cross-Compilation for Linux
# -----------------------------------------------------------------------------

# Native cross-compilation (requires cross-compiler toolchain)
build-linux-amd64: build-frontend ## Build for Linux AMD64 (requires cross-compiler)
	@echo "Building for Linux AMD64..."
	@echo "Note: Requires CGO. Run on Linux or use 'make build-linux-docker'"
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-gnu-gcc \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 ./cmd/seed

# ARM64 build for Raspberry Pi and ARM servers
build-linux-arm64: build-frontend ## Build for Linux ARM64 (Raspberry Pi, ARM servers)
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-arm64 ./cmd/seed

# Docker-based cross-compilation (works from any platform)
# Uses official Go image with libpcap installed
build-linux-docker: build-frontend ## Build for Linux AMD64 using Docker (cross-platform)
	@echo "Building for Linux AMD64 using Docker..."
	docker run --rm -v "$(PWD):/build" -w /build golang:1.21 bash -c "\
		apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
		go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-amd64 ./cmd/seed"
	@echo "Built: $(BINARY_NAME)-linux-amd64"

# -----------------------------------------------------------------------------
# macOS Builds
# -----------------------------------------------------------------------------

# Build for macOS (native, current architecture)
build-darwin: build-frontend ## Build for macOS (native architecture)
	@echo "Building for macOS ($(shell uname -m))..."
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-darwin-$(shell uname -m) ./cmd/seed
	@echo "Built: $(BINARY_NAME)-darwin-$(shell uname -m)"

# -----------------------------------------------------------------------------
# Multi-Platform Builds
# -----------------------------------------------------------------------------

# Build for all supported platforms
# Note: Cross-compilation requires appropriate toolchains or Docker
build-all: build-frontend ## Build for all platforms (native + Linux via Docker)
	@printf "$(BOLD)$(CYAN)┌─ Building All Platforms ────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/3]$(RESET) macOS (native)                                                       $(CYAN)│$(RESET)\n"
	$(call timer-start,build-darwin)
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-darwin-$(shell uname -m) ./cmd/seed
	$(call timer-end,build-darwin,macOS build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/3]$(RESET) Linux AMD64 (Docker)                                                 $(CYAN)│$(RESET)\n"
	$(call timer-start,build-linux-amd64)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		docker run --rm -v "$(PWD):/build" -w /build golang:1.21 bash -c "\
			apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
			go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-amd64 ./cmd/seed"; \
	else \
		printf "$(YELLOW)   SKIP: Docker not available$(RESET)\n"; \
	fi
	$(call timer-end,build-linux-amd64,Linux AMD64 build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/3]$(RESET) Linux ARM64 (Docker)                                                 $(CYAN)│$(RESET)\n"
	$(call timer-start,build-linux-arm64)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		docker run --rm -v "$(PWD):/build" -w /build --platform linux/arm64 golang:1.21 bash -c "\
			apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
			go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-arm64 ./cmd/seed" 2>/dev/null || \
		printf "$(YELLOW)   SKIP: ARM64 emulation not available$(RESET)\n"; \
	else \
		printf "$(YELLOW)   SKIP: Docker not available$(RESET)\n"; \
	fi
	$(call timer-end,build-linux-arm64,Linux ARM64 build)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "\n$(GREEN)✓ Multi-platform build complete$(RESET)\n"
	@ls -lh $(BINARY_NAME)-* 2>/dev/null || true

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
	@docker rm -f seed-test 2>/dev/null || true
	@docker run -d --name seed-test -p 18443:8443 $(DOCKER_IMAGE):$(DOCKER_TAG)
	@sleep 5
	@curl -sf -k "https://localhost:18443/api/health" > /dev/null && \
		echo "PASS: Docker container health check" || \
		(echo "FAIL: Container not responding" && docker logs seed-test && docker rm -f seed-test && exit 1)
	@docker rm -f seed-test > /dev/null 2>&1

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
# Deployment (closes #468)
# =============================================================================
#
# Deployment uses systemd for service management (recommended) or falls back
# to manual process management. Configure via environment variables:
#
#   DEPLOY_HOST     - Target server IP (default: 192.168.64.7)
#   DEPLOY_USER     - SSH user (default: krisarmstrong)
#   DEPLOY_PATH     - Remote installation path (default: /home/$USER/seed)
#   DEPLOY_PORT     - HTTPS port for smoke tests (default: 8443)
#   DEPLOY_SYSTEMD  - Use systemd (default: 1, set to 0 for manual mode)
#   REBUILD         - Force rebuild (default: 0)
#
# Prerequisites:
#   - SSH key authentication configured
#   - sudo access on remote host (for setcap and systemd)
#   - systemd service installed (run: sudo ./deploy/systemd/install.sh)
#
# =============================================================================

DEPLOY_SYSTEMD?=1

# Full deployment pipeline:
# 1. Build Linux binary (native or Docker)
# 2. Rsync binary, iperf3, and configs to remote
# 3. Restart service via systemd (or manual fallback)
# 4. Run smoke tests to verify deployment
deploy: ## Deploy to Ubuntu server (builds via Docker if needed)
	@# Verify SSH connectivity first
	@echo "Verifying SSH connectivity to $(DEPLOY_HOST)..."
	@ssh -o BatchMode=yes -o ConnectTimeout=5 $(DEPLOY_USER)@$(DEPLOY_HOST) "echo 'SSH OK'" || \
		(echo "ERROR: Cannot connect to $(DEPLOY_HOST). Check SSH configuration." && exit 1)
	@# Build Linux binary - try direct first, fallback to Docker
	@if [ ! -f $(BINARY_NAME)-linux-amd64 ] || [ "$(REBUILD)" = "1" ]; then \
		if command -v x86_64-linux-gnu-gcc > /dev/null 2>&1; then \
			$(MAKE) build-linux-amd64; \
		else \
			echo "Cross-compiler not found, using Docker..."; \
			$(MAKE) build-linux-docker; \
		fi; \
	fi
	@echo ""
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
	@# Set permissions and capabilities
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "\
		cd $(DEPLOY_PATH) && \
		chmod +x $(BINARY_NAME) && \
		if [ -f bin/iperf3 ]; then chmod +x bin/iperf3; fi && \
		sudo setcap cap_net_raw=+ep ./$(BINARY_NAME)"
	@# Restart service
	@if [ "$(DEPLOY_SYSTEMD)" = "1" ]; then \
		echo "Restarting via systemd..."; \
		ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "\
			if systemctl is-active --quiet $(BINARY_NAME) 2>/dev/null; then \
				sudo systemctl restart $(BINARY_NAME) && \
				sleep 2 && \
				sudo systemctl is-active --quiet $(BINARY_NAME) && \
				echo 'Service restarted successfully'; \
			else \
				echo 'systemd service not installed. Installing...'; \
				cd $(DEPLOY_PATH) && \
				if [ -f deploy/systemd/install.sh ]; then \
					sudo ./deploy/systemd/install.sh && \
					sudo systemctl start $(BINARY_NAME); \
				else \
					echo 'ERROR: Service not installed. Run: sudo ./deploy/systemd/install.sh'; \
					exit 1; \
				fi; \
			fi"; \
	else \
		echo "Restarting manually (DEPLOY_SYSTEMD=0)..."; \
		ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "\
			cd $(DEPLOY_PATH) && \
			pkill $(BINARY_NAME) 2>/dev/null || true && \
			sleep 2 && \
			nohup ./$(BINARY_NAME) > $(BINARY_NAME).log 2>&1 & \
			sleep 3 && \
			pgrep -x $(BINARY_NAME) > /dev/null && echo 'Server started' || echo 'WARNING: Server may not have started'"; \
	fi
	@echo ""
	@echo "Deployment complete. Running smoke tests..."
	@$(MAKE) smoke-test

# -----------------------------------------------------------------------------
# Smoke Tests - Quick verification of deployed application
# -----------------------------------------------------------------------------

# Verify deployed server is operational
# Tries port 443 first, falls back to 8443 if not responding
smoke-test: ## Run smoke tests against deployed server
	@echo ""
	@# Determine which port is responding
	@PORT=$(DEPLOY_PORT); \
	if ! curl -sf -k --connect-timeout 2 "https://$(DEPLOY_HOST):$$PORT/api/health" > /dev/null 2>&1; then \
		if [ "$$PORT" = "443" ]; then \
			PORT=8443; \
			echo "Port 443 not responding, trying 8443..."; \
		fi; \
	fi; \
	echo "=== Smoke Tests for $(DEPLOY_HOST):$$PORT ==="; \
	echo ""; \
	echo "1. Checking server process..."; \
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "pgrep -x $(BINARY_NAME) > /dev/null" && \
		echo "   PASS: Server process running" || \
		(echo "   FAIL: Server process not found" && exit 1); \
	echo ""; \
	echo "2. Checking API /api/status..."; \
	curl -sf -k "https://$(DEPLOY_HOST):$$PORT/api/status" > /dev/null && \
		echo "   PASS: API responding on port $$PORT" || \
		(echo "   FAIL: API not responding" && exit 1); \
	echo ""; \
	echo "3. Checking frontend loads..."; \
	curl -sf -k "https://$(DEPLOY_HOST):$$PORT/" | grep -q "<!DOCTYPE" && \
		echo "   PASS: Frontend HTML served" || \
		(echo "   FAIL: Frontend not loading" && exit 1); \
	echo ""; \
	echo "4. Checking iperf3 availability..."; \
	curl -sf -k "https://$(DEPLOY_HOST):$$PORT/api/iperf/info" | grep -q '"installed":true' && \
		echo "   PASS: iperf3 available" || \
		echo "   WARN: iperf3 not available (optional)"; \
	echo ""; \
	echo "=== All smoke tests passed! ==="; \
	echo "Server running at: https://$(DEPLOY_HOST):$$PORT"

# Local smoke tests for development
# Tries port 443 first, falls back to 8443
smoke-test-local: ## Run smoke tests against local server
	@echo "Running local smoke tests..."
	@PORT=443; \
	if ! curl -sf -k --connect-timeout 2 "https://localhost:$$PORT/api/health" > /dev/null 2>&1; then \
		PORT=8443; \
	fi; \
	echo "Testing on port $$PORT..."; \
	echo "1. API /api/status..."; \
	curl -sf -k "https://localhost:$$PORT/api/status" > /dev/null && \
		echo "   PASS" || (echo "   FAIL" && exit 1); \
	echo "2. Frontend loads..."; \
	curl -sf -k "https://localhost:$$PORT/" | grep -q "<!DOCTYPE" && \
		echo "   PASS" || (echo "   FAIL" && exit 1); \
	echo "All local tests passed! (port $$PORT)"

# =============================================================================
# Development
# =============================================================================

# Run with elevated privileges (required for raw socket operations)
run: ## Run the application (requires sudo for packet capture)
	sudo go run ./cmd/seed

# Development mode: backend reads frontend from disk
# Run alongside 'make dev-frontend' for full hot-reload experience
dev: ## Run backend in development mode (reads frontend from disk)
	go run -tags dev ./cmd/seed

# Vite dev server with hot module replacement
# Proxies API requests to backend on :8443
dev-frontend: ## Run frontend in development mode
	cd web && npm run dev

# =============================================================================
# Local Installation
# =============================================================================
#
# Install/uninstall Seed as a system service on the current machine.
# Automatically detects platform and uses appropriate service manager.
#
# =============================================================================

# Install on current system (macOS via launchd, Linux via systemd)
install: build ## Install as system service on current machine
	@printf "$(BOLD)Installing Seed on $$(uname -s)...$(RESET)\n"
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		if [ ! -f deploy/launchd/install.sh ]; then \
			echo "ERROR: deploy/launchd/install.sh not found"; \
			exit 1; \
		fi; \
		sudo ./deploy/launchd/install.sh ./$(BINARY_NAME); \
	elif [ "$$(uname -s)" = "Linux" ]; then \
		if [ ! -f deploy/systemd/install.sh ]; then \
			echo "ERROR: deploy/systemd/install.sh not found"; \
			exit 1; \
		fi; \
		sudo ./deploy/systemd/install.sh ./$(BINARY_NAME); \
	else \
		echo "ERROR: Unsupported platform: $$(uname -s)"; \
		exit 1; \
	fi

# Uninstall from current system
uninstall: ## Uninstall system service from current machine
	@printf "$(BOLD)Uninstalling Seed from $$(uname -s)...$(RESET)\n"
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		if [ -f deploy/launchd/uninstall.sh ]; then \
			sudo ./deploy/launchd/uninstall.sh; \
		else \
			echo "Removing launchd service manually..."; \
			sudo launchctl unload /Library/LaunchDaemons/com.seed.plist 2>/dev/null || true; \
			sudo rm -f /Library/LaunchDaemons/com.seed.plist; \
		fi; \
	elif [ "$$(uname -s)" = "Linux" ]; then \
		if [ -f deploy/systemd/uninstall.sh ]; then \
			sudo ./deploy/systemd/uninstall.sh; \
		else \
			echo "Removing systemd service manually..."; \
			sudo systemctl stop seed 2>/dev/null || true; \
			sudo systemctl disable seed 2>/dev/null || true; \
			sudo rm -f /etc/systemd/system/seed.service; \
			sudo systemctl daemon-reload; \
		fi; \
	else \
		echo "ERROR: Unsupported platform: $$(uname -s)"; \
		exit 1; \
	fi
	@printf "$(GREEN)✓ Uninstall complete$(RESET)\n"

# Show service status on current system
status: ## Show service status (local or remote)
	@printf "$(BOLD)Seed Service Status$(RESET)\n\n"
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		if launchctl list 2>/dev/null | grep -q com.seed; then \
			printf "$(GREEN)●$(RESET) Service: loaded\n"; \
			if pgrep -f "/usr/local/seed/seed" > /dev/null 2>&1; then \
				printf "$(GREEN)●$(RESET) Process: running (PID: $$(pgrep -f '/usr/local/seed/seed'))\n"; \
			else \
				printf "$(RED)●$(RESET) Process: not running\n"; \
			fi; \
		else \
			printf "$(YELLOW)●$(RESET) Service: not installed\n"; \
		fi; \
	elif [ "$$(uname -s)" = "Linux" ]; then \
		if systemctl is-enabled seed > /dev/null 2>&1; then \
			systemctl status seed --no-pager || true; \
		else \
			printf "$(YELLOW)●$(RESET) Service: not installed\n"; \
		fi; \
	fi
	@printf "\n$(BOLD)Remote Server ($(DEPLOY_HOST))$(RESET)\n"
	@if ssh -o BatchMode=yes -o ConnectTimeout=3 $(DEPLOY_USER)@$(DEPLOY_HOST) "true" 2>/dev/null; then \
		ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "systemctl is-active seed 2>/dev/null && echo 'Remote: running' || echo 'Remote: not running'"; \
	else \
		printf "$(YELLOW)●$(RESET) Remote: unreachable\n"; \
	fi

# Show version information
version: ## Show version info (current build and installed)
	@printf "$(BOLD)Seed Version Information$(RESET)\n\n"
	@printf "$(BOLD)Build:$(RESET)\n"
	@printf "  Version:    $(VERSION)\n"
	@printf "  Commit:     $(COMMIT)\n"
	@printf "  Build Time: $(BUILD_TIME)\n"
	@printf "  Platform:   $$(uname -s)/$$(uname -m)\n"
	@printf "\n$(BOLD)Installed:$(RESET)\n"
	@if [ -f /usr/local/seed/seed ]; then \
		printf "  Local:      $$(/usr/local/seed/seed version 2>/dev/null || echo 'unknown')\n"; \
	else \
		printf "  Local:      not installed\n"; \
	fi
	@if [ -f ./$(BINARY_NAME) ]; then \
		printf "  Binary:     $$(./$(BINARY_NAME) version 2>/dev/null || echo 'unknown')\n"; \
	fi

# =============================================================================
# Testing
# =============================================================================

# Run complete test suite (unit tests only)
test: ## Run unit tests (backend + frontend)
	@printf "$(BOLD)$(CYAN)┌─ Unit Tests ─────────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Backend (Go)                                                          $(CYAN)│$(RESET)\n"
	$(call timer-start,test-backend)
	@$(MAKE) --no-print-directory test-backend-quiet
	$(call timer-end,test-backend,Backend tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) Frontend (Vitest)                                                      $(CYAN)│$(RESET)\n"
	$(call timer-start,test-frontend)
	@$(MAKE) --no-print-directory test-frontend-quiet
	$(call timer-end,test-frontend,Frontend tests)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# Run ALL tests including E2E and Storybook
test-all: ## Run ALL tests (unit + E2E + Storybook)
	@printf "$(BOLD)$(CYAN)┌─ Full Test Suite ────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/4]$(RESET) Backend unit tests                                                    $(CYAN)│$(RESET)\n"
	$(call timer-start,test-backend)
	@$(MAKE) --no-print-directory test-backend-quiet
	$(call timer-end,test-backend,Backend tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/4]$(RESET) Frontend unit tests                                                   $(CYAN)│$(RESET)\n"
	$(call timer-start,test-frontend)
	@$(MAKE) --no-print-directory test-frontend-quiet
	$(call timer-end,test-frontend,Frontend tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/4]$(RESET) E2E tests (Playwright)                                                $(CYAN)│$(RESET)\n"
	$(call timer-start,test-e2e)
	@$(MAKE) --no-print-directory test-e2e
	$(call timer-end,test-e2e,E2E tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[4/4]$(RESET) Storybook build                                                       $(CYAN)│$(RESET)\n"
	$(call timer-start,test-storybook)
	@$(MAKE) --no-print-directory test-storybook
	$(call timer-end,test-storybook,Storybook)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# Go tests with race detection and coverage (verbose output)
# Excludes packages without test files to avoid covdata tool errors
test-backend: ## Run Go tests with progress
	@printf "\n$(BOLD)🧪 Running backend tests...$(RESET)\n"
	@PKGS=$$(go list ./... | grep -v '/cmd/' | grep -v '/web$$' | grep -v '/i18n$$' | grep -v '/mcp$$' | grep -v '/oauth$$'); \
	PKG_COUNT=$$(echo "$$PKGS" | wc -l | tr -d ' '); \
	printf "   📦 Testing $$PKG_COUNT packages...\n\n"; \
	if command -v gotestsum > /dev/null 2>&1; then \
		gotestsum --format pkgname-and-test-fails -- -race -coverprofile=coverage.out $$PKGS; \
	else \
		go test -race -coverprofile=coverage.out $$PKGS; \
	fi
	@if [ -f coverage.out ]; then \
		COV=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}'); \
		printf "\n   📊 Coverage: %s\n" "$$COV"; \
	fi
	@printf "\n$(GREEN)✓ Backend tests complete$(RESET)\n"

# Go tests (quiet mode for use in pipelines)
test-backend-quiet:
	@PKGS=$$(go list ./... | grep -v '/cmd/' | grep -v '/web$$' | grep -v '/i18n$$' | grep -v '/mcp$$' | grep -v '/oauth$$'); \
	PKG_COUNT=$$(echo "$$PKGS" | wc -l | tr -d ' '); \
	printf "   Testing $$PKG_COUNT packages...\n"; \
	go test -race -coverprofile=coverage.out $$PKGS 2>&1 | grep -E "^(ok|FAIL|---)" || true
	@if [ -f coverage.out ]; then \
		COV=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}'); \
		printf "   📊 Coverage: %s\n" "$$COV"; \
	fi

# Frontend unit tests via Vitest (verbose output)
test-frontend: ## Run frontend tests with progress
	@printf "\n$(BOLD)🧪 Running frontend tests...$(RESET)\n"
	@STORY_COUNT=$$(find web/src -name "*.test.ts" -o -name "*.test.tsx" 2>/dev/null | wc -l | tr -d ' '); \
	printf "   📦 Running $$STORY_COUNT test files...\n\n"
	@cd web && npm test
	@printf "\n$(GREEN)✓ Frontend tests complete$(RESET)\n"

# Frontend tests (quiet mode for use in pipelines)
test-frontend-quiet:
	@STORY_COUNT=$$(find web/src -name "*.test.ts" -o -name "*.test.tsx" 2>/dev/null | wc -l | tr -d ' '); \
	printf "   Running $$STORY_COUNT test files...\n"
	@cd web && npm test 2>&1 | grep -E "(PASS|FAIL|Tests:)" || true

# Frontend E2E tests via Playwright (closes #482, #309)
# Requires backend to be running on port 8443
test-e2e: ## Run frontend E2E tests (requires backend running)
	@echo ""
	@echo "🎭 Running E2E tests (Playwright)..."
	@E2E_COUNT=$$(find web/e2e -name "*.spec.ts" 2>/dev/null | wc -l | tr -d ' '); \
	echo "   📦 Running $$E2E_COUNT spec files..."
	@echo ""
	@cd web && npm run test:e2e
	@echo ""
	@echo "✅ E2E tests complete"

# Run E2E tests with UI mode for debugging
test-e2e-ui: ## Run E2E tests with Playwright UI
	@echo "🎭 Starting Playwright UI mode..."
	cd web && npm run test:e2e:ui

# Install Playwright browsers (required before first E2E run)
test-e2e-install: ## Install Playwright browsers
	cd web && npx playwright install --with-deps chromium

# Generate HTML coverage report
test-coverage: ## Generate coverage report
	@PKGS=$$(go list ./... | grep -v '/cmd/' | grep -v '/web$$' | grep -v '/i18n$$' | grep -v '/mcp$$' | grep -v '/oauth$$'); \
	go test -race -coverprofile=coverage.out $$PKGS
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# -----------------------------------------------------------------------------
# Storybook - Component Documentation & Visual Testing
# -----------------------------------------------------------------------------

# Run Storybook development server
storybook: ## Run Storybook development server (port 6006)
	@echo "📚 Starting Storybook..."
	cd web && npm run storybook

# Build static Storybook for deployment
build-storybook: ## Build static Storybook
	@echo "📚 Building Storybook..."
	@cd web && npm run build-storybook
	@echo "✅ Storybook built to web/storybook-static/"

# Test Storybook stories compile (useful in CI)
test-storybook: build-storybook ## Verify Storybook builds successfully
	@echo "✅ Storybook compilation test passed"

# Full integration test: deploy, install as systemd service, run tests
test-integration: build-linux-docker ## Full integration test on Ubuntu server via systemd
	@echo "Running integration tests on $(DEPLOY_HOST)..."
	rsync -avz $(BINARY_NAME)-linux-amd64 $(DEPLOY_USER)@$(DEPLOY_HOST):/tmp/seed-test
	ssh $(DEPLOY_USER)@$(DEPLOY_HOST) "\
		sudo systemctl stop seed-test 2>/dev/null || true && \
		sudo cp /tmp/seed-test /usr/local/bin/seed-test && \
		sudo chmod +x /usr/local/bin/seed-test && \
		sudo setcap cap_net_raw=+ep /usr/local/bin/seed-test && \
		echo '[Unit]' | sudo tee /etc/systemd/system/seed-test.service > /dev/null && \
		echo 'Description=The Seed Integration Test' | sudo tee -a /etc/systemd/system/seed-test.service > /dev/null && \
		echo '[Service]' | sudo tee -a /etc/systemd/system/seed-test.service > /dev/null && \
		echo 'Type=simple' | sudo tee -a /etc/systemd/system/seed-test.service > /dev/null && \
		echo 'ExecStart=/usr/local/bin/seed-test' | sudo tee -a /etc/systemd/system/seed-test.service > /dev/null && \
		echo 'WorkingDirectory=/tmp' | sudo tee -a /etc/systemd/system/seed-test.service > /dev/null && \
		echo '[Install]' | sudo tee -a /etc/systemd/system/seed-test.service > /dev/null && \
		echo 'WantedBy=multi-user.target' | sudo tee -a /etc/systemd/system/seed-test.service > /dev/null && \
		sudo systemctl daemon-reload && \
		sudo systemctl start seed-test && \
		sleep 5 && \
		sudo systemctl is-active seed-test && \
		curl -sf -k https://localhost:8443/api/health && \
		echo 'PASS: Integration test passed' && \
		sudo systemctl stop seed-test && \
		sudo rm -f /etc/systemd/system/seed-test.service /usr/local/bin/seed-test"
	@echo "Integration tests completed successfully"

# =============================================================================
# Linting & Formatting
# =============================================================================

# Run all linters
lint: ## Run all linters
	@printf "$(BOLD)$(CYAN)┌─ Linting ────────────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Backend (golangci-lint)                                               $(CYAN)│$(RESET)\n"
	$(call timer-start,lint-backend)
	@$(MAKE) --no-print-directory lint-backend-quiet
	$(call timer-end,lint-backend,Backend lint)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) Frontend (ESLint)                                                     $(CYAN)│$(RESET)\n"
	$(call timer-start,lint-frontend)
	@$(MAKE) --no-print-directory lint-frontend-quiet
	$(call timer-end,lint-frontend,Frontend lint)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# golangci-lint v2 with project configuration (.golangci.yml)
# Auto-installs if not found, uses full path to avoid PATH issues
lint-backend: ## Run Go linter
	@printf "$(BOLD)🔍 Running backend linter...$(RESET)\n"
	@GOLANGCI_LINT="$$(go env GOPATH)/bin/golangci-lint"; \
	if [ ! -f "$$GOLANGCI_LINT" ]; then \
		printf "📦 Installing golangci-lint v2...\n"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi; \
	$$GOLANGCI_LINT run
	@printf "$(GREEN)✓ Backend lint complete$(RESET)\n"

# Backend lint (quiet mode for pipelines)
lint-backend-quiet:
	@GOLANGCI_LINT="$$(go env GOPATH)/bin/golangci-lint"; \
	if [ ! -f "$$GOLANGCI_LINT" ]; then \
		printf "   Installing golangci-lint v2...\n"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi; \
	LINTER_COUNT=$$(grep -c "^    - " .golangci.yml 2>/dev/null || echo "30+"); \
	printf "   Running $$LINTER_COUNT linters...\n"; \
	$$GOLANGCI_LINT run 2>&1 | head -20 || true

# ESLint with TypeScript rules
lint-frontend: ## Run frontend linter
	@printf "$(BOLD)🔍 Running frontend linter...$(RESET)\n"
	@cd web && npm run lint
	@printf "$(GREEN)✓ Frontend lint complete$(RESET)\n"

# Frontend lint (quiet mode for pipelines)
lint-frontend-quiet:
	@FILE_COUNT=$$(find web/src -name "*.ts" -o -name "*.tsx" 2>/dev/null | wc -l | tr -d ' '); \
	printf "   Checking $$FILE_COUNT files...\n"
	@cd web && npm run lint 2>&1 | tail -5 || true

# -----------------------------------------------------------------------------
# Auto-Fix - Automatically fix linting and formatting issues
# -----------------------------------------------------------------------------

# Fix all code issues (Go + Frontend)
fix: ## Auto-fix all linting issues (Go + Frontend)
	@printf "$(BOLD)$(CYAN)┌─ Auto-Fix ───────────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Backend (golangci-lint --fix)                                         $(CYAN)│$(RESET)\n"
	$(call timer-start,fix-backend)
	@$(MAKE) --no-print-directory fix-backend-quiet
	$(call timer-end,fix-backend,Backend fix)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) Frontend (eslint --fix + prettier)                                    $(CYAN)│$(RESET)\n"
	$(call timer-start,fix-frontend)
	@$(MAKE) --no-print-directory fix-frontend-quiet
	$(call timer-end,fix-frontend,Frontend fix)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# Auto-fix Go linting issues
fix-backend: ## Auto-fix Go linting issues
	@printf "$(BOLD)🔧 Auto-fixing Go code...$(RESET)\n"
	@GOLANGCI_LINT="$$(go env GOPATH)/bin/golangci-lint"; \
	if [ ! -f "$$GOLANGCI_LINT" ]; then \
		printf "📦 Installing golangci-lint v2...\n"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi; \
	$$GOLANGCI_LINT run --fix
	@gofmt -w -s .
	@printf "$(GREEN)✓ Go auto-fix complete$(RESET)\n"

# Backend fix (quiet mode for pipelines)
fix-backend-quiet:
	@GOLANGCI_LINT="$$(go env GOPATH)/bin/golangci-lint"; \
	if [ ! -f "$$GOLANGCI_LINT" ]; then \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi; \
	$$GOLANGCI_LINT run --fix 2>&1 | grep -E "^[0-9]+ issues" || printf "   No issues found\n"
	@gofmt -w -s .

# Auto-fix frontend linting issues
fix-frontend: ## Auto-fix frontend linting issues
	@printf "$(BOLD)🔧 Auto-fixing frontend code...$(RESET)\n"
	@cd web && npm run lint:fix
	@cd web && npx prettier --write .
	@printf "$(GREEN)✓ Frontend auto-fix complete$(RESET)\n"

# Frontend fix (quiet mode for pipelines)
fix-frontend-quiet:
	@cd web && npm run lint:fix 2>&1 | tail -3 || true
	@cd web && npx prettier --write . 2>&1 | tail -1 || true

# Auto-fix markdown formatting
fix-md: fmt-md ## Auto-fix markdown formatting

# Fix everything (Go + Frontend + Markdown)
fix-all: fix fix-md ## Auto-fix all code and documentation

# Format Go code
fmt: ## Format Go code with gofumpt
	@if ! command -v gofumpt > /dev/null 2>&1; then \
		echo "📦 Installing gofumpt..."; \
		go install mvdan.cc/gofumpt@latest; \
	fi
	@gofumpt -w .
	@echo "✅ Go code formatted"

# Format frontend code
fmt-frontend: ## Format frontend code with Prettier
	@cd web && npx prettier --write .
	@echo "✅ Frontend code formatted"

# Format markdown files
fmt-md: ## Format markdown files with Prettier
	npm run format:md
	@echo "Markdown files formatted"

# Format all code
fmt-all: fmt fmt-frontend fmt-md ## Format all code (Go + frontend + markdown)

# Check formatting without making changes
fmt-check: ## Check all formatting (Go + frontend + markdown) without fixing
	@echo "🔍 Checking formatting..."
	@FAILED=0; \
	echo "Checking Go formatting..."; \
	if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ Go files need formatting:"; \
		gofmt -l .; \
		FAILED=1; \
	else \
		echo "✅ Go formatting OK"; \
	fi; \
	echo "Checking frontend formatting..."; \
	if ! (cd web && npx prettier --check . 2>/dev/null); then \
		echo "❌ Frontend files need formatting"; \
		FAILED=1; \
	else \
		echo "✅ Frontend formatting OK"; \
	fi; \
	echo "Checking markdown formatting..."; \
	if ! npm run format:md:check 2>/dev/null; then \
		echo "❌ Markdown files need formatting"; \
		FAILED=1; \
	else \
		echo "✅ Markdown formatting OK"; \
	fi; \
	if [ $$FAILED -ne 0 ]; then \
		echo ""; \
		echo "Run 'make fmt-all' to fix formatting issues"; \
		exit 1; \
	fi
	@echo "✅ All formatting checks passed"

# Lint markdown files with markdownlint
lint-md: ## Lint markdown files with markdownlint
	@printf "$(BOLD)🔍 Linting markdown files...$(RESET)\n"
	@if command -v markdownlint-cli2 > /dev/null 2>&1; then \
		markdownlint-cli2 "**/*.md"; \
	elif npx markdownlint-cli2 --help > /dev/null 2>&1; then \
		npx markdownlint-cli2 "**/*.md"; \
	else \
		printf "$(YELLOW)SKIP: markdownlint-cli2 not installed (npm install -g markdownlint-cli2)$(RESET)\n"; \
	fi
	@printf "$(GREEN)✓ Markdown lint complete$(RESET)\n"

# =============================================================================
# Security Scanning
# =============================================================================

# Run all security scans
security: ## Run all security scans
	@printf "$(BOLD)$(CYAN)┌─ Security Scanning ──────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/3]$(RESET) Go Vulnerabilities (govulncheck)                                       $(CYAN)│$(RESET)\n"
	$(call timer-start,security-backend)
	@$(MAKE) --no-print-directory security-backend-quiet
	$(call timer-end,security-backend,Go vulnerability scan)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/3]$(RESET) npm Vulnerabilities (npm audit)                                        $(CYAN)│$(RESET)\n"
	$(call timer-start,security-frontend)
	@$(MAKE) --no-print-directory security-frontend-quiet
	$(call timer-end,security-frontend,npm audit)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/3]$(RESET) Secret Scanning (gitleaks)                                             $(CYAN)│$(RESET)\n"
	$(call timer-start,security-secrets)
	@$(MAKE) --no-print-directory security-secrets-quiet
	$(call timer-end,security-secrets,Secret scan)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# License compliance checking
license-check: ## Check license compliance (Go + npm)
	@printf "$(BOLD)🔍 Checking license compliance...$(RESET)\n\n"
	@printf "$(BOLD)=== Go Dependencies ===$(RESET)\n"
	@if command -v go-licenses > /dev/null 2>&1; then \
		go-licenses check ./... --disallowed_types=forbidden,restricted --allowed_licenses=MIT,Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0,CC0-1.0; \
	else \
		printf "$(YELLOW)SKIP: go-licenses not installed$(RESET)\n"; \
	fi
	@printf "\n$(BOLD)=== npm Dependencies ===$(RESET)\n"
	@cd web && npx license-checker --production --excludePrivatePackages --onlyAllow "MIT;Apache-2.0;BSD-2-Clause;BSD-3-Clause;ISC;CC0-1.0;0BSD;BlueOak-1.0.0;Unlicense;MPL-2.0" --summary
	@printf "\n$(GREEN)✓ License compliance check complete$(RESET)\n"

license-report: ## Generate license compliance reports
	@printf "$(BOLD)Generating license reports...$(RESET)\n"
	@mkdir -p build/reports
	@if command -v go-licenses > /dev/null 2>&1; then \
		go-licenses report ./... > build/reports/go-licenses.csv; \
		printf "Go licenses: build/reports/go-licenses.csv\n"; \
	fi
	@cd web && npx license-checker --production --csv > ../build/reports/npm-licenses.csv
	@printf "npm licenses: build/reports/npm-licenses.csv\n"
	@printf "$(GREEN)✓ License reports generated$(RESET)\n"

# Go vulnerability scanning (govulncheck)
# Note: gosec runs via golangci-lint in lint-backend, no need to duplicate
# Auto-installs govulncheck if not found
security-backend: ## Run Go vulnerability scan (govulncheck)
	@printf "$(BOLD)🔒 Running Go vulnerability scan...$(RESET)\n"
	@if ! command -v govulncheck > /dev/null 2>&1; then \
		printf "📦 Installing govulncheck...\n"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck ./...
	@printf "$(GREEN)✓ Go vulnerability scan complete$(RESET)\n"

# Backend security (quiet mode)
security-backend-quiet:
	@if ! command -v govulncheck > /dev/null 2>&1; then \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@printf "   Scanning Go dependencies...\n"
	@govulncheck ./... 2>&1 | grep -E "(Vulnerability|No vulnerabilities)" | head -5 || printf "   No vulnerabilities found\n"

# Frontend security scanning (npm audit)
security-frontend: ## Run frontend security scan (npm audit)
	@printf "$(BOLD)🔒 Running npm audit...$(RESET)\n"
	@cd web && npm audit --audit-level=high
	@printf "$(GREEN)✓ npm audit complete$(RESET)\n"

# Frontend security (quiet mode)
security-frontend-quiet:
	@printf "   Auditing npm packages...\n"
	@cd web && npm audit --audit-level=high 2>&1 | grep -E "(found|vulnerabilities)" | head -3 || printf "   No vulnerabilities found\n"

# Secret scanning (gitleaks)
# Auto-installs if not found, uses full path for portability
security-secrets: ## Scan for secrets in codebase (gitleaks)
	@printf "$(BOLD)🔒 Running gitleaks...$(RESET)\n"
	@GITLEAKS=$$(command -v gitleaks 2>/dev/null || echo "$$(go env GOPATH)/bin/gitleaks"); \
	if [ ! -x "$$GITLEAKS" ]; then \
		printf "📦 Installing gitleaks...\n"; \
		go install github.com/zricethezav/gitleaks/v8@latest; \
		GITLEAKS="$$(go env GOPATH)/bin/gitleaks"; \
	fi; \
	$$GITLEAKS detect --source . --config .gitleaks.toml --verbose
	@printf "$(GREEN)✓ Secret scan complete$(RESET)\n"

# Secret scanning (quiet mode)
security-secrets-quiet:
	@GITLEAKS=$$(command -v gitleaks 2>/dev/null || echo "$$(go env GOPATH)/bin/gitleaks"); \
	if [ ! -x "$$GITLEAKS" ]; then \
		go install github.com/zricethezav/gitleaks/v8@latest; \
		GITLEAKS="$$(go env GOPATH)/bin/gitleaks"; \
	fi; \
	printf "   Scanning for secrets...\n"; \
	$$GITLEAKS detect --source . --config .gitleaks.toml 2>&1 | grep -E "(leaks found|no leaks)" || printf "   No secrets found\n"

# Trivy filesystem scan (if available)
security-trivy: ## Run Trivy vulnerability scan
	@printf "$(BOLD)🔒 Running Trivy filesystem scan...$(RESET)\n"
	@if command -v trivy > /dev/null 2>&1; then \
		trivy fs --severity HIGH,CRITICAL .; \
	else \
		printf "$(YELLOW)SKIP: trivy not installed (brew install trivy)$(RESET)\n"; \
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
	npm ci

# =============================================================================
# Dependency Updates - Keep everything current
# =============================================================================

# Update everything to latest versions
update: ## Update ALL dependencies and tools to latest versions
	@printf "$(BOLD)$(CYAN)┌─ Updating All Dependencies ─────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/5]$(RESET) Go modules                                                            $(CYAN)│$(RESET)\n"
	$(call timer-start,update-go)
	@$(MAKE) --no-print-directory update-go-quiet
	$(call timer-end,update-go,Go modules)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/5]$(RESET) npm packages (root)                                                   $(CYAN)│$(RESET)\n"
	$(call timer-start,update-npm-root)
	@npm update 2>&1 | tail -3 || true
	$(call timer-end,update-npm-root,npm root)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/5]$(RESET) npm packages (web)                                                    $(CYAN)│$(RESET)\n"
	$(call timer-start,update-npm-web)
	@cd web && npm update 2>&1 | tail -3 || true
	$(call timer-end,update-npm-web,npm web)
	@printf "$(CYAN)│$(RESET) $(BOLD)[4/5]$(RESET) Go development tools                                                  $(CYAN)│$(RESET)\n"
	$(call timer-start,update-tools)
	@$(MAKE) --no-print-directory tools-go-quiet
	$(call timer-end,update-tools,Go tools)
	@printf "$(CYAN)│$(RESET) $(BOLD)[5/5]$(RESET) Playwright browsers                                                   $(CYAN)│$(RESET)\n"
	$(call timer-start,update-playwright)
	@cd web && npx playwright install chromium 2>&1 | tail -2 || true
	$(call timer-end,update-playwright,Playwright)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "\n$(GREEN)✓ All dependencies updated!$(RESET)\n"
	@printf "Run '$(BOLD)make version-check$(RESET)' to verify versions\n\n"

# Update Go modules only
update-go: ## Update Go dependencies
	@printf "$(BOLD)📦 Updating Go modules...$(RESET)\n"
	@go get -u ./...
	@go mod tidy
	@printf "$(GREEN)✓ Go modules updated$(RESET)\n"

# Update Go modules (quiet mode)
update-go-quiet:
	@printf "   Fetching latest versions...\n"
	@go get -u ./... 2>&1 | tail -5 || true
	@go mod tidy
	@MOD_COUNT=$$(go list -m all | wc -l | tr -d ' '); \
	printf "   $$MOD_COUNT modules in go.mod\n"

# Update npm packages only
update-npm: ## Update npm dependencies
	@printf "$(BOLD)📦 Updating npm packages...$(RESET)\n"
	@npm update
	@cd web && npm update
	@printf "$(GREEN)✓ npm packages updated$(RESET)\n"

# Show version information for all tools and dependencies
version-check: ## Show versions of all tools and dependencies
	@printf "$(BOLD)$(CYAN)═══════════════════════════════════════════════════════════════════════════════$(RESET)\n"
	@printf "$(BOLD)                           VERSION CHECK$(RESET)\n"
	@printf "$(CYAN)═══════════════════════════════════════════════════════════════════════════════$(RESET)\n\n"
	@printf "$(BOLD)Runtime:$(RESET)\n"
	@printf "  Go:            %s\n" "$$(go version | awk '{print $$3}')"
	@printf "  Node.js:       %s\n" "$$(node --version)"
	@printf "  npm:           %s\n" "$$(npm --version)"
	@printf "\n$(BOLD)Go Tools:$(RESET) (in $$(go env GOPATH)/bin)\n"
	@printf "  golangci-lint: %s\n" "$$($$(go env GOPATH)/bin/golangci-lint --version 2>/dev/null | head -1 | awk '{print $$4}' || echo 'not installed')"
	@printf "  govulncheck:   %s\n" "$$(test -f $$(go env GOPATH)/bin/govulncheck && echo 'installed' || echo 'not installed')"
	@printf "  gofumpt:       %s\n" "$$($$(go env GOPATH)/bin/gofumpt --version 2>/dev/null || echo 'not installed')"
	@printf "  gitleaks:      %s\n" "$$($$(go env GOPATH)/bin/gitleaks version 2>/dev/null || echo 'not installed')"
	@printf "\n$(BOLD)Project Dependencies:$(RESET)\n"
	@printf "  Go modules:    %s packages\n" "$$(go list -m all 2>/dev/null | wc -l | tr -d ' ')"
	@printf "  npm (root):    %s packages\n" "$$(npm ls --depth=0 2>/dev/null | wc -l | tr -d ' ')"
	@printf "  npm (web):     %s packages\n" "$$(cd web && npm ls --depth=0 2>/dev/null | wc -l | tr -d ' ')"
	@printf "\n$(BOLD)Outdated Packages:$(RESET)\n"
	@GO_OUTDATED=$$(go list -u -m all 2>/dev/null | grep '\[' | wc -l | tr -d ' '); \
	printf "  Go:            %s can be updated\n" "$$GO_OUTDATED"
	@NPM_OUTDATED=$$(cd web && npm outdated 2>/dev/null | wc -l | tr -d ' '); \
	printf "  npm:           %s can be updated\n" "$$NPM_OUTDATED"
	@printf "\n$(CYAN)═══════════════════════════════════════════════════════════════════════════════$(RESET)\n"

# Show outdated packages with details
outdated: ## Show outdated packages (Go + npm)
	@printf "$(BOLD)$(CYAN)┌─ Outdated Go Modules ────────────────────────────────────────────────────────┐$(RESET)\n"
	@go list -u -m all 2>/dev/null | grep '\[' | head -20 || printf "   All Go modules are up to date\n"
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "\n$(BOLD)$(CYAN)┌─ Outdated npm Packages (web) ───────────────────────────────────────────────┐$(RESET)\n"
	@cd web && npm outdated 2>/dev/null || printf "   All npm packages are up to date\n"
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# =============================================================================
# Development Tools Installation
# =============================================================================

# Install all development tools (Go + Frontend)
tools: tools-go tools-frontend ## Install all development tools (Go + Node)
	@printf "\n$(GREEN)✓ All development tools installed!$(RESET)\n\n"
	@printf "Go tools: $$(go env GOPATH)/bin\n"
	@printf "Ensure $$(go env GOPATH)/bin is in your PATH\n"

# Install Go development tools (linters, security scanners, formatters)
tools-go: ## Install Go development tools
	@printf "$(BOLD)=== Installing Go Development Tools ===$(RESET)\n"
	@printf "  golangci-lint (comprehensive linter)...\n"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@printf "  govulncheck (CVE vulnerability checker)...\n"
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@printf "  goimports (import formatter)...\n"
	@go install golang.org/x/tools/cmd/goimports@latest
	@printf "  gofumpt (strict formatter)...\n"
	@go install mvdan.cc/gofumpt@latest
	@printf "  gitleaks (secret scanner)...\n"
	@go install github.com/zricethezav/gitleaks/v8@latest
	@printf "  deadcode (unused code finder)...\n"
	@go install golang.org/x/tools/cmd/deadcode@latest
	@printf "  gotestsum (test runner with better output)...\n"
	@go install gotest.tools/gotestsum@latest
	@printf "  go-licenses (license compliance checker)...\n"
	@go install github.com/google/go-licenses@latest
	@printf "$(GREEN)✓ Go tools installed$(RESET)\n"

# Go tools (quiet mode)
tools-go-quiet:
	@printf "   Installing latest versions...\n"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null
	@go install golang.org/x/vuln/cmd/govulncheck@latest 2>/dev/null
	@go install golang.org/x/tools/cmd/goimports@latest 2>/dev/null
	@go install mvdan.cc/gofumpt@latest 2>/dev/null
	@go install github.com/zricethezav/gitleaks/v8@latest 2>/dev/null
	@go install golang.org/x/tools/cmd/deadcode@latest 2>/dev/null
	@go install gotest.tools/gotestsum@latest 2>/dev/null
	@go install github.com/google/go-licenses@latest 2>/dev/null
	@printf "   8 tools updated\n"

# Install frontend development tools
tools-frontend: ## Install frontend development tools
	@printf "\n$(BOLD)=== Installing Frontend Development Tools ===$(RESET)\n"
	@printf "Installing root npm dependencies (prettier, husky)...\n"
	@npm ci
	@printf "Installing web dependencies...\n"
	@cd web && npm ci
	@printf "Installing Playwright browsers...\n"
	@cd web && npx playwright install --with-deps chromium 2>/dev/null || printf "Playwright install skipped (run 'make test-e2e-install' manually)\n"
	@echo "✅ Frontend tools installed"

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
	@echo "The Seed - Network Diagnostics Tool by Mustard Seed Networks"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
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
verify: ## Full verification (lint, test, security, build, docker if available)
	@printf "\n$(BOLD)$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(BOLD)$(CYAN)║                        FULL VERIFICATION PIPELINE                           ║$(RESET)\n"
	@printf "$(BOLD)$(CYAN)║                        Version: $(VERSION)$(RESET)\n"
	@printf "$(BOLD)$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	$(call timer-start,verify-total)
	$(call step,1,5,Linting Code)
	$(call timer-start,lint)
	@$(MAKE) --no-print-directory lint
	$(call timer-end,lint,Linting)
	$(call step,2,5,Running Tests)
	$(call timer-start,test)
	@$(MAKE) --no-print-directory test
	$(call timer-end,test,Tests)
	$(call step,3,5,Security Scanning)
	$(call timer-start,security)
	@$(MAKE) --no-print-directory security
	$(call timer-end,security,Security)
	$(call step,4,5,Building Application)
	$(call timer-start,build)
	@$(MAKE) --no-print-directory build
	$(call timer-end,build,Build)
	$(call step,5,5,Docker Verification)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		$(call timer-start,docker); \
		$(MAKE) --no-print-directory docker-test; \
		$(call timer-end,docker,Docker); \
	else \
		printf "$(YELLOW)⊘ SKIP: Docker not available - skipping docker-test$(RESET)\n"; \
		printf "$(YELLOW)  Run 'make docker-test' manually when Docker is running$(RESET)\n"; \
	fi
	@printf "\n$(BOLD)$(GREEN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(BOLD)$(GREEN)║                        ✓ VERIFICATION COMPLETE                               ║$(RESET)\n"
	@printf "$(BOLD)$(GREEN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	$(call timer-end,verify-total,Total verification)
	@printf "\n  $(BOLD)Version:$(RESET)     $(VERSION)\n"
	@printf "  $(BOLD)Commit:$(RESET)      $(COMMIT)\n"
	@printf "  $(BOLD)Binary:$(RESET)      $(BINARY_NAME)\n"
	@printf "  $(BOLD)Docker:$(RESET)      $(DOCKER_IMAGE):$(DOCKER_TAG)\n\n"
	@printf "$(GREEN)Ready for deployment. Run 'make deploy' to deploy.$(RESET)\n\n"

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
# Internationalization (i18n)
# =============================================================================

# Sync locale files from root to internal/i18n for Go embedding
i18n-sync: ## Sync locale files for Go embedding
	@echo "Syncing locale files..."
	@mkdir -p internal/i18n/locales/en internal/i18n/locales/es
	@cp locales/en/*.json internal/i18n/locales/en/
	@cp locales/es/*.json internal/i18n/locales/es/
	@echo "✅ Locale files synced to internal/i18n/locales/"

# Check for missing translation keys between locales
i18n-check: ## Check for missing translation keys
	@echo "Checking translation key coverage..."
	@bash -c 'for file in locales/en/*.json; do \
		name=$$(basename $$file); \
		if [ ! -f "locales/es/$$name" ]; then \
			echo "❌ Missing Spanish file: $$name"; \
			exit 1; \
		fi; \
		en_keys=$$(jq -r "paths(scalars) | join(\".\")" $$file | sort); \
		es_keys=$$(jq -r "paths(scalars) | join(\".\")" locales/es/$$name | sort); \
		missing=$$(comm -23 <(echo "$$en_keys") <(echo "$$es_keys")); \
		if [ -n "$$missing" ]; then \
			echo "❌ Missing keys in es/$$name:"; \
			echo "$$missing" | head -10; \
			exit 1; \
		fi; \
	done'
	@echo "✅ All translation keys present in all locales"

# List all translation keys
i18n-list: ## List all translation keys in English locale
	@for file in locales/en/*.json; do \
		echo "=== $$(basename $$file) ==="; \
		jq -r 'paths(scalars) | join(".")' $$file; \
	done

# =============================================================================
# ISO Creation (Manual Process)
# =============================================================================

# Custom Ubuntu ISO creation is a separate, manual process
iso-info: ## Show instructions for creating custom Ubuntu ISO
	@echo ""
	@echo "=== Custom Ubuntu ISO Creation ==="
	@echo ""
	@echo "Creating a custom Ubuntu ISO with The Seed pre-installed requires:"
	@echo "  1. Ubuntu host system (or VM)"
	@echo "  2. cubic or live-build package"
	@echo "  3. Built Linux binary (seed-linux-amd64)"
	@echo ""
	@echo "Steps:"
	@echo "  1. Install cubic: sudo apt install cubic"
	@echo "  2. Download Ubuntu Server ISO"
	@echo "  3. Run cubic and customize:"
	@echo "     - Copy seed-linux-amd64 to /usr/local/bin/seed"
	@echo "     - Copy deploy/systemd/seed.service to /etc/systemd/system/"
	@echo "     - Enable service: systemctl enable seed"
	@echo "  4. Generate ISO"

# =============================================================================
# Release Management
# =============================================================================
#
# Automated release workflow:
#   1. make release-check       - Validate everything is ready
#   2. make release VERSION=vX.Y.Z - Tag, build, and create GitHub release
#
# =============================================================================

# Create a new release with multi-platform builds
# Usage: make release VERSION=v1.0.0
release: ## Create release (VERSION=vX.Y.Z required)
	@if [ -z "$(VERSION)" ] || ! echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+'; then \
		echo "ERROR: VERSION must be set in format vX.Y.Z"; \
		echo "Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "ERROR: Working directory not clean. Commit or stash changes first."; \
		exit 1; \
	fi
	@printf "$(BOLD)$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(BOLD)$(CYAN)║                         RELEASE $(VERSION)                                   ║$(RESET)\n"
	@printf "$(BOLD)$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	@echo ""
	@echo "Step 1/5: Running verification..."
	@$(MAKE) --no-print-directory verify
	@echo ""
	@echo "Step 2/5: Building all platforms..."
	@$(MAKE) --no-print-directory build-all
	@echo ""
	@echo "Step 3/5: Creating git tag..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Tagged: $(VERSION)"
	@echo ""
	@echo "Step 4/5: Pushing tag to origin..."
	@git push origin $(VERSION)
	@echo ""
	@echo "Step 5/5: Creating GitHub release..."
	@if command -v gh > /dev/null 2>&1; then \
		gh release create $(VERSION) \
			--title "$(VERSION)" \
			--generate-notes \
			$(BINARY_NAME)-darwin-* $(BINARY_NAME)-linux-* 2>/dev/null || \
			echo "Note: Upload binaries manually if gh release failed"; \
	else \
		echo "GitHub CLI not installed. Create release manually at:"; \
		echo "https://github.com/krisarmstrong/seed/releases/new?tag=$(VERSION)"; \
	fi
	@echo ""
	@printf "$(GREEN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(GREEN)║                     ✓ RELEASE $(VERSION) COMPLETE                            ║$(RESET)\n"
	@printf "$(GREEN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	@echo ""
	@echo "Binaries:"
	@ls -lh $(BINARY_NAME)-* 2>/dev/null || true
	@echo ""
	@echo "Next steps:"
	@echo "  - Verify release at: https://github.com/krisarmstrong/seed/releases"
	@echo "  - Update CHANGELOG.md if needed"
	@echo "  - Announce release"

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
