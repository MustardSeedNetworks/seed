# =============================================================================
# Linting & Formatting Targets
# =============================================================================
#
# Code quality and formatting:
#   - Linting (golangci-lint, Biome, markdownlint)
#   - Formatting (gofumpt, Biome, Prettier)
#   - Auto-fix capabilities
#
# =============================================================================

.PHONY: lint lint-backend lint-backend-quiet lint-frontend lint-frontend-quiet lint-md \
        fix fix-backend fix-backend-quiet fix-frontend fix-frontend-quiet fix-md fix-all \
        fmt fmt-frontend fmt-md fmt-all fmt-check

# =============================================================================
# Linting
# =============================================================================

lint: ## Run all linters
	@printf "$(BOLD)$(CYAN)┌─ Linting ────────────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Backend (golangci-lint)                                               $(CYAN)│$(RESET)\n"
	$(call timer-start,lint-backend)
	@$(MAKE) --no-print-directory lint-backend-quiet
	$(call timer-end,lint-backend,Backend lint)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) Frontend (Biome)                                                      $(CYAN)│$(RESET)\n"
	$(call timer-start,lint-frontend)
	@$(MAKE) --no-print-directory lint-frontend-quiet
	$(call timer-end,lint-frontend,Frontend lint)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

lint-backend: ## Run Go linter
	@printf "$(BOLD)🔍 Running backend linter...$(RESET)\n"
	@GOLANGCI_LINT="$$(go env GOPATH)/bin/golangci-lint"; \
	if [ ! -f "$$GOLANGCI_LINT" ]; then \
		printf "📦 Installing golangci-lint v2...\n"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi; \
	$$GOLANGCI_LINT run
	@printf "$(GREEN)✓ Backend lint complete$(RESET)\n"

lint-backend-quiet:
	@GOLANGCI_LINT="$$(go env GOPATH)/bin/golangci-lint"; \
	if [ ! -f "$$GOLANGCI_LINT" ]; then \
		printf "   Installing golangci-lint v2...\n"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi; \
	LINTER_COUNT=$$(grep -c "^    - " .golangci.yml 2>/dev/null || echo "30+"); \
	printf "   Running $$LINTER_COUNT linters...\n"; \
	$$GOLANGCI_LINT run 2>&1 | head -20 || true

lint-frontend: ## Run frontend linter (Biome)
	@printf "$(BOLD)🔍 Running frontend linter (Biome)...$(RESET)\n"
	@cd ui && npx @biomejs/biome check src/
	@printf "$(GREEN)✓ Frontend lint complete$(RESET)\n"

lint-frontend-quiet:
	@FILE_COUNT=$$(find ui/src -name "*.ts" -o -name "*.tsx" 2>/dev/null | wc -l | tr -d ' '); \
	printf "   Checking $$FILE_COUNT files...\n"
	@cd ui && npx @biomejs/biome check src/ 2>&1 | tail -5 || true

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
# Auto-Fix
# =============================================================================

fix: ## Auto-fix all linting issues (Go + Frontend)
	@printf "$(BOLD)$(CYAN)┌─ Auto-Fix ───────────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Backend (golangci-lint --fix)                                         $(CYAN)│$(RESET)\n"
	$(call timer-start,fix-backend)
	@$(MAKE) --no-print-directory fix-backend-quiet
	$(call timer-end,fix-backend,Backend fix)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) Frontend (biome check --fix)                                          $(CYAN)│$(RESET)\n"
	$(call timer-start,fix-frontend)
	@$(MAKE) --no-print-directory fix-frontend-quiet
	$(call timer-end,fix-frontend,Frontend fix)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

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

fix-backend-quiet:
	@GOLANGCI_LINT="$$(go env GOPATH)/bin/golangci-lint"; \
	if [ ! -f "$$GOLANGCI_LINT" ]; then \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi; \
	$$GOLANGCI_LINT run --fix 2>&1 | grep -E "^[0-9]+ issues" || printf "   No issues found\n"
	@gofmt -w -s .

fix-frontend: ## Auto-fix frontend linting issues
	@printf "$(BOLD)🔧 Auto-fixing frontend code...$(RESET)\n"
	@cd ui && npx @biomejs/biome check --write .
	@printf "$(GREEN)✓ Frontend auto-fix complete$(RESET)\n"

fix-frontend-quiet:
	@cd ui && npx @biomejs/biome check --write . 2>&1 | tail -3 || true

fix-md: fmt-md ## Auto-fix markdown formatting

fix-all: fix fix-md ## Auto-fix all code and documentation

# =============================================================================
# Formatting
# =============================================================================

fmt: ## Format Go code with gofumpt
	@if ! command -v gofumpt > /dev/null 2>&1; then \
		echo "📦 Installing gofumpt..."; \
		go install mvdan.cc/gofumpt@latest; \
	fi
	@gofumpt -w .
	@echo "✅ Go code formatted"

fmt-frontend: ## Format frontend code with Biome
	@cd ui && npx @biomejs/biome format --write src/
	@echo "✅ Frontend code formatted"

fmt-md: ## Format markdown and JSON files with Biome
	@cd ui && npx @biomejs/biome format --write ..
	@echo "✅ Markdown/JSON files formatted"

fmt-all: fmt fmt-frontend fmt-md ## Format all code (Go + frontend + markdown)

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
	echo "Checking frontend formatting (Biome)..."; \
	TEMPDIR=$$(mktemp -d); \
	cp -r ui/src "$$TEMPDIR/"; \
	(cd ui && npx @biomejs/biome format --write "$$TEMPDIR/src/" 2>/dev/null > /dev/null); \
	if diff -rq ui/src/ "$$TEMPDIR/src/" > /dev/null 2>&1; then \
		echo "✅ Frontend formatting OK"; \
	else \
		echo "❌ Frontend files need formatting"; \
		FAILED=1; \
	fi; \
	rm -rf "$$TEMPDIR"; \
	echo "Checking markdown/JSON formatting..."; \
	TEMPDIR=$$(mktemp -d); \
	for f in README.md CONTRIBUTING.md SECURITY.md; do \
		[ -f "$$f" ] && cp "$$f" "$$TEMPDIR/" || true; \
	done; \
	(cd ui && npx @biomejs/biome format --write "$$TEMPDIR/" 2>/dev/null > /dev/null); \
	CHANGED=0; \
	for f in README.md CONTRIBUTING.md SECURITY.md; do \
		if [ -f "$$f" ] && [ -f "$$TEMPDIR/$$f" ]; then \
			if ! diff -q "$$f" "$$TEMPDIR/$$f" > /dev/null 2>&1; then \
				CHANGED=1; \
			fi; \
		fi; \
	done; \
	if [ $$CHANGED -eq 0 ]; then \
		echo "✅ Markdown formatting OK"; \
	else \
		echo "❌ Markdown files need formatting"; \
		FAILED=1; \
	fi; \
	rm -rf "$$TEMPDIR"; \
	if [ $$FAILED -ne 0 ]; then \
		echo ""; \
		echo "Run 'make fmt-all' to fix formatting issues"; \
		exit 1; \
	fi
	@echo "✅ All formatting checks passed"
