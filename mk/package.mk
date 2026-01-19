# =============================================================================
# Package Creation Targets
# =============================================================================
#
# Package creation for distribution:
#   - Debian packages (.deb) for Ubuntu/Debian
#   - RPM packages (.rpm) for RHEL/CentOS/Fedora
#   - Container images via Pack (Cloud Native Buildpacks)
#
# =============================================================================

.PHONY: deb rpm packages install-service container

# =============================================================================
# Package Variables
# =============================================================================

PKG_VERSION := $(shell echo $(VERSION) | sed 's/^v//')
DEB_ARCH := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

# =============================================================================
# Debian Package
# =============================================================================

deb: build ## Build Debian package (.deb)
	@printf "$(BOLD)📦 Building Debian package...$(RESET)\n"
	@mkdir -p dist/deb/DEBIAN
	@mkdir -p dist/deb/usr/bin
	@mkdir -p dist/deb/usr/lib/systemd/system
	@mkdir -p dist/deb/var/lib/niac
	@mkdir -p dist/deb/var/log/niac
	@cp $(BIN_DIR)/niac dist/deb/usr/bin/niac
	@chmod 755 dist/deb/usr/bin/niac
	@cp deploy/systemd/niac.service dist/deb/usr/lib/systemd/system/
	@sed 's/__VERSION__/$(PKG_VERSION)/g; s/__ARCHITECTURE__/$(DEB_ARCH)/g' \
		deploy/deb/control > dist/deb/DEBIAN/control
	@cp deploy/deb/postinst dist/deb/DEBIAN/
	@cp deploy/deb/postrm dist/deb/DEBIAN/
	@chmod 755 dist/deb/DEBIAN/postinst dist/deb/DEBIAN/postrm
	@dpkg-deb --build dist/deb dist/niac_$(PKG_VERSION)_$(DEB_ARCH).deb
	@printf "$(GREEN)✓ Debian package: dist/niac_$(PKG_VERSION)_$(DEB_ARCH).deb$(RESET)\n"

# =============================================================================
# RPM Package
# =============================================================================

rpm: build ## Build RPM package (.rpm)
	@printf "$(BOLD)📦 Building RPM package...$(RESET)\n"
	@echo "Note: Requires rpmbuild to be installed"
	@mkdir -p ~/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
	@cp $(BIN_DIR)/niac ~/rpmbuild/SOURCES/niac
	@echo "Name: niac" > ~/rpmbuild/SPECS/niac.spec
	@echo "Version: $(PKG_VERSION)" >> ~/rpmbuild/SPECS/niac.spec
	@echo "Release: 1" >> ~/rpmbuild/SPECS/niac.spec
	@echo "Summary: Network Device Simulator" >> ~/rpmbuild/SPECS/niac.spec
	@echo "License: Proprietary" >> ~/rpmbuild/SPECS/niac.spec
	@echo "%description" >> ~/rpmbuild/SPECS/niac.spec
	@echo "NIAC - Network Infrastructure Agent Cluster" >> ~/rpmbuild/SPECS/niac.spec
	@echo "%install" >> ~/rpmbuild/SPECS/niac.spec
	@echo "mkdir -p %{buildroot}/usr/bin" >> ~/rpmbuild/SPECS/niac.spec
	@echo "cp %{SOURCE0} %{buildroot}/usr/bin/niac" >> ~/rpmbuild/SPECS/niac.spec
	@echo "%files" >> ~/rpmbuild/SPECS/niac.spec
	@echo "/usr/bin/niac" >> ~/rpmbuild/SPECS/niac.spec
	rpmbuild -bb ~/rpmbuild/SPECS/niac.spec
	@printf "$(GREEN)✓ RPM package built$(RESET)\n"

# =============================================================================
# Combined Targets
# =============================================================================

packages: deb rpm ## Build all packages (deb + rpm)

# =============================================================================
# Systemd Service Installation
# =============================================================================

install-service: build ## Install as systemd service (requires root)
	@printf "$(BOLD)Installing systemd service...$(RESET)\n"
	install -D -m 0755 $(BIN_DIR)/niac /usr/bin/niac
	install -D -m 0644 deploy/systemd/niac.service /lib/systemd/system/niac.service
	@if ! getent group niac >/dev/null; then groupadd -r niac; fi
	@if ! getent passwd niac >/dev/null; then \
		useradd -r -g niac -d /var/lib/niac -s /sbin/nologin niac; \
	fi
	install -d -m 0750 -o niac -g niac /var/lib/niac
	install -d -m 0750 -o niac -g niac /var/log/niac
	systemctl daemon-reload
	@echo "Service installed. Run: systemctl enable --now niac"

# =============================================================================
# Container Images (Pack/Buildpacks) - LOCAL DEV ONLY
# =============================================================================
# NOTE: NIAC may be open-sourced. Registry push disabled during development.
# TODO: Decide on open source vs commercial before enabling public distribution.

CONTAINER_IMAGE := niac

container: ## Build container image locally (Pack/Buildpacks)
	@printf "$(BOLD)🐺 Building container with Pack (local only)...$(RESET)\n"
	@pack build $(CONTAINER_IMAGE):$(VERSION) \
		--builder paketobuildpacks/builder-jammy-base \
		--env BP_GO_TARGETS="./cmd/niac" \
		--env BP_GO_BUILD_LDFLAGS="-s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT)"
	@printf "$(GREEN)✓ Container: $(CONTAINER_IMAGE):$(VERSION) (local)$(RESET)\n"
	@printf "$(YELLOW)⚠ Local build only - no registry push during development$(RESET)\n"
