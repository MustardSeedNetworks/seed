# =============================================================================
# Package Creation Targets (goreleaser snapshot)
# =============================================================================
# Canonical local packaging. Invokes the locally-installed goreleaser binary
# with --snapshot mode using the SAME .goreleaser.yml that CI uses, so local
# artifacts are byte-for-byte the same shape as CI publishes (just unsigned).
#
# Platform coverage from local macOS depends on what cross-compile toolchains
# are installed. For the full multi-arch matrix (linux + windows + darwin
# with CGO and libpcap), use the dev servers or trigger release.yml via
# workflow_dispatch (which runs inside goreleaser-cross).
#
# Outputs land in dist/ — archives, .deb, .rpm, checksums, sbom stubs.
# =============================================================================

.PHONY: deb rpm pkg windows-zip packages packages-all ensure-goreleaser \
        container \
        deploy-validate deploy-ubuntu deploy-fedora deploy-all

# =============================================================================
# Tooling check
# =============================================================================

ensure-goreleaser:
	@command -v goreleaser >/dev/null 2>&1 || { \
		printf "$(RED)ERROR: goreleaser not installed.$(RESET)\n"; \
		printf "Install with:\n"; \
		printf "  macOS:  brew install goreleaser\n"; \
		printf "  Linux:  go install github.com/goreleaser/goreleaser/v2@latest\n"; \
		exit 1; \
	}

# =============================================================================
# Canonical packaging — one command builds all release artifacts
# =============================================================================
# --snapshot:       version stamp uses dryrun pattern, no git tag required
# --clean:          wipe dist/ first
# --skip=publish:   no GitHub release upload
# --skip=sign:      no cosign signing (cosign keyless requires GitHub OIDC)
# --skip=validate:  tolerate dirty working tree (UI build writes into
#                   internal/api/ui/ which is in-tree)
# --skip=announce:  no notifications
# =============================================================================

packages: build ensure-goreleaser ## Build all release artifacts (goreleaser snapshot)
	@printf "$(BOLD)Building release artifacts via goreleaser snapshot...$(RESET)\n"
	@UI_BUILD_HASH=$(UI_BUILD_HASH) goreleaser release --snapshot --clean \
		--skip=publish,sign,validate,announce
	@printf "$(GREEN)Artifacts in dist/:$(RESET)\n"
	@ls -1 dist/ 2>/dev/null | grep -E '\.(tar\.gz|zip|deb|rpm)$$' | sed 's/^/  /' || true

# Aliases — one config produces everything from a single goreleaser run.
deb: packages         ## Alias for 'packages' (goreleaser produces all formats)
rpm: packages         ## Alias for 'packages'
windows-zip: packages ## Alias for 'packages'
packages-all: packages ## Alias for 'packages'

# =============================================================================
# macOS .pkg — custom script (goreleaser doesn't produce .pkg)
# =============================================================================

pkg: build-darwin ## Build macOS installer package (.pkg)
	@if [ "$$(uname -s)" != "Darwin" ]; then \
		printf "$(RED)ERROR: macOS .pkg can only be built on macOS$(RESET)\n"; \
		exit 1; \
	fi
	@printf "$(BOLD)Building macOS .pkg package...$(RESET)\n"
	@./deploy/macos/build-pkg.sh ./$(BINARY_NAME)-darwin-$$(uname -m) $(VERSION)
	@printf "$(GREEN)macOS package: dist/seed-$(VERSION)-$$(uname -m | sed 's/x86_64/amd64/').pkg$(RESET)\n"

# =============================================================================
# Container Images (Pack/Buildpacks) - LOCAL DEV ONLY
# =============================================================================
# NOTE: CI uses Dockerfile + buildx (see .github/workflows/container.yml).
# This Pack target is a developer convenience and does not push to a registry.

CONTAINER_IMAGE := seed

container: ## Build container image locally (Pack/Buildpacks)
	@printf "$(BOLD)Building container with Pack (local only)...$(RESET)\n"
	@pack build $(CONTAINER_IMAGE):$(VERSION) \
		--builder paketobuildpacks/builder-jammy-base \
		--env BP_GO_TARGETS="./cmd/seed" \
		--env BP_GO_BUILD_LDFLAGS="-s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).BuildTime=$(BUILD_TIME) -X $(VERSION_PKG).UIBuildHash=$(UI_BUILD_HASH)"
	@printf "$(GREEN)Container: $(CONTAINER_IMAGE):$(VERSION) (local)$(RESET)\n"

