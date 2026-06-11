# =============================================================================
# Build Targets
# =============================================================================
#
# Local build targets for The Seed:
#   - iperf3 compilation for the current host
#   - Frontend build (React/Vite)
#   - Backend build (Go with embedded assets)
#
# GitHub Actions owns cross-platform artifacts, provenance, and checksums.
#
# =============================================================================

.PHONY: build build-iperf3 build-iperf3-quiet \
        frontend-deps generate-types schema build-frontend build-frontend-quiet \
        build-backend build-backend-quiet build-backend-dev \
        run dev dev-frontend

# =============================================================================
# Main Build Target
# =============================================================================

# Build complete application (frontend embedded in Go binary)
build: ## Build current host binary with embedded frontend
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

generate-types: frontend-deps ## Generate TypeScript types from docs/schemas/api/*.json
	@echo "🔧 Generating TypeScript types from schema..."
	@cd ui && npm run gen-types
	@echo "✅ TypeScript types generated"

schema: ## Regenerate docs/schemas/api/*.json from internal/api Go DTOs
	@printf "$(BOLD)Generating JSON Schemas for API DTOs...$(RESET)\n"
	@go run ./cmd/seed-schema -o docs/schemas/api
	@printf "$(GREEN)Wrote $$(ls -1 docs/schemas/api/*.json 2>/dev/null | wc -l | tr -d ' ') schema(s) to docs/schemas/api/$(RESET)\n"

build-frontend: frontend-deps ## Build React frontend
	@printf "$(BOLD)🔨 Building frontend...$(RESET)\n"
	@cd ui && npm run build
	@printf "$(GREEN)✓ Frontend build complete$(RESET)\n"

build-frontend-quiet: frontend-deps
	@printf "   Bundling React application...\n"
	@command -v npm >/dev/null 2>&1 || { printf "$(RED)ERROR: npm not found. Frontend build requires npm.$(RESET)\n"; exit 1; }
	@cd ui && npm run build
	@SIZE=$$(du -sh internal/api/ui 2>/dev/null | cut -f1 || echo "unknown"); \
	printf "   Output: internal/api/ui ($$SIZE)\n"

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
