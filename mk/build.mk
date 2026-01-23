# =============================================================================
# Build Targets
# =============================================================================
#
# All build-related targets for The Seed:
#   - iperf3 compilation (native + cross-platform)
#   - Frontend build (React/Vite)
#   - Backend build (Go with embedded assets)
#   - Cross-compilation (Linux, macOS, ARM64)
#   - Docker image builds
#
# =============================================================================

.PHONY: build build-iperf3 build-iperf3-quiet build-iperf3-linux build-iperf3-linux-amd64 \
        build-iperf3-linux-arm64 build-iperf3-all \
        frontend-deps generate-types build-frontend build-frontend-quiet \
        build-backend build-backend-quiet build-backend-dev \
        build-linux-amd64 build-linux-arm64 build-linux-docker \
        build-darwin build-windows build-all \
        docker-build docker-test docker docker-push \
        run dev dev-frontend

# =============================================================================
# Main Build Target
# =============================================================================

# Build complete application (frontend embedded in Go binary)
build: ## Build everything (frontend embedded in binary)
	@printf "$(BOLD)$(CYAN)┌─ Building Application ───────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/3]$(RESET) iperf3 (native platform)                                              $(CYAN)│$(RESET)\n"
	$(call timer-start,build-iperf3)
	@$(MAKE) --no-print-directory build-iperf3-quiet
	$(call timer-end,build-iperf3,iperf3 build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/3]$(RESET) Frontend (React/TypeScript)                                           $(CYAN)│$(RESET)\n"
	$(call timer-start,build-frontend)
	@$(MAKE) --no-print-directory build-frontend-quiet
	$(call timer-end,build-frontend,Frontend build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/3]$(RESET) Backend (Go)                                                          $(CYAN)│$(RESET)\n"
	$(call timer-start,build-backend)
	@$(MAKE) --no-print-directory build-backend-quiet
	$(call timer-end,build-backend,Backend build)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "$(GREEN)✓ Build complete: $(BINARY_NAME) ($(VERSION))$(RESET)\n"

# =============================================================================
# iperf3 Builds
# =============================================================================

build-iperf3: ## Build iperf3 from source (native platform)
	@echo "Building iperf3 for native platform..."
	@./scripts/build-iperf3.sh

