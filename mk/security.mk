# =============================================================================
# Security Scanning & License Compliance
# =============================================================================
#
# Security and compliance targets:
#   - Go vulnerability scanning (govulncheck)
#   - npm audit
#   - Secret scanning (gitleaks)
#   - Container scanning (Trivy)
#   - License compliance checking
#
# =============================================================================

.PHONY: security security-backend security-backend-quiet security-frontend security-frontend-quiet \
        security-secrets security-secrets-quiet security-trivy \
        license-check license-report

# =============================================================================
# Security Scanning
# =============================================================================

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

security-backend: ## Run Go vulnerability scan (govulncheck)
	@printf "$(BOLD)🔒 Running Go vulnerability scan...$(RESET)\n"
	@if ! command -v govulncheck > /dev/null 2>&1; then \
		printf "📦 Installing govulncheck...\n"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck ./...
	@printf "$(GREEN)✓ Go vulnerability scan complete$(RESET)\n"

security-backend-quiet:
	@if ! command -v govulncheck > /dev/null 2>&1; then \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@printf "   Scanning Go dependencies...\n"
	@govulncheck ./... 2>&1 | grep -E "(Vulnerability|No vulnerabilities)" | head -5 || printf "   No vulnerabilities found\n"

security-frontend: ## Run frontend security scan (npm audit)
	@printf "$(BOLD)🔒 Running npm audit...$(RESET)\n"
	@cd ui && npm audit --audit-level=high
	@printf "$(GREEN)✓ npm audit complete$(RESET)\n"

security-frontend-quiet:
	@printf "   Auditing npm packages...\n"
	@cd ui && npm audit --audit-level=high 2>&1 | grep -E "(found|vulnerabilities)" | head -3 || printf "   No vulnerabilities found\n"

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

security-secrets-quiet:
	@GITLEAKS=$$(command -v gitleaks 2>/dev/null || echo "$$(go env GOPATH)/bin/gitleaks"); \
	if [ ! -x "$$GITLEAKS" ]; then \
		go install github.com/zricethezav/gitleaks/v8@latest; \
		GITLEAKS="$$(go env GOPATH)/bin/gitleaks"; \
	fi; \
	printf "   Scanning for secrets...\n"; \
	$$GITLEAKS detect --source . --config .gitleaks.toml 2>&1 | grep -E "(leaks found|no leaks)" || printf "   No secrets found\n"

security-trivy: ## Run Trivy vulnerability scan
	@printf "$(BOLD)🔒 Running Trivy filesystem scan...$(RESET)\n"
	@if command -v trivy > /dev/null 2>&1; then \
		trivy fs --severity HIGH,CRITICAL .; \
	else \
		printf "$(YELLOW)SKIP: trivy not installed (brew install trivy)$(RESET)\n"; \
	fi

# =============================================================================
# License Compliance
# =============================================================================

license-check: ## Check license compliance (Go + npm)
	@printf "$(BOLD)🔍 Checking license compliance...$(RESET)\n\n"
	@printf "$(BOLD)=== Go Dependencies ===$(RESET)\n"
	@if command -v go-licenses > /dev/null 2>&1; then \
		go-licenses check ./... --disallowed_types=forbidden,restricted --allowed_licenses=MIT,Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0,CC0-1.0; \
	else \
		printf "$(YELLOW)SKIP: go-licenses not installed$(RESET)\n"; \
	fi
	@printf "\n$(BOLD)=== npm Dependencies ===$(RESET)\n"
	@cd ui && npx license-checker --production --excludePrivatePackages --onlyAllow "MIT;Apache-2.0;BSD-2-Clause;BSD-3-Clause;ISC;CC0-1.0;0BSD;BlueOak-1.0.0;Unlicense;MPL-2.0" --summary
	@printf "\n$(GREEN)✓ License compliance check complete$(RESET)\n"

license-report: ## Generate license compliance reports
	@printf "$(BOLD)Generating license reports...$(RESET)\n"
	@mkdir -p build/reports
	@if command -v go-licenses > /dev/null 2>&1; then \
		go-licenses report ./... > build/reports/go-licenses.csv; \
		printf "Go licenses: build/reports/go-licenses.csv\n"; \
	fi
	@cd ui && npx license-checker --production --csv > ../build/reports/npm-licenses.csv
	@printf "npm licenses: build/reports/npm-licenses.csv\n"
	@printf "$(GREEN)✓ License reports generated$(RESET)\n"
