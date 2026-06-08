# =============================================================================
# Test Targets
# =============================================================================
#
# All testing targets:
#   - Unit tests (backend + frontend)
#   - E2E tests (Playwright)
#   - Coverage reports
#
# =============================================================================

.PHONY: test test-all test-backend test-backend-quiet test-fast test-frontend test-frontend-quiet \
        test-e2e test-e2e-ui test-e2e-install test-coverage

# =============================================================================
# Main Test Targets
# =============================================================================

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

test-all: ## Run ALL tests (unit + E2E)
	@printf "$(BOLD)$(CYAN)┌─ Full Test Suite ────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/3]$(RESET) Backend unit tests                                                    $(CYAN)│$(RESET)\n"
	$(call timer-start,test-backend)
	@$(MAKE) --no-print-directory test-backend-quiet
	$(call timer-end,test-backend,Backend tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/3]$(RESET) Frontend unit tests                                                   $(CYAN)│$(RESET)\n"
	$(call timer-start,test-frontend)
	@$(MAKE) --no-print-directory test-frontend-quiet
	$(call timer-end,test-frontend,Frontend tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/3]$(RESET) E2E tests (Playwright)                                                $(CYAN)│$(RESET)\n"
	$(call timer-start,test-e2e)
	@$(MAKE) --no-print-directory test-e2e
	$(call timer-end,test-e2e,E2E tests)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# =============================================================================
# Backend Tests
# =============================================================================

test-backend: ## Run Go tests with progress
	@printf "\n$(BOLD)🧪 Running backend tests...$(RESET)\n"
	@PKGS=$$(go list ./... | grep -v '/cmd/' | grep -v '/ui$$' | grep -v '/i18n$$' | grep -v '/mcp$$' | grep -v '/oauth$$'); \
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

test-backend-quiet:
	@PKGS=$$(go list ./... | grep -v '/cmd/' | grep -v '/ui$$' | grep -v '/i18n$$' | grep -v '/mcp$$' | grep -v '/oauth$$'); \
	PKG_COUNT=$$(echo "$$PKGS" | wc -l | tr -d ' '); \
	printf "   Testing $$PKG_COUNT packages...\n"; \
	go test -race -coverprofile=coverage.out $$PKGS 2>&1 | grep -E "^(ok|FAIL|---)" || true
	@if [ -f coverage.out ]; then \
		COV=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}'); \
		printf "   📊 Coverage: %s\n" "$$COV"; \
	fi

# test-fast is the libpcap-free inner loop. Local `make test` runs `go test
# -race`, which forces CGO=1; under CGO=1 internal/api compiles its
# wire_capture_real.go (//go:build cgo), pulling in internal/capture/pcap and
# thus requiring a system libpcap. CGO=0 selects the null-capture stub instead,
# so the backend suite runs with no libpcap and no C toolchain — much faster for
# iterating, at the cost of the race detector and the real-capture package
# (both still covered by `make test` and CI). The cgo-tagged pcap package has no
# buildable files under CGO=0, so it is excluded explicitly.
test-fast: ## Fast libpcap-free backend tests (CGO=0, null capture, no -race)
	@printf "\n$(BOLD)🏃 Fast backend tests (CGO=0, no libpcap, no -race)...$(RESET)\n"
	@PKGS=$$(go list ./... 2>/dev/null | grep -v '/cmd/' | grep -v '/ui$$' | grep -v '/i18n$$' | grep -v '/oauth$$' | grep -v '/capture/pcap$$'); \
	CGO_ENABLED=0 go test $$PKGS
	@printf "\n$(GREEN)✓ Fast tests complete — run 'make test' (or CI) for -race + real capture$(RESET)\n"

# =============================================================================
# Frontend Tests
# =============================================================================

test-frontend: ## Run frontend tests with progress
	@printf "\n$(BOLD)🧪 Running frontend tests...$(RESET)\n"
	@STORY_COUNT=$$(find ui/src -name "*.test.ts" -o -name "*.test.tsx" 2>/dev/null | wc -l | tr -d ' '); \
	printf "   📦 Running $$STORY_COUNT test files...\n\n"
	@cd ui && npm test
	@printf "\n$(GREEN)✓ Frontend tests complete$(RESET)\n"

test-frontend-quiet:
	@STORY_COUNT=$$(find ui/src -name "*.test.ts" -o -name "*.test.tsx" 2>/dev/null | wc -l | tr -d ' '); \
	printf "   Running $$STORY_COUNT test files...\n"
	@cd ui && npm test 2>&1 | grep -E "(PASS|FAIL|Tests:)" || true

# =============================================================================
# E2E Tests
# =============================================================================

test-e2e: ## Run frontend E2E tests (requires backend running)
	@echo ""
	@echo "🎭 Running E2E tests (Playwright)..."
	@E2E_COUNT=$$(find ui/e2e -name "*.spec.ts" 2>/dev/null | wc -l | tr -d ' '); \
	echo "   📦 Running $$E2E_COUNT spec files..."
	@echo ""
	@cd ui && npm run test:e2e
	@echo ""
	@echo "✅ E2E tests complete"

test-e2e-ui: ## Run E2E tests with Playwright UI
	@echo "🎭 Starting Playwright UI mode..."
	cd ui && npm run test:e2e:ui

test-e2e-install: ## Install Playwright browsers
	cd ui && npx playwright install --with-deps chromium

# =============================================================================
# Coverage & Integration
# =============================================================================

test-coverage: ## Generate coverage report
	@PKGS=$$(go list ./... | grep -v '/cmd/' | grep -v '/ui$$' | grep -v '/i18n$$' | grep -v '/mcp$$' | grep -v '/oauth$$'); \
	go test -race -coverprofile=coverage.out $$PKGS
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