build-iperf3-quiet:
	@if [ ! -f internal/iperf/binaries/iperf3-$$(uname -s | tr '[:upper:]' '[:lower:]')-$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') ]; then \
		printf "   Building iperf3...\n"; \
		./scripts/build-iperf3.sh > /dev/null 2>&1; \
		printf "   Output: internal/iperf/binaries/\n"; \
	else \
		printf "   iperf3 already built (skipping)\n"; \
	fi

build-iperf3-linux: ## Build iperf3 for Linux (AMD64 + ARM64 via Docker)
	@echo "Cross-compiling iperf3 for Linux (all architectures)..."
	@./scripts/build-iperf3-linux.sh all

build-iperf3-linux-amd64:
	@./scripts/build-iperf3-linux.sh amd64

build-iperf3-linux-arm64:
	@./scripts/build-iperf3-linux.sh arm64

build-iperf3-all: build-iperf3 build-iperf3-linux ## Build iperf3 for all platforms

# =============================================================================
# Frontend Build
# =============================================================================

frontend-deps: ## Install frontend dependencies (cached)
	@if [ ! -d ui/node_modules ] || [ ui/package-lock.json -nt ui/node_modules/.package-lock.json ]; then \
		echo "Installing frontend dependencies..."; \
		cd ui && npm ci; \
	else \
		echo "Frontend dependencies up to date"; \
	fi

generate-types: frontend-deps ## Generate TypeScript types from JSON Schema
	@echo "🔧 Generating TypeScript types from schema..."
	@./scripts/generate-types.sh
	@echo "✅ TypeScript types generated"

build-frontend: frontend-deps ## Build React frontend
	@printf "$(BOLD)🔨 Building frontend...$(RESET)\n"
	@cd ui && npm run build
	@printf "$(GREEN)✓ Frontend build complete$(RESET)\n"

build-frontend-quiet: frontend-deps
	@printf "   Bundling React application...\n"
	@command -v npm >/dev/null 2>&1 || { printf "$(RED)ERROR: npm not found. Frontend build requires npm.$(RESET)\n"; exit 1; }
	@cd ui && npm run build
	@SIZE=$$(du -sh ui/dist 2>/dev/null | cut -f1 || echo "unknown"); \
	printf "   Output: ui/dist ($$SIZE)\n"

# =============================================================================
# Backend Build
# =============================================================================

build-backend: ## Build Go backend with embedded frontend
	@printf "$(BOLD)🔨 Building backend...$(RESET)\n"
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/seed
	@printf "$(GREEN)✓ Backend build complete: $(BINARY_NAME)$(RESET)\n"

build-backend-quiet:
	@printf "   Compiling Go binary...\n"
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/seed
	@SIZE=$$(ls -lh $(BINARY_NAME) | awk '{print $$5}'); \
	printf "   Output: $(BINARY_NAME) ($$SIZE)\n"

build-backend-dev: ## Build Go backend in dev mode (reads frontend from disk)
	CGO_ENABLED=1 go build -tags dev $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/seed

# =============================================================================
# Cross-Platform Builds
# =============================================================================

build-linux-amd64: build-frontend ## Build for Linux AMD64 (requires cross-compiler)
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-gnu-gcc \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 ./cmd/seed

build-linux-arm64: build-frontend ## Build for Linux ARM64 (Raspberry Pi, ARM servers)
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-arm64 ./cmd/seed

build-linux-docker: build-frontend ## Build for Linux AMD64 using Docker (cross-platform)
	@echo "Building for Linux AMD64 using Docker..."
	docker run --rm -v "$(PWD):/build" -w /build golang:1.25.5 bash -c "\
		apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
		go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-amd64 ./cmd/seed"
	@echo "Built: $(BINARY_NAME)-linux-amd64"

build-darwin: build-frontend ## Build for macOS (native architecture)
	@echo "Building for macOS ($(shell uname -m))..."
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-darwin-$(shell uname -m) ./cmd/seed
	@echo "Built: $(BINARY_NAME)-darwin-$(shell uname -m)"

build-windows: build-frontend ## Build for Windows AMD64 (cross-compile)
	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-windows-amd64.exe ./cmd/seed
	@echo "Built: $(BINARY_NAME)-windows-amd64.exe"

build-all: build-frontend ## Build for all platforms (native + Linux via Docker)
	@printf "$(BOLD)$(CYAN)┌─ Building All Platforms ────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/3]$(RESET) macOS (native)                                                       $(CYAN)│$(RESET)\n"
	$(call timer-start,build-darwin)
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-darwin-$(shell uname -m) ./cmd/seed
	$(call timer-end,build-darwin,macOS build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/3]$(RESET) Linux AMD64 (Docker)                                                 $(CYAN)│$(RESET)\n"
	$(call timer-start,build-linux-amd64)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		docker run --rm -v "$(PWD):/build" -w /build golang:1.25.5 bash -c "\
			apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
			go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-amd64 ./cmd/seed"; \
	else \
		printf "$(YELLOW)   SKIP: Docker not available$(RESET)\n"; \
	fi
	$(call timer-end,build-linux-amd64,Linux AMD64 build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/3]$(RESET) Linux ARM64 (Docker)                                                 $(CYAN)│$(RESET)\n"
	$(call timer-start,build-linux-arm64)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		docker run --rm -v "$(PWD):/build" -w /build --platform linux/arm64 golang:1.25.5 bash -c "\
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

docker-build: ## Build Docker image
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-test: docker-build ## Test Docker image starts and responds
	@echo "Testing Docker image..."
	@docker run --rm -d --name seed-test -p 8443:8443 $(DOCKER_IMAGE):$(DOCKER_TAG)
	@sleep 5
	@curl -sf -k https://localhost:8443/api/health || (docker stop seed-test && exit 1)
	@docker stop seed-test
	@echo "Docker test passed"

docker: docker-test ## Build and test Docker image

docker-push: docker-build ## Push Docker image to registry
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "ERROR: DOCKER_REGISTRY not set"; \
		exit 1; \
	fi
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

# =============================================================================
# Development Targets
# =============================================================================

run: ## Run the application (requires sudo for packet capture)
	@echo "Running $(BINARY_NAME) (use Ctrl+C to stop)..."
	@sudo ./$(BINARY_NAME)

dev: ## Run backend in development mode (reads frontend from disk)
	@echo "Running backend in development mode..."
	@go run -tags dev ./cmd/seed

dev-frontend: ## Run frontend in development mode
	@echo "Starting Vite dev server..."
	@cd ui && npm run dev
