# =============================================================================
# Container Build Targets (Cloud Native Buildpacks)
# =============================================================================
#
# Build OCI-compliant container images using `pack` (Cloud Native Buildpacks).
# This replaces Docker-based builds with a more standardized approach.
#
# Requirements:
#   - pack CLI: https://buildpacks.io/docs/tools/pack/
#   - Install: brew install buildpacks/tap/pack
#
# =============================================================================

.PHONY: container container-build container-run container-push container-clean

# Container configuration
CONTAINER_IMAGE ?= seed
CONTAINER_TAG ?= $(VERSION)
CONTAINER_REGISTRY ?= ghcr.io/krisarmstrong

# =============================================================================
# Main Container Targets
# =============================================================================

container: container-build ## Build container image using pack (alias)

container-build: ## Build container image using Cloud Native Buildpacks
	@printf "$(BOLD)$(CYAN)Building container with pack...$(RESET)\n"
	@command -v pack >/dev/null 2>&1 || { \
		printf "$(RED)ERROR: pack CLI not found$(RESET)\n"; \
		printf "Install: brew install buildpacks/tap/pack\n"; \
		exit 1; \
	}
	@pack build $(CONTAINER_IMAGE):$(CONTAINER_TAG) \
		--builder paketobuildpacks/builder-jammy-base \
		--trust-builder \
		--env BP_GO_TARGETS="./cmd/seed" \
		--env BP_GO_BUILD_LDFLAGS="-s -w -X github.com/krisarmstrong/seed/internal/version.Version=$(VERSION)"
	@printf "$(GREEN)✓ Container built: $(CONTAINER_IMAGE):$(CONTAINER_TAG)$(RESET)\n"

container-run: ## Run container locally
	@printf "$(CYAN)Running $(CONTAINER_IMAGE):$(CONTAINER_TAG)...$(RESET)\n"
	docker run --rm -it \
		-p 8443:8443 \
		--name seed-local \
		$(CONTAINER_IMAGE):$(CONTAINER_TAG)

container-push: container-build ## Push container to registry
	@if [ -z "$(CONTAINER_REGISTRY)" ]; then \
		printf "$(RED)ERROR: CONTAINER_REGISTRY not set$(RESET)\n"; \
		exit 1; \
	fi
	@printf "$(CYAN)Tagging and pushing to $(CONTAINER_REGISTRY)...$(RESET)\n"
	docker tag $(CONTAINER_IMAGE):$(CONTAINER_TAG) $(CONTAINER_REGISTRY)/$(CONTAINER_IMAGE):$(CONTAINER_TAG)
	docker tag $(CONTAINER_IMAGE):$(CONTAINER_TAG) $(CONTAINER_REGISTRY)/$(CONTAINER_IMAGE):latest
	docker push $(CONTAINER_REGISTRY)/$(CONTAINER_IMAGE):$(CONTAINER_TAG)
	docker push $(CONTAINER_REGISTRY)/$(CONTAINER_IMAGE):latest
	@printf "$(GREEN)✓ Pushed to $(CONTAINER_REGISTRY)/$(CONTAINER_IMAGE)$(RESET)\n"

container-clean: ## Remove local container images
	@printf "$(CYAN)Removing container images...$(RESET)\n"
	-docker rmi $(CONTAINER_IMAGE):$(CONTAINER_TAG) 2>/dev/null
	-docker rmi $(CONTAINER_IMAGE):latest 2>/dev/null
	-docker rmi $(CONTAINER_REGISTRY)/$(CONTAINER_IMAGE):$(CONTAINER_TAG) 2>/dev/null
	@printf "$(GREEN)✓ Container images removed$(RESET)\n"
