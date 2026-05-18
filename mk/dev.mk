# =============================================================================
# Dev-experience helpers — capability-aware run for local iteration
# =============================================================================
# Production installs (.deb/.rpm/.pkg) use systemd/launchd units that already
# declare CAP_NET_RAW + CAP_NET_ADMIN and run as a non-root user. For local
# dev (`make build && ./seed`) those capabilities have to be set on the
# binary OR you have to sudo. These targets handle that automatically per
# platform.
#
#   Linux: `sudo setcap` once on the binary, then run as user (no sudo)
#   macOS: `sudo` the binary (macOS userspace has no fcaps for raw sockets)
#
# After `make dev-run`, future direct invocations of the binary (until next
# rebuild that replaces the file) don't need sudo on Linux because the file
# caps stick.
# =============================================================================

.PHONY: dev-run dev-start dev-stop dev-status dev-restart

DEV_BINARY ?= ./seed
DEV_ARGS   ?=
DEV_PID    ?= /tmp/seed.pid
DEV_LOG    ?= /tmp/seed.log

DEV_UNAME := $(shell uname -s)

dev-run: build ## Build + run in foreground with required capabilities
ifeq ($(DEV_UNAME),Linux)
	@if ! getcap $(DEV_BINARY) 2>/dev/null | grep -q cap_net_raw; then \
		printf "$(BOLD)Setting capabilities on $(DEV_BINARY) (one-time per build)...$(RESET)\n"; \
		sudo setcap cap_net_raw,cap_net_admin+ep $(DEV_BINARY) || { \
			printf "$(YELLOW)setcap unavailable — falling back to sudo$(RESET)\n"; \
			exec sudo $(DEV_BINARY) $(DEV_ARGS); \
		}; \
	fi
	@printf "$(GREEN)→ $(DEV_BINARY) $(DEV_ARGS)$(RESET)\n"
	@$(DEV_BINARY) $(DEV_ARGS)
else
	@printf "$(BOLD)Running via sudo (macOS has no userspace fcaps for raw sockets)$(RESET)\n"
	@sudo $(DEV_BINARY) $(DEV_ARGS)
endif

dev-start: build ## Background variant of dev-run; PID -> $(DEV_PID), logs -> $(DEV_LOG)
ifeq ($(DEV_UNAME),Linux)
	@if ! getcap $(DEV_BINARY) 2>/dev/null | grep -q cap_net_raw; then \
		sudo setcap cap_net_raw,cap_net_admin+ep $(DEV_BINARY); \
	fi
	@nohup $(DEV_BINARY) $(DEV_ARGS) > $(DEV_LOG) 2>&1 & echo $$! > $(DEV_PID)
else
	@nohup sudo $(DEV_BINARY) $(DEV_ARGS) > $(DEV_LOG) 2>&1 & echo $$! > $(DEV_PID)
endif
	@sleep 2
	@printf "$(GREEN)✓ started (PID $$(cat $(DEV_PID))). log: $(DEV_LOG)$(RESET)\n"

dev-stop: ## Stop a backgrounded dev-start
	@if [ -f $(DEV_PID) ] && ps -p $$(cat $(DEV_PID)) > /dev/null 2>&1; then \
		kill $$(cat $(DEV_PID)) && rm -f $(DEV_PID); \
		printf "$(GREEN)✓ stopped$(RESET)\n"; \
	else \
		printf "$(YELLOW)not running$(RESET)\n"; \
		rm -f $(DEV_PID); \
	fi

dev-status: ## Show whether the backgrounded process is running
	@if [ -f $(DEV_PID) ] && ps -p $$(cat $(DEV_PID)) > /dev/null 2>&1; then \
		printf "$(GREEN)✓ running (PID $$(cat $(DEV_PID))), log: $(DEV_LOG)$(RESET)\n"; \
	else \
		printf "$(YELLOW)not running$(RESET)\n"; \
	fi

dev-restart: dev-stop dev-start ## Stop + start
