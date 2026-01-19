# =============================================================================
# NIAC Makefile
# =============================================================================
#
# Build, test, and package automation for NIAC network simulator.
#
# QUICK START
# -----------
#   make build          Build frontend and backend (default)
#   make test           Run tests with coverage
#   make lint           Run all linters
#   make help           Show all available targets
#
# COMMON WORKFLOWS
# ----------------
#   Development:        make dev (then run commands in separate terminals)
#   Before commit:      make lint test
#   Quick rebuild:      make quick (backend only)
#   Release:            make release
#
# REQUIREMENTS
# ------------
#   - Go 1.25+ (with CGO for libpcap)
#   - Node.js 25.2.1+ and npm
#   - golangci-lint
#
# =============================================================================

# =============================================================================
# Shared Infrastructure (version, colors, timers)
# =============================================================================

include Makefile.common

# =============================================================================
# Project Configuration
# =============================================================================

# Build flags (uses VERSION, COMMIT, BUILD_TIME from Makefile.common)
LDFLAGS := -X main.version=$(VERSION) -X main.date=$(BUILD_TIME) -X main.commit=$(COMMIT)

# Directories
UI_DIR := ui
UI_DIST := $(UI_DIR)/dist
EMBED_DIR := pkg/api/ui
BIN_DIR := .

# Binary names
BINARY := niac
CONVERTER := niac-convert

# =============================================================================
# Include Domain-Specific Makefiles
# =============================================================================

include mk/build.mk
include mk/test.mk
include mk/lint.mk
include mk/security.mk
include mk/deps.mk

# =============================================================================
# Version Information
# =============================================================================

.PHONY: version help

# Show version info (uses common-info from Makefile.common)
version: common-info ## Show version information

# =============================================================================
# Help
# =============================================================================

help: ## Show this help message
	@echo "$(CYAN)NIAC Build System$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@echo "  make [target]"
	@echo ""
	@echo "$(YELLOW)Build Targets:$(RESET)"
	@echo "  $(GREEN)build$(RESET)         Build frontend and backend (default)"
	@echo "  $(GREEN)frontend$(RESET)      Build only the frontend (ui)"
	@echo "  $(GREEN)backend$(RESET)       Build only the backend (Go binary)"
	@echo "  $(GREEN)quick$(RESET)         Quick rebuild (backend only)"
	@echo "  $(GREEN)converter$(RESET)     Build the niac-convert tool"
	@echo "  $(GREEN)release$(RESET)       Build release binaries"
	@echo ""
	@echo "$(YELLOW)Development:$(RESET)"
	@echo "  $(GREEN)dev$(RESET)           Show development mode instructions"
	@echo "  $(GREEN)deps$(RESET)          Install all dependencies"
	@echo "  $(GREEN)clean$(RESET)         Remove all build artifacts"
	@echo "  $(GREEN)clean-go$(RESET)      Remove only Go artifacts"
	@echo ""
	@echo "$(YELLOW)Quality:$(RESET)"
	@echo "  $(GREEN)lint$(RESET)          Run all linters (golangci-lint + Biome)"
	@echo "  $(GREEN)lint-go$(RESET)       Run Go linter only"
	@echo "  $(GREEN)lint-frontend$(RESET) Run frontend linter only"
	@echo "  $(GREEN)fmt$(RESET)           Format all code"
	@echo "  $(GREEN)fmt-go$(RESET)        Format Go code only"
	@echo "  $(GREEN)fmt-frontend$(RESET)  Format frontend code only"
	@echo "  $(GREEN)test$(RESET)          Run tests with coverage"
	@echo "  $(GREEN)security$(RESET)      Run security scans (govulncheck + gosec)"
	@echo ""
	@echo "$(YELLOW)Info:$(RESET)"
	@echo "  $(GREEN)version$(RESET)       Show version information"
	@echo "  $(GREEN)help$(RESET)          Show this help message"
	@echo ""
	@echo "$(YELLOW)Examples:$(RESET)"
	@echo "  make                    # Full build"
	@echo "  make quick              # Rebuild Go only (faster)"
	@echo "  make lint test          # Lint and test"
	@echo "  make clean build        # Clean rebuild"
