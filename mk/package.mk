# =============================================================================
# Package Creation Targets
# =============================================================================
#
# Package creation for distribution:
#   - Debian packages (.deb) for Ubuntu/Debian
#   - RPM packages (.rpm) for RHEL/CentOS/Fedora
#   - macOS installer (.pkg)
#   - Multi-architecture support (AMD64/ARM64)
#
# =============================================================================

.PHONY: deb rpm pkg packages packages-all \
        deb-amd64 deb-arm64 rpm-amd64 rpm-arm64 _deb-arch _rpm-arch \
        container

# =============================================================================
# Package Variables
# =============================================================================

PKG_ARCH=$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
PKG_VERSION=$(shell echo $(VERSION) | sed 's/^v//')
DEB_ARCH=$(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
RPM_ARCH=$(shell uname -m | sed 's/amd64/x86_64/;s/arm64/aarch64/')

# =============================================================================
# Debian Packages
# =============================================================================

deb: build ## Build Debian package (.deb)
	@printf "$(BOLD)📦 Building Debian package...$(RESET)\n"
	@mkdir -p dist/deb/DEBIAN
	@mkdir -p dist/deb/usr/bin
	@mkdir -p dist/deb/usr/lib/systemd/system
	@mkdir -p dist/deb/var/lib/seed
	@mkdir -p dist/deb/var/log/seed
	@cp $(BINARY_NAME) dist/deb/usr/bin/seed
	@chmod 755 dist/deb/usr/bin/seed
	@cp packaging/seed.service dist/deb/usr/lib/systemd/system/
	@sed 's/__VERSION__/$(PKG_VERSION)/g; s/__ARCHITECTURE__/$(DEB_ARCH)/g' \
		packaging/control > dist/deb/DEBIAN/control
	@cp packaging/postinst dist/deb/DEBIAN/
	@cp packaging/prerm dist/deb/DEBIAN/
	@cp packaging/postrm dist/deb/DEBIAN/
	@chmod 755 dist/deb/DEBIAN/postinst dist/deb/DEBIAN/prerm dist/deb/DEBIAN/postrm
	@dpkg-deb --build dist/deb dist/seed_$(PKG_VERSION)_$(DEB_ARCH).deb
	@printf "$(GREEN)✓ Debian package: dist/seed_$(PKG_VERSION)_$(DEB_ARCH).deb$(RESET)\n"

deb-amd64: ## Build Debian package for amd64
	@$(MAKE) _deb-arch ARCH=amd64 BINARY=seed-linux-amd64

deb-arm64: ## Build Debian package for arm64
	@$(MAKE) _deb-arch ARCH=arm64 BINARY=seed-linux-arm64

_deb-arch:
	@printf "$(BOLD)📦 Building Debian package for $(ARCH)...$(RESET)\n"
	@mkdir -p dist/deb-$(ARCH)/DEBIAN
	@mkdir -p dist/deb-$(ARCH)/usr/bin
	@mkdir -p dist/deb-$(ARCH)/usr/lib/systemd/system
	@mkdir -p dist/deb-$(ARCH)/var/lib/seed
	@mkdir -p dist/deb-$(ARCH)/var/log/seed
	@cp $(BINARY) dist/deb-$(ARCH)/usr/bin/seed
	@chmod 755 dist/deb-$(ARCH)/usr/bin/seed
	@cp packaging/seed.service dist/deb-$(ARCH)/usr/lib/systemd/system/
	@sed 's/__VERSION__/$(PKG_VERSION)/g; s/__ARCHITECTURE__/$(ARCH)/g' \
		packaging/control > dist/deb-$(ARCH)/DEBIAN/control
	@cp packaging/postinst dist/deb-$(ARCH)/DEBIAN/
	@cp packaging/prerm dist/deb-$(ARCH)/DEBIAN/
	@cp packaging/postrm dist/deb-$(ARCH)/DEBIAN/
	@chmod 755 dist/deb-$(ARCH)/DEBIAN/postinst dist/deb-$(ARCH)/DEBIAN/prerm dist/deb-$(ARCH)/DEBIAN/postrm
	@dpkg-deb --build dist/deb-$(ARCH) dist/seed_$(PKG_VERSION)_$(ARCH).deb
	@printf "$(GREEN)✓ dist/seed_$(PKG_VERSION)_$(ARCH).deb$(RESET)\n"

# =============================================================================
# RPM Packages
# =============================================================================

rpm: build ## Build RPM package (.rpm)
	@printf "$(BOLD)📦 Building RPM package...$(RESET)\n"
	@mkdir -p dist/rpm/BUILD dist/rpm/RPMS dist/rpm/SOURCES dist/rpm/SPECS dist/rpm/SRPMS
	@mkdir -p dist/rpm/SOURCES/seed-$(PKG_VERSION)
	@cp $(BINARY_NAME) dist/rpm/SOURCES/seed-$(PKG_VERSION)/seed
	@cp packaging/seed.service dist/rpm/SOURCES/seed-$(PKG_VERSION)/
	@sed 's/__VERSION__/$(PKG_VERSION)/g; s/__ARCHITECTURE__/$(RPM_ARCH)/g; s|%{_repo_root}|$(CURDIR)|g' \
		packaging/seed.spec > dist/rpm/SPECS/seed.spec
	@rpmbuild --define "_topdir $(CURDIR)/dist/rpm" \
		--define "_repo_root $(CURDIR)" \
		-bb dist/rpm/SPECS/seed.spec
	@mv dist/rpm/RPMS/$(RPM_ARCH)/*.rpm dist/ 2>/dev/null || true
	@printf "$(GREEN)✓ RPM package: dist/seed-$(PKG_VERSION)-1.*.$(RPM_ARCH).rpm$(RESET)\n"

rpm-amd64: ## Build RPM package for amd64
	@$(MAKE) _rpm-arch ARCH=x86_64 BINARY=seed-linux-amd64

rpm-arm64: ## Build RPM package for arm64
	@$(MAKE) _rpm-arch ARCH=aarch64 BINARY=seed-linux-arm64

_rpm-arch:
	@printf "$(BOLD)📦 Building RPM package for $(ARCH)...$(RESET)\n"
	@mkdir -p dist/rpm-$(ARCH)/BUILD dist/rpm-$(ARCH)/RPMS dist/rpm-$(ARCH)/SOURCES dist/rpm-$(ARCH)/SPECS dist/rpm-$(ARCH)/SRPMS
	@mkdir -p dist/rpm-$(ARCH)/SOURCES/seed-$(PKG_VERSION)
	@cp $(BINARY) dist/rpm-$(ARCH)/SOURCES/seed-$(PKG_VERSION)/seed
	@cp packaging/seed.service dist/rpm-$(ARCH)/SOURCES/seed-$(PKG_VERSION)/
	@sed 's/__VERSION__/$(PKG_VERSION)/g; s/__RPM_ARCH__/$(ARCH)/g; s|%{_repo_root}|$(CURDIR)|g' \
		packaging/seed.spec > dist/rpm-$(ARCH)/SPECS/seed.spec
	@rpmbuild --define "_topdir $(CURDIR)/dist/rpm-$(ARCH)" \
		--define "_repo_root $(CURDIR)" \
		--target $(ARCH) \
		-bb dist/rpm-$(ARCH)/SPECS/seed.spec
	@mv dist/rpm-$(ARCH)/RPMS/$(ARCH)/*.rpm dist/ 2>/dev/null || true
	@printf "$(GREEN)✓ dist/seed-$(PKG_VERSION)-1.$(ARCH).rpm$(RESET)\n"

# =============================================================================
# macOS Package
# =============================================================================

pkg: build-darwin ## Build macOS installer package (.pkg)
	@if [ "$$(uname -s)" != "Darwin" ]; then \
		printf "$(RED)ERROR: macOS .pkg can only be built on macOS$(RESET)\n"; \
		exit 1; \
	fi
	@printf "$(BOLD)📦 Building macOS .pkg package...$(RESET)\n"
	@./packaging/macos/build-pkg.sh ./$(BINARY_NAME)-darwin-$$(uname -m) $(PKG_VERSION)
	@printf "$(GREEN)✓ macOS package: dist/seed-$(PKG_VERSION)-$$(uname -m | sed 's/x86_64/amd64/').pkg$(RESET)\n"

# =============================================================================
# Multi-Package Targets
# =============================================================================

packages: deb rpm ## Build both .deb and .rpm packages
	@printf "$(GREEN)✓ All packages built in dist/$(RESET)\n"
	@ls -la dist/*.deb dist/*.rpm 2>/dev/null || true

packages-all: ## Build .deb and .rpm for both amd64 and arm64
	@printf "$(BOLD)📦 Building packages for all architectures...$(RESET)\n"
	@printf "$(CYAN)This requires Docker for cross-compilation$(RESET)\n"
	@$(MAKE) build-iperf3-linux
	@$(MAKE) build-linux-amd64
	@$(MAKE) build-linux-arm64
	@$(MAKE) deb-amd64
	@$(MAKE) rpm-amd64
	@$(MAKE) deb-arm64
	@$(MAKE) rpm-arm64
	@printf "$(GREEN)✓ All packages built:$(RESET)\n"
	@ls -la dist/*.deb dist/*.rpm 2>/dev/null || true

# =============================================================================
# Container Images (Pack/Buildpacks) - LOCAL DEV ONLY
# =============================================================================
# NOTE: No public registry pushing during development.
# License validation required before commercial distribution.
# TODO: Add private registry when ready for deployment.

CONTAINER_IMAGE := seed

container: ## Build container image locally (Pack/Buildpacks)
	@printf "$(BOLD)🐺 Building container with Pack (local only)...$(RESET)\n"
	@pack build $(CONTAINER_IMAGE):$(VERSION) \
		--builder paketobuildpacks/builder-jammy-base \
		--env BP_GO_TARGETS="./cmd/seed" \
		--env BP_GO_BUILD_LDFLAGS="-s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT)"
	@printf "$(GREEN)✓ Container: $(CONTAINER_IMAGE):$(VERSION) (local)$(RESET)\n"
	@printf "$(YELLOW)⚠ Local build only - no registry push during development$(RESET)\n"
