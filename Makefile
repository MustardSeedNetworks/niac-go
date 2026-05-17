# =============================================================================
# NIAC Makefile
# =============================================================================
#
# Build, test, and package automation for NIAC network simulator.
#
# QUICK START
# -----------
#   make build          Build production binary (frontend + backend)
#   make test           Run all unit tests (backend + frontend)
#   make verify         Full CI pipeline (lint, test, security, build)
#   make dev            Run backend in dev mode
#   make help           Show all available targets
#
# COMMON WORKFLOWS
# ----------------
#   Development:        make dev
#   Before commit:      make verify
#   Release:            make release VERSION=v1.0.0
#   Cross-compile:      make build-all
#   Package:            make packages (deb + rpm)
#
# REQUIREMENTS
# ------------
#   - Go 1.25.5+ (with CGO for libpcap)
#   - Node.js 25.2.1+ and npm
#   - libpcap-dev (Linux) or libpcap (macOS via Homebrew)
#   - Docker (optional, for cross-compilation and container builds)
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
BINARY_NAME=niac
CONVERTER_NAME=niac-convert

# Version package path for ldflags injection
VERSION_PKG=github.com/krisarmstrong/niac/internal/version

# Go build flags for reproducible builds
GO_BUILD_FLAGS := -trimpath -buildvcs=false
GOFLAGS=$(GO_BUILD_FLAGS)

# Docker settings
DOCKER_IMAGE?=niac
DOCKER_TAG?=$(VERSION)

# Directories — defined BEFORE UI_BUILD_HASH because that variable
# uses `:=` (immediate evaluation) and references EMBED_DIR. Defining
# EMBED_DIR after UI_BUILD_HASH meant the hash always evaluated
# against an empty path and the embedded /__version endpoint reported
# uiBuildHash="" in the make-build binary. The goreleaser pipeline
# doesn't hit this because it sets uiBuildHash via its own ldflags.
UI_DIR := ui
UI_DIST := $(UI_DIR)/dist
EMBED_DIR := internal/api/ui

# UI build hash for deployment verification (generated from embedded assets)
UI_BUILD_HASH := $(shell if [ -d "$(EMBED_DIR)" ] && [ -n "$$(ls -A $(EMBED_DIR) 2>/dev/null)" ]; then \
	find $(EMBED_DIR) -type f -exec md5 -q {} \; 2>/dev/null | sort | md5 -q 2>/dev/null || \
	find $(EMBED_DIR) -type f -exec md5sum {} \; 2>/dev/null | sort | md5sum 2>/dev/null | cut -d' ' -f1; \
else echo ""; fi)

# Standard linker flags with version injection
GO_LDFLAGS = -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(BUILD_TIME) \
	-X main.uiBuildHash=$(UI_BUILD_HASH)
LDFLAGS=$(GO_LDFLAGS)

# =============================================================================
# Include Domain-Specific Makefiles
# =============================================================================

include mk/build.mk
include mk/test.mk
include mk/lint.mk
include mk/security.mk
include mk/package.mk
include mk/deps.mk
include mk/container.mk

# =============================================================================
# Default Target
# =============================================================================

all: verify ## Full build and validation (recommended before release)

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean clean-all

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -f $(CONVERTER_NAME) $(CONVERTER_NAME)-*
	rm -f coverage.out coverage.html
	rm -rf $(UI_DIST)

clean-all: clean ## Clean everything including dependencies
	rm -rf $(UI_DIR)/node_modules
	rm -rf dist/
	rm -rf releases/

# =============================================================================
# Version Information
# =============================================================================

.PHONY: version

