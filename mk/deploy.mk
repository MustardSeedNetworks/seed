# =============================================================================
# Deployment validation
# =============================================================================
# Packages are built by goreleaser in CI. What the build contract needs locally
# is the check that runs AFTER an install: that the service answering on the
# host is the artifact the release published, with its UI embedded.
#
#   make deploy-validate HOST=10.44.40.30
#   make deploy-validate HOST=10.44.40.30 RELEASE=v0.221.3 PORT=8443
#
# RELEASE defaults to the latest published release. RELEASE, not VERSION:
# mk/vars.mk already owns VERSION as this working tree's `git describe`. The
# target only reads /__version; it never installs or restarts anything.
# =============================================================================

.PHONY: deploy-validate

deploy-validate: ## Check HOST's /__version against a release (version, commit, uiBuildHash)
ifndef HOST
	$(error HOST is required, e.g. make deploy-validate HOST=10.44.40.30)
endif
	@release="$(RELEASE)"; \
	if [ -z "$$release" ]; then \
		release=$$(gh release view --repo MustardSeedNetworks/seed --json tagName --jq .tagName) || exit 1; \
	fi; \
	commit=$$(git ls-remote origin "refs/tags/$$release" "refs/tags/$$release^{}" | tail -n 1 | cut -f 1); \
	if [ -z "$$commit" ]; then \
		printf "$(RED)No tag $$release on origin$(RESET)\n"; \
		exit 1; \
	fi; \
	scripts/deploy-validate.sh "$$release" "$$commit" "$(HOST)" "$(or $(PORT),8443)"
