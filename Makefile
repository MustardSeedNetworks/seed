# =============================================================================
# Seed Makefile
# =============================================================================
#
# Local build, test, and development automation for Seed network diagnostics tool.
#
# QUICK START
# -----------
#   make build          Build current-host binary (frontend + backend)
#   make test           Run all unit tests (backend + frontend)
#   make verify         Full local verification (lint, test, security, build)
#   make dev            Run backend in dev mode (hot reload frontend)
#   make help           Show all available targets
#
# COMMON WORKFLOWS
# ----------------
#   Development:        make dev & make dev-frontend
#   Before commit:      make verify
#   Release artifacts:  built by GitHub Actions on tag/release
#
# REQUIREMENTS
# ------------
#   - Go 1.25.5+ (with CGO for libpcap)
#   - Node.js 26.2.0+ and npm
#   - libpcap-dev (Linux) or libpcap (macOS via Homebrew)
#
# =============================================================================

# =============================================================================
# Version and Build Information
# =============================================================================

# Version information (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Platform detection
UNAME := $(shell uname -s)
ifeq ($(UNAME),Darwin)
    PLATFORM := darwin
    PLATFORM_PRETTY := macOS
else ifeq ($(UNAME),Linux)
    PLATFORM := linux
    PLATFORM_PRETTY := Linux
else
    PLATFORM := unknown
    PLATFORM_PRETTY := Unknown
endif

# Architecture detection
ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
    GOARCH := amd64
else ifeq ($(ARCH),arm64)
    GOARCH := arm64
else ifeq ($(ARCH),aarch64)
    GOARCH := arm64
endif

# =============================================================================
# ANSI Color Codes
# =============================================================================

