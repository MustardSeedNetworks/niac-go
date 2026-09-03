# =============================================================================
# NIAC Makefile
# =============================================================================
#
# Local build, test, and development automation for NIAC network simulator.
#
# QUICK START
# -----------
#   make build          Build current-host binary (frontend + backend)
#   make test           Run all unit tests (backend + frontend)
#   make verify         Full local verification (lint, test, security, build)
#   make dev            Run backend in dev mode
#   make help           Show all available targets
#
# COMMON WORKFLOWS
# ----------------
#   Development:        make dev
#   Before commit:      make verify
#   Release artifacts:  built by GitHub Actions on tag/release
#
# REQUIREMENTS
# ------------
#   - Go 1.26.6 (with CGO for libpcap)
#   - Node.js 26.5.0 and npm 12.0.1
#   - libpcap-dev (Linux) or libpcap (macOS via Homebrew)
#
# =============================================================================

# =============================================================================
# Shared Variables (single source of truth)
# =============================================================================
# VERSION/COMMIT/BUILD_TIME, platform/arch detection, and ANSI color codes
# all live in mk/vars.mk. Include it first so every other mk/*.mk file (and
# the targets below) can rely on it.

include mk/vars.mk

# Platform pretty name for display — not in vars.mk since it's presentation
# only, derived from vars.mk's PLATFORM.
ifeq ($(PLATFORM),darwin)
    PLATFORM_PRETTY := macOS
else ifeq ($(PLATFORM),linux)
    PLATFORM_PRETTY := Linux
else
    PLATFORM_PRETTY := Unknown
endif

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

# VERSION_PKG comes from mk/vars.mk (single source of truth).

# Go build flags for reproducible builds
GO_BUILD_FLAGS := -trimpath -buildvcs=false
GOFLAGS=$(GO_BUILD_FLAGS)

# Directories — UI_DIR comes from mk/vars.mk. UI_DIST/EMBED_DIR defined
# BEFORE UI_BUILD_HASH because that variable uses `:=` (immediate
# evaluation) and references EMBED_DIR. Defining EMBED_DIR after
# UI_BUILD_HASH meant the hash always evaluated against an empty path and
# the embedded /__version endpoint reported uiBuildHash="" in the
# make-built binary. GitHub release workflows set uiBuildHash through
# their own ldflags.
UI_DIST := $(UI_DIR)/dist
EMBED_DIR := internal/api/ui

# UI build hash for build verification (generated from embedded assets)
UI_BUILD_HASH := $(shell if [ -d "$(EMBED_DIR)" ] && [ -n "$$(ls -A $(EMBED_DIR) 2>/dev/null)" ]; then \
	find $(EMBED_DIR) -type f -exec md5 -q {} \; 2>/dev/null | sort | md5 -q 2>/dev/null || \
	find $(EMBED_DIR) -type f -exec md5sum {} \; 2>/dev/null | sort | md5sum 2>/dev/null | cut -d' ' -f1; \
else echo ""; fi)

# Standard linker flags with version injection into internal/version
# (canonical contract shared with seed and stem).
GO_LDFLAGS = -s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).BuildTime=$(BUILD_TIME) \
	-X $(VERSION_PKG).UIBuildHash=$(UI_BUILD_HASH)
LDFLAGS=$(GO_LDFLAGS)

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
	rm -f $(CONVERTER_NAME) $(CONVERTER_NAME)-*
	rm -f coverage.out coverage.html
	find $(EMBED_DIR) -mindepth 1 ! -name .gitkeep -exec rm -rf {} +
	rm -rf $(UI_DIST)

clean-all: clean ## Clean everything including dependencies
	rm -rf $(UI_DIR)/node_modules
	rm -rf dist/

# =============================================================================
# Content Bundle
# =============================================================================
# Reproducible gzip-tar bundles in the internal/content.Extract format
# (top-level walks/ dir). Source corpus is the sanitized SNMP walk catalog
# in the sibling niac-demo-catalog repo — override CONTENT_CORPUS if yours
# lives elsewhere. `full` feeds the niac-content deb/rpm package; the deb/rpm
# ships the bundle to disk, it does not embed it — see .goreleaser.yml.
# `essentials` is the small L1 subset embedded directly in the daemon binary.

.PHONY: content-bundle content-bundle-essentials

CONTENT_CORPUS ?= ../niac-demo-catalog/walks/sanitized
CONTENT_BUNDLE := $(DIST_DIR)/niac-content-$(VERSION).tar.gz
CONTENT_BUNDLE_ESSENTIALS := $(DIST_DIR)/niac-content-essentials-$(VERSION).tar.gz

content-bundle: ## Generate the full content bundle (dist/niac-content-<version>.tar.gz)
	python3 scripts/gen-content-bundle.py --corpus $(CONTENT_CORPUS) --out $(CONTENT_BUNDLE) --mode full

content-bundle-essentials: ## Generate the essentials (L1) content bundle for embedding
	python3 scripts/gen-content-bundle.py --corpus $(CONTENT_CORPUS) --out $(CONTENT_BUNDLE_ESSENTIALS) --mode essentials

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
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) 2>/dev/null | sort -u | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build                    Build current-host binary"
	@echo "  make verify                   Full local verification"
	@echo "  make schema                   Regenerate committed config schema"

# =============================================================================
# Verification & Release
# =============================================================================

.PHONY: verify pre-commit pre-commit-install

verify: ## Full verification (lint, test, security, build, schema)
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
	$(call step,5,5,Schema Drift Check)
	@$(MAKE) --no-print-directory schema
	@git diff --exit-code -- docs/schemas/niac.schema.json docs/openapi.yaml ui/src/components/device-editor/generated >/dev/null || { \
		printf "$(RED)ERROR: a generated artefact changed (schema, OpenAPI or device-editor manifest). Run 'make schema' and commit the result.$(RESET)\n"; \
		exit 1; \
	}
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
