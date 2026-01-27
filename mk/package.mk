# =============================================================================
# Package Creation Targets
# =============================================================================
#
# Package creation for distribution:
#   - Debian packages (.deb) for Ubuntu/Debian
#   - RPM packages (.rpm) for RHEL/CentOS/Fedora
#   - macOS installer (.pkg)
#   - Windows zip distribution
#   - Container images via Pack (Cloud Native Buildpacks)
#
# =============================================================================

.PHONY: deb rpm pkg windows-zip packages install-service container \
        deploy-ubuntu deploy-fedora deploy-all deploy-validate

# =============================================================================
# Package Variables
# =============================================================================

PKG_VERSION := $(shell echo $(VERSION) | sed 's/^v//;s/-dirty$$//;s/-[0-9]*-g[0-9a-f]*$$//')
DEB_ARCH := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
RPM_ARCH := $(shell uname -m | sed 's/amd64/x86_64/;s/arm64/aarch64/')

# =============================================================================
# Debian Package
# =============================================================================

deb: build ## Build Debian package (.deb)
	@printf "$(BOLD)Building Debian package...$(RESET)\n"
	@mkdir -p dist/deb/DEBIAN
	@mkdir -p dist/deb/usr/bin
	@mkdir -p dist/deb/usr/lib/systemd/system
	@mkdir -p dist/deb/var/lib/niac
	@mkdir -p dist/deb/var/lib/niac/templates
	@mkdir -p dist/deb/var/log/niac
	@cp $(BINARY_NAME) dist/deb/usr/bin/niac
	@chmod 755 dist/deb/usr/bin/niac
	@cp deploy/systemd/niac.service dist/deb/usr/lib/systemd/system/
	@cp -r cmd/niac/templates/* dist/deb/var/lib/niac/templates/ 2>/dev/null || true
	@# Only copy YAML config examples, not large walk files or captures
	@find examples -name "*.yaml" -o -name "*.yml" | while read f; do \
		dir=$$(dirname "$$f" | sed 's|^examples/||'); \
		mkdir -p "dist/deb/var/lib/niac/templates/$$dir"; \
		cp "$$f" "dist/deb/var/lib/niac/templates/$$dir/"; \
	done 2>/dev/null || true
	@sed 's/__VERSION__/$(PKG_VERSION)/g; s/__ARCHITECTURE__/$(DEB_ARCH)/g' \
		deploy/deb/control > dist/deb/DEBIAN/control
	@cp deploy/deb/postinst dist/deb/DEBIAN/
	@cp deploy/deb/prerm dist/deb/DEBIAN/
	@cp deploy/deb/postrm dist/deb/DEBIAN/
	@chmod 755 dist/deb/DEBIAN/postinst dist/deb/DEBIAN/prerm dist/deb/DEBIAN/postrm
	@dpkg-deb --build dist/deb dist/niac_$(PKG_VERSION)_$(DEB_ARCH).deb
	@printf "$(GREEN)Debian package: dist/niac_$(PKG_VERSION)_$(DEB_ARCH).deb$(RESET)\n"

# =============================================================================
# RPM Package
# =============================================================================

rpm: build ## Build RPM package (.rpm)
	@printf "$(BOLD)Building RPM package...$(RESET)\n"
	@mkdir -p dist/rpm/BUILD dist/rpm/RPMS dist/rpm/SOURCES dist/rpm/SPECS dist/rpm/SRPMS
	@mkdir -p dist/rpm/SOURCES/niac-$(PKG_VERSION)
	@cp $(BINARY_NAME) dist/rpm/SOURCES/niac-$(PKG_VERSION)/niac
	@cp deploy/systemd/niac.service dist/rpm/SOURCES/niac-$(PKG_VERSION)/
	@sed 's/__VERSION__/$(PKG_VERSION)/g; s/__ARCHITECTURE__/$(RPM_ARCH)/g; s|%{_repo_root}|$(CURDIR)|g' \
		deploy/rpm/niac.spec > dist/rpm/SPECS/niac.spec
	@rpmbuild --define "_topdir $(CURDIR)/dist/rpm" \
		--define "_repo_root $(CURDIR)" \
		-bb dist/rpm/SPECS/niac.spec
	@mv dist/rpm/RPMS/$(RPM_ARCH)/*.rpm dist/ 2>/dev/null || true
	@printf "$(GREEN)RPM package: dist/niac-$(PKG_VERSION)-1.*.$(RPM_ARCH).rpm$(RESET)\n"

# =============================================================================
# macOS Package
# =============================================================================

pkg: build-darwin ## Build macOS installer package (.pkg)
	@if [ "$$(uname -s)" != "Darwin" ]; then \
		printf "$(RED)ERROR: macOS .pkg can only be built on macOS$(RESET)\n"; \
		exit 1; \
	fi
	@printf "$(BOLD)Building macOS .pkg package...$(RESET)\n"
	@./deploy/macos/build-pkg.sh ./$(BINARY_NAME)-darwin-$$(uname -m) $(PKG_VERSION)
	@printf "$(GREEN)macOS package: dist/niac-$(PKG_VERSION)-$$(uname -m | sed 's/x86_64/amd64/').pkg$(RESET)\n"

# =============================================================================
# Windows Distribution
# =============================================================================

windows-zip: build-windows ## Build Windows zip distribution
	@printf "$(BOLD)Building Windows distribution...$(RESET)\n"
	@mkdir -p dist
	@PKG_NAME="niac-$(PKG_VERSION)-windows-amd64"; \
	mkdir -p "dist/$$PKG_NAME"; \
	cp niac-windows-amd64.exe "dist/$$PKG_NAME/niac.exe" 2>/dev/null || \
	cp $(BINARY_NAME).exe "dist/$$PKG_NAME/niac.exe"; \
	cp deploy/windows/build.ps1 "dist/$$PKG_NAME/install.ps1"; \
	cd dist && zip -r "$$PKG_NAME.zip" "$$PKG_NAME" && rm -rf "$$PKG_NAME"
	@printf "$(GREEN)Windows distribution: dist/niac-$(PKG_VERSION)-windows-amd64.zip$(RESET)\n"

# =============================================================================
# Combined Targets
# =============================================================================

packages: deb rpm ## Build all packages (deb + rpm)
	@printf "$(GREEN)All packages built in dist/$(RESET)\n"
	@ls -la dist/*.deb dist/*.rpm 2>/dev/null || true

# =============================================================================
# Systemd Service Installation
# =============================================================================

install-service: build ## Install as systemd service (requires root)
	@printf "$(BOLD)Installing systemd service...$(RESET)\n"
	install -D -m 0755 $(BINARY_NAME) /usr/bin/niac
	install -D -m 0644 deploy/systemd/niac.service /lib/systemd/system/niac.service
	@if ! getent group niac >/dev/null; then groupadd -r niac; fi
	@if ! getent passwd niac >/dev/null; then \
		useradd -r -g niac -d /var/lib/niac -s /sbin/nologin niac; \
	fi
	install -d -m 0750 -o niac -g niac /var/lib/niac
	install -d -m 0750 -o niac -g niac /var/log/niac
	@if command -v setcap >/dev/null; then \
		setcap 'cap_net_raw,cap_net_admin=+ep' /usr/bin/niac || true; \
	fi
	systemctl daemon-reload
	@echo "Service installed. Run: systemctl enable --now niac"

# =============================================================================
# Container Images (Pack/Buildpacks) - LOCAL DEV ONLY
# =============================================================================

CONTAINER_IMAGE := niac

container: ## Build container image locally (Pack/Buildpacks)
	@printf "$(BOLD)Building container with Pack (local only)...$(RESET)\n"
	@pack build $(CONTAINER_IMAGE):$(VERSION) \
		--builder paketobuildpacks/builder-jammy-base \
		--env BP_GO_TARGETS="./cmd/niac" \
		--env BP_GO_BUILD_LDFLAGS="-s -w -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT)"
	@printf "$(GREEN)Container: $(CONTAINER_IMAGE):$(VERSION) (local)$(RESET)\n"

# =============================================================================
# Deployment Targets
# =============================================================================
# These targets deploy packages to test servers and validate the installation.
#
# Prerequisites:
#   - SSH access to target servers (key-based auth recommended)
#   - Packages built with 'make packages'
#   - Target servers: niac-srv-ubuntu, niac-srv-fedora in ~/.ssh/config
#
# =============================================================================

# Deployment server hostnames (configure in ~/.ssh/config)
DEPLOY_UBUNTU_HOST ?= niac-srv-ubuntu
DEPLOY_FEDORA_HOST ?= niac-srv-fedora
DEPLOY_PORT ?= 8080

deploy-validate: ## Validate deployment on a remote host (HOST=hostname)
	@if [ -z "$(HOST)" ]; then \
		printf "$(RED)ERROR: HOST is required. Usage: make deploy-validate HOST=hostname$(RESET)\n"; \
		exit 1; \
	fi
	@printf "$(BOLD)Validating deployment on $(HOST)...$(RESET)\n"
	./scripts/deploy-validate.sh $(VERSION) $(COMMIT) $(HOST) $(DEPLOY_PORT)

deploy-ubuntu: packages ## Deploy .deb to Ubuntu test server and validate
	@printf "$(BOLD)Deploying to $(DEPLOY_UBUNTU_HOST)...$(RESET)\n"
	@DEB_FILE=$$(ls -t dist/niac_*_amd64.deb 2>/dev/null | head -1); \
	if [ -z "$$DEB_FILE" ]; then \
		printf "$(RED)ERROR: No .deb file found in dist/. Run 'make packages' first.$(RESET)\n"; \
		exit 1; \
	fi; \
	printf "  Uploading $$DEB_FILE...\n"; \
	scp "$$DEB_FILE" $(DEPLOY_UBUNTU_HOST):/tmp/niac.deb; \
	printf "  Installing package...\n"; \
	ssh $(DEPLOY_UBUNTU_HOST) 'sudo dpkg -i /tmp/niac.deb'; \
	printf "  Restarting service...\n"; \
	ssh $(DEPLOY_UBUNTU_HOST) 'sudo systemctl restart niac || sudo systemctl start niac'; \
	printf "  Waiting for service startup...\n"; \
	sleep 3; \
	printf "  Validating deployment...\n"
	@./scripts/deploy-validate.sh $(VERSION) $(COMMIT) $(DEPLOY_UBUNTU_HOST) $(DEPLOY_PORT)
	@printf "$(GREEN)✓ Ubuntu deployment complete and validated$(RESET)\n"

deploy-fedora: packages ## Deploy .rpm to Fedora test server and validate
	@printf "$(BOLD)Deploying to $(DEPLOY_FEDORA_HOST)...$(RESET)\n"
	@RPM_FILE=$$(ls -t dist/niac-*-1.*.rpm 2>/dev/null | head -1); \
	if [ -z "$$RPM_FILE" ]; then \
		printf "$(RED)ERROR: No .rpm file found in dist/. Run 'make packages' first.$(RESET)\n"; \
		exit 1; \
	fi; \
	printf "  Uploading $$RPM_FILE...\n"; \
	scp "$$RPM_FILE" $(DEPLOY_FEDORA_HOST):/tmp/niac.rpm; \
	printf "  Installing package...\n"; \
	ssh $(DEPLOY_FEDORA_HOST) 'sudo rpm -Uvh --nodeps /tmp/niac.rpm || sudo rpm -ivh --nodeps /tmp/niac.rpm'; \
	printf "  Restarting service...\n"; \
	ssh $(DEPLOY_FEDORA_HOST) 'sudo systemctl restart niac || sudo systemctl start niac'; \
	printf "  Waiting for service startup...\n"; \
	sleep 3; \
	printf "  Validating deployment...\n"
	@./scripts/deploy-validate.sh $(VERSION) $(COMMIT) $(DEPLOY_FEDORA_HOST) $(DEPLOY_PORT)
	@printf "$(GREEN)✓ Fedora deployment complete and validated$(RESET)\n"

deploy-all: deploy-ubuntu deploy-fedora ## Deploy to all test servers
	@printf "\n$(GREEN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(GREEN)║                    ✓ ALL DEPLOYMENTS VALIDATED                                ║$(RESET)\n"
	@printf "$(GREEN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
