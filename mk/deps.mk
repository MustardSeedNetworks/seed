# =============================================================================
# Dependency Management & Tools
# =============================================================================
#
# Dependency and tooling management:
#   - Go modules and npm package management
#   - Development tools installation
#   - Version checking and updates
#   - i18n locale management
#
# =============================================================================

.PHONY: deps update update-go update-go-quiet update-npm \
        version-check outdated \
        tools tools-go tools-go-quiet tools-frontend \
        i18n-sync i18n-check i18n-list

# =============================================================================
# Dependencies
# =============================================================================

deps: ## Install dependencies
	go mod download
	cd ui && npm ci
	npm ci

# =============================================================================
# Dependency Updates
# =============================================================================

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
	@cd ui && npm update 2>&1 | tail -3 || true
	$(call timer-end,update-npm-web,npm web)
	@printf "$(CYAN)│$(RESET) $(BOLD)[4/5]$(RESET) Go development tools                                                  $(CYAN)│$(RESET)\n"
	$(call timer-start,update-tools)
	@$(MAKE) --no-print-directory tools-go-quiet
	$(call timer-end,update-tools,Go tools)
	@printf "$(CYAN)│$(RESET) $(BOLD)[5/5]$(RESET) Playwright browsers                                                   $(CYAN)│$(RESET)\n"
	$(call timer-start,update-playwright)
	@cd ui && npx playwright install chromium 2>&1 | tail -2 || true
	$(call timer-end,update-playwright,Playwright)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "\n$(GREEN)✓ All dependencies updated!$(RESET)\n"
	@printf "Run '$(BOLD)make version-check$(RESET)' to verify versions\n\n"

update-go: ## Update Go dependencies
	@printf "$(BOLD)📦 Updating Go modules...$(RESET)\n"
	@go get -u ./...
	@go mod tidy
	@printf "$(GREEN)✓ Go modules updated$(RESET)\n"

update-go-quiet:
	@printf "   Fetching latest versions...\n"
	@go get -u ./... 2>&1 | tail -5 || true
	@go mod tidy
	@MOD_COUNT=$$(go list -m all | wc -l | tr -d ' '); \
	printf "   $$MOD_COUNT modules in go.mod\n"

update-npm: ## Update npm dependencies
	@printf "$(BOLD)📦 Updating npm packages...$(RESET)\n"
	@npm update
	@cd ui && npm update
	@printf "$(GREEN)✓ npm packages updated$(RESET)\n"

# =============================================================================
# Version Checking
# =============================================================================

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
	@printf "  npm (web):     %s packages\n" "$$(cd ui && npm ls --depth=0 2>/dev/null | wc -l | tr -d ' ')"
	@printf "\n$(BOLD)Outdated Packages:$(RESET)\n"
	@GO_OUTDATED=$$(go list -u -m all 2>/dev/null | grep '\[' | wc -l | tr -d ' '); \
	printf "  Go:            %s can be updated\n" "$$GO_OUTDATED"
	@NPM_OUTDATED=$$(cd ui && npm outdated 2>/dev/null | wc -l | tr -d ' '); \
	printf "  npm:           %s can be updated\n" "$$NPM_OUTDATED"
	@printf "\n$(CYAN)═══════════════════════════════════════════════════════════════════════════════$(RESET)\n"

outdated: ## Show outdated packages (Go + npm)
	@printf "$(BOLD)$(CYAN)┌─ Outdated Go Modules ────────────────────────────────────────────────────────┐$(RESET)\n"
	@go list -u -m all 2>/dev/null | grep '\[' | head -20 || printf "   All Go modules are up to date\n"
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "\n$(BOLD)$(CYAN)┌─ Outdated npm Packages (web) ───────────────────────────────────────────────┐$(RESET)\n"
	@cd ui && npm outdated 2>/dev/null || printf "   All npm packages are up to date\n"
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# =============================================================================
# Development Tools Installation
# =============================================================================

tools: tools-go tools-frontend ## Install all development tools (Go + Node)
	@printf "\n$(GREEN)✓ All development tools installed!$(RESET)\n\n"
	@printf "Go tools: $$(go env GOPATH)/bin\n"
	@printf "Ensure $$(go env GOPATH)/bin is in your PATH\n"

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

tools-frontend: ## Install frontend development tools
	@printf "\n$(BOLD)=== Installing Frontend Development Tools ===$(RESET)\n"
	@printf "Installing root npm dependencies (prettier, husky)...\n"
	@npm ci
	@printf "Installing ui dependencies...\n"
	@cd ui && npm ci
	@printf "Installing Playwright browsers...\n"
	@cd ui && npx playwright install --with-deps chromium 2>/dev/null || printf "Playwright install skipped (run 'make test-e2e-install' manually)\n"
	@echo "✅ Frontend tools installed"

# =============================================================================
# Internationalization (i18n)
# =============================================================================

i18n-sync: ## Sync locale files for Go embedding
	@echo "Syncing locale files..."
	@mkdir -p internal/i18n/locales/en internal/i18n/locales/es
	@cp locales/en/*.json internal/i18n/locales/en/
	@cp locales/es/*.json internal/i18n/locales/es/
	@echo "✅ Locale files synced to internal/i18n/locales/"

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

i18n-list: ## List all translation keys in English locale
	@for file in locales/en/*.json; do \
		echo "=== $$(basename $$file) ==="; \
		jq -r 'paths(scalars) | join(".")' $$file; \
	done