# =============================================================================
# Deployment Targets
# =============================================================================
# Mirrors niac/go's deploy block so the three projects share a deployment
# contract. Honors the Universal Build Contract in CLAUDE.md.
#
# Prerequisites:
#   - SSH access to target servers (key-based auth recommended)
#   - Packages built with 'make packages'
#   - Target servers configured in ~/.ssh/config
# =============================================================================

DEPLOY_UBUNTU_HOST ?= niac-srv-ubuntu
DEPLOY_FEDORA_HOST ?= niac-srv-fedora
DEPLOY_PORT ?= 8443

deploy-validate: ## Validate deployment on a remote host (HOST=hostname)
	@if [ -z "$(HOST)" ]; then \
		printf "$(RED)ERROR: HOST is required. Usage: make deploy-validate HOST=hostname$(RESET)\n"; \
		exit 1; \
	fi
	@printf "$(BOLD)Validating deployment on $(HOST)...$(RESET)\n"
	./scripts/deploy-validate.sh $(VERSION) $(COMMIT) $(HOST) $(DEPLOY_PORT)

deploy-ubuntu: packages ## Deploy .deb to Ubuntu test server and validate
	@printf "$(BOLD)Deploying to $(DEPLOY_UBUNTU_HOST)...$(RESET)\n"
	@DEB_FILE=$$(ls -t dist/seed_*_amd64.deb 2>/dev/null | head -1); \
	if [ -z "$$DEB_FILE" ]; then \
		printf "$(RED)ERROR: No .deb file found in dist/. Run 'make packages' first.$(RESET)\n"; \
		exit 1; \
	fi; \
	printf "  Uploading $$DEB_FILE...\n"; \
	scp "$$DEB_FILE" $(DEPLOY_UBUNTU_HOST):/tmp/seed.deb; \
	printf "  Installing package...\n"; \
	ssh $(DEPLOY_UBUNTU_HOST) 'sudo dpkg -i /tmp/seed.deb'; \
	printf "  Restarting service...\n"; \
	ssh $(DEPLOY_UBUNTU_HOST) 'sudo systemctl restart seed || sudo systemctl start seed'; \
	printf "  Waiting for service startup...\n"; \
	sleep 3; \
	printf "  Validating deployment...\n"
	@./scripts/deploy-validate.sh $(VERSION) $(COMMIT) $(DEPLOY_UBUNTU_HOST) $(DEPLOY_PORT)
	@printf "$(GREEN)✓ Ubuntu deployment complete and validated$(RESET)\n"

deploy-fedora: packages ## Deploy .rpm to Fedora test server and validate
	@printf "$(BOLD)Deploying to $(DEPLOY_FEDORA_HOST)...$(RESET)\n"
	@RPM_FILE=$$(ls -t dist/seed-*-1.*.rpm 2>/dev/null | head -1); \
	if [ -z "$$RPM_FILE" ]; then \
		printf "$(RED)ERROR: No .rpm file found in dist/. Run 'make packages' first.$(RESET)\n"; \
		exit 1; \
	fi; \
	printf "  Uploading $$RPM_FILE...\n"; \
	scp "$$RPM_FILE" $(DEPLOY_FEDORA_HOST):/tmp/seed.rpm; \
	printf "  Installing package...\n"; \
	ssh $(DEPLOY_FEDORA_HOST) 'sudo rpm -Uvh --nodeps /tmp/seed.rpm || sudo rpm -ivh --nodeps /tmp/seed.rpm'; \
	printf "  Restarting service...\n"; \
	ssh $(DEPLOY_FEDORA_HOST) 'sudo systemctl restart seed || sudo systemctl start seed'; \
	printf "  Waiting for service startup...\n"; \
	sleep 3; \
	printf "  Validating deployment...\n"
	@./scripts/deploy-validate.sh $(VERSION) $(COMMIT) $(DEPLOY_FEDORA_HOST) $(DEPLOY_PORT)
	@printf "$(GREEN)✓ Fedora deployment complete and validated$(RESET)\n"

deploy-all: deploy-ubuntu deploy-fedora ## Deploy to all test servers
	@printf "\n$(GREEN)✓ ALL DEPLOYMENTS VALIDATED$(RESET)\n"