version: ## Show version info (current build and installed)
	@printf "$(BOLD)NIAC Version Information$(RESET)\n"
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
	@echo "NIAC - Network Infrastructure Analysis & Capture by Mustard Seed Networks"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) mk/*.mk 2>/dev/null | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build                    Build production binary"
	@echo "  make verify                   Full CI pipeline locally"
	@echo "  make packages                 Build deb and rpm packages"
	@echo "  make build-all                Build for all platforms"

# =============================================================================
# Verification & Release
# =============================================================================

.PHONY: verify pre-commit pre-commit-install release release-check

verify: ## Full verification (lint, test, security, build, docker if available)
	@printf "\n$(BOLD)$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(BOLD)$(CYAN)║                        FULL VERIFICATION PIPELINE                           ║$(RESET)\n"
	@printf "$(BOLD)$(CYAN)║                        Version: $(VERSION)$(RESET)\n"
	@printf "$(BOLD)$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	$(call timer-start,verify-total)
	$(call step,1,5,Linting Code)
	$(call timer-start,lint)
	@$(MAKE) --no-print-directory lint
	$(call timer-end,lint,Linting)
	$(call step,2,5,Running Tests)
	$(call timer-start,test)
	@$(MAKE) --no-print-directory test
	$(call timer-end,test,Tests)
	$(call step,3,5,Security Scanning)
	$(call timer-start,security)
	@$(MAKE) --no-print-directory security
	$(call timer-end,security,Security)
	$(call step,4,5,Building Application)
	$(call timer-start,build)
	@$(MAKE) --no-print-directory build
	$(call timer-end,build,Build)
	$(call step,5,5,Docker Verification)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		printf "$(CYAN)│$(RESET) Docker available - running docker-test...$(CYAN)│$(RESET)\n"; \
		$(MAKE) --no-print-directory docker-test || true; \
	else \
		printf "$(YELLOW)⊘ SKIP: Docker not available - skipping docker-test$(RESET)\n"; \
	fi
	@printf "\n$(BOLD)$(GREEN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(BOLD)$(GREEN)║                        ✓ VERIFICATION COMPLETE                               ║$(RESET)\n"
	@printf "$(BOLD)$(GREEN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	$(call timer-end,verify-total,Total verification)
	@printf "\n  $(BOLD)Version:$(RESET)     $(VERSION)\n"
	@printf "  $(BOLD)Commit:$(RESET)      $(COMMIT)\n"
	@printf "  $(BOLD)Binary:$(RESET)      $(BINARY_NAME)\n"
	@printf "  $(BOLD)Docker:$(RESET)      $(DOCKER_IMAGE):$(DOCKER_TAG)\n\n"
	@printf "$(GREEN)Ready for release! Run 'make packages' to build packages.$(RESET)\n\n"

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

release: ## Create release (VERSION=vX.Y.Z required)
	@if [ -z "$(VERSION)" ] || ! echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+'; then \
		echo "ERROR: VERSION must be set in format vX.Y.Z"; \
		echo "Usage: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "ERROR: Working directory not clean. Commit or stash changes first."; \
		exit 1; \
	fi
	@printf "$(BOLD)$(CYAN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(BOLD)$(CYAN)║                         RELEASE $(VERSION)                                   ║$(RESET)\n"
	@printf "$(BOLD)$(CYAN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	@echo ""
	@echo "Step 1/5: Running verification..."
	@$(MAKE) --no-print-directory verify
	@echo ""
	@echo "Step 2/5: Building all platforms..."
	@$(MAKE) --no-print-directory build-all
	@echo ""
	@echo "Step 3/5: Creating git tag..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "Tagged: $(VERSION)"
	@echo ""
	@echo "Step 4/5: Pushing tag to origin..."
	@git push origin $(VERSION)
	@echo ""
	@echo "Step 5/5: Creating GitHub release..."
	@if command -v gh > /dev/null 2>&1; then \
		gh release create $(VERSION) \
			--title "$(VERSION)" \
			--generate-notes \
			$(BINARY_NAME)-darwin-* $(BINARY_NAME)-linux-* 2>/dev/null || \
			echo "Note: Upload binaries manually if gh release failed"; \
	else \
		echo "GitHub CLI not installed. Create release manually at:"; \
		echo "https://github.com/krisarmstrong/niac/releases/new?tag=$(VERSION)"; \
	fi
	@echo ""
	@printf "$(GREEN)╔══════════════════════════════════════════════════════════════════════════════╗$(RESET)\n"
	@printf "$(GREEN)║                     ✓ RELEASE $(VERSION) COMPLETE                            ║$(RESET)\n"
	@printf "$(GREEN)╚══════════════════════════════════════════════════════════════════════════════╝$(RESET)\n"
	@echo ""
	@echo "Binaries:"
	@ls -lh $(BINARY_NAME)-* 2>/dev/null || true

release-check: verify ## Full release validation (verify + install script check)
	@echo "Running release checks..."
	@if [ -f deploy/systemd/install.sh ]; then \
		bash -n deploy/systemd/install.sh && echo "PASS: install.sh syntax valid"; \
	fi
	@if echo "$(VERSION)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+'; then \
		echo "PASS: Version format valid ($(VERSION))"; \
	else \
		echo "WARN: Version $(VERSION) doesn't match vX.Y.Z format"; \
	fi
	@if git diff --quiet && git diff --staged --quiet; then \
		echo "PASS: No uncommitted changes"; \
	else \
		echo "WARN: Uncommitted changes detected"; \
	fi
	@echo ""
	@echo "Release checks complete. Ready for 'git tag $(VERSION) && git push --tags'"