BOLD := \033[1m
RESET := \033[0m
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
WHITE := \033[37m

# =============================================================================
# Display Helpers
# =============================================================================

# Print a section header
define section
	@printf "\n$(BOLD)$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)\n"
	@printf "$(BOLD)$(CYAN)  $(1)$(RESET)\n"
	@printf "$(CYAN)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)\n\n"
endef

# Print a step in a multi-step process
define step
	@printf "$(BOLD)[$(1)/$(2)]$(RESET) $(3)\n"
endef

# Print a success message
define success
	@printf "$(GREEN)✓ $(1)$(RESET)\n"
endef

# Print a warning message
define warn
	@printf "$(YELLOW)⚠ $(1)$(RESET)\n"
endef

# Print an error message
define error
	@printf "$(RED)✗ $(1)$(RESET)\n"
endef

# =============================================================================
# Timer Functions
# =============================================================================

# Start a named timer
define timer-start
	@date +%s > /tmp/make-timer-$(1)
endef

# End a timer and display elapsed time
define timer-end
	@if [ -f /tmp/make-timer-$(1) ]; then \
		START=$$(cat /tmp/make-timer-$(1)); \
		END=$$(date +%s); \
		ELAPSED=$$((END - START)); \
		MINS=$$((ELAPSED / 60)); \
		SECS=$$((ELAPSED % 60)); \
		if [ $$MINS -gt 0 ]; then \
			printf "$(GREEN)✓ $(2) $(YELLOW)($$MINS min $$SECS sec)$(RESET)\n"; \
		else \
			printf "$(GREEN)✓ $(2) $(YELLOW)($$SECS sec)$(RESET)\n"; \
		fi; \
		rm -f /tmp/make-timer-$(1); \
	fi
endef

# =============================================================================
# Configuration Variables
# =============================================================================

# Application name
BINARY_NAME=seed

# Version package path for ldflags injection
VERSION_PKG=github.com/MustardSeedNetworks/seed/internal/version

# Go build flags for reproducible builds
GO_BUILD_FLAGS := -trimpath -buildvcs=false
GOFLAGS=$(GO_BUILD_FLAGS)

# Standard linker flags with version injection
GO_LDFLAGS = -s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).BuildTime=$(BUILD_TIME)
LDFLAGS=$(GO_LDFLAGS)

# =============================================================================
# CGO Configuration
# =============================================================================
# CGO_ENABLED controls whether C code can be compiled and linked.
#
# CGO_ENABLED=1 (default on native builds):
#   - Required for libpcap (packet capture functionality)
#   - Binaries are dynamically linked to system libraries
#
# CGO_ENABLED=0 (for static/portable builds):
#   - Creates fully static binaries (no external dependencies)
#   - Used by GitHub release builds for portable artifacts
#   - Disables libpcap (packet capture won't work)
# =============================================================================

# =============================================================================
# Include Domain-Specific Makefiles
# =============================================================================

include mk/build.mk
include mk/test.mk
include mk/lint.mk
include mk/security.mk
include mk/deps.mk
include mk/dev.mk

# =============================================================================
# Default Target
# =============================================================================

all: verify ## Full local verification

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean clean-all

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	find internal/api/ui -mindepth 1 ! -name .gitkeep -exec rm -rf {} +

clean-all: clean ## Clean everything including dependencies
	rm -rf ui/node_modules
	rm -rf build/iperf3 bin/iperf3*
	rm -rf dist/

# =============================================================================
# Version Information
# =============================================================================

.PHONY: version

version: ## Show version info (current build and installed)
	@printf "$(BOLD)Seed Version Information$(RESET)\n"
	@printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
	@printf "  Version:     $(VERSION)\n"
	@printf "  Commit:      $(COMMIT)\n"
	@printf "  Build Time:  $(BUILD_TIME)\n"
	@printf "  Platform:    $(PLATFORM_PRETTY) ($(PLATFORM)/$(GOARCH))\n"
	@printf "  Go:          $$(go version | awk '{print $$3}')\n"
	@printf "  Node:        $$(node --version 2>/dev/null || echo 'not installed')\n"
	@if [ -f "./$(BINARY_NAME)" ]; then \
		printf "\n$(BOLD)Binary:$(RESET)\n"; \
		ls -lh ./$(BINARY_NAME); \
	fi
	@if command -v $(BINARY_NAME) > /dev/null 2>&1; then \
		printf "\n$(BOLD)Installed:$(RESET)\n"; \
		which $(BINARY_NAME); \
	fi

# =============================================================================
# Help
# =============================================================================

.PHONY: help

help: ## Show this help
	@echo "The Seed - Network Diagnostics Tool by Mustard Seed Networks"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) 2>/dev/null | sort -u | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build                    Build current-host binary"
	@echo "  make dev & make dev-frontend  Full development environment"

# =============================================================================
# Verification & Release
# =============================================================================

.PHONY: verify pre-commit pre-commit-install

verify: ## Full local verification (lint, test, security, build)
	@printf "\n$(BOLD)$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(BOLD)$(CYAN)║                        FULL VERIFICATION PIPELINE                           ║$(RESET)\n"
	@printf "$(BOLD)$(CYAN)║                        Version: $(VERSION)$(RESET)\n"
	@printf "$(BOLD)$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	$(call timer-start,verify-total)
	$(call step,1,4,Linting Code)
	$(call timer-start,lint)
	@$(MAKE) --no-print-directory lint
	$(call timer-end,lint,Linting)
	$(call step,2,4,Running Tests)
	$(call timer-start,test)
	@$(MAKE) --no-print-directory test
	$(call timer-end,test,Tests)
	$(call step,3,4,Security Scanning)
	$(call timer-start,security)
	@$(MAKE) --no-print-directory security
	$(call timer-end,security,Security)
	$(call step,4,4,Building Application)
	$(call timer-start,build)
	@$(MAKE) --no-print-directory build
	$(call timer-end,build,Build)
	@printf "\n$(BOLD)$(GREEN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(BOLD)$(GREEN)║                        ✓ VERIFICATION COMPLETE                               ║$(RESET)\n"
	@printf "$(BOLD)$(GREEN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	$(call timer-end,verify-total,Total verification)
	@printf "\n  $(BOLD)Version:$(RESET)     $(VERSION)\n"
	@printf "  $(BOLD)Commit:$(RESET)      $(COMMIT)\n"
	@printf "  $(BOLD)Binary:$(RESET)      $(BINARY_NAME)\n\n"
	@printf "$(GREEN)Local verification complete. GitHub Actions owns release artifacts.$(RESET)\n\n"

pre-commit: ## Run pre-commit hooks manually
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "pre-commit not installed. Install with: pip install pre-commit"; \
		exit 1; \
	fi

pre-commit-install: ## Install pre-commit hooks
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit install; \
		pre-commit install --hook-type pre-push; \
		echo "Pre-commit hooks installed successfully"; \
	else \
		echo "pre-commit not installed. Install with: pip install pre-commit"; \
		exit 1; \
	fi
