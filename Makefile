# NIAC Makefile
# Builds frontend and backend into a single binary

# Include shared build infrastructure
include Makefile.common

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

.PHONY: all build clean dev frontend backend help lint lint-go lint-frontend test release security version deps quick converter clean-go fmt fmt-go fmt-frontend

# Default target
all: build

# Full build: frontend + backend
build: frontend backend
	@echo "$(GREEN)✓ Build complete: $(BINARY)$(RESET)"

# Build only the frontend
frontend:
	@echo "$(CYAN)Building frontend...$(RESET)"
	@cd $(UI_DIR) && npm install --silent && npm run build
	@echo "$(CYAN)Copying frontend to embed directory...$(RESET)"
	@rm -rf $(EMBED_DIR)/*
	@cp -r $(UI_DIST)/* $(EMBED_DIR)/
	@echo "$(GREEN)✓ Frontend built$(RESET)"

# Build only the backend (assumes frontend is already built)
backend:
	@echo "$(CYAN)Building backend...$(RESET)"
	@go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/niac
	@echo "$(GREEN)✓ Backend built: $(BINARY)$(RESET)"

# Build the converter tool (no libpcap dependency)
converter:
	@echo "$(CYAN)Building converter...$(RESET)"
	@go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(CONVERTER) ./cmd/niac-convert
	@echo "$(GREEN)✓ Converter built: $(CONVERTER)$(RESET)"

# Quick backend rebuild (skip frontend)
quick: backend
	@echo "$(GREEN)✓ Quick build complete$(RESET)"

# Development mode: watch frontend and rebuild
dev:
	@echo "$(CYAN)Starting development mode...$(RESET)"
	@echo "Run these in separate terminals:"
	@echo "  Terminal 1: cd $(UI_DIR) && npm run dev"
	@echo "  Terminal 2: make quick && sudo ./$(BINARY) run <interface> <config> --web"

# Clean build artifacts
clean:
	@echo "$(CYAN)Cleaning build artifacts...$(RESET)"
	@rm -f $(BINARY) $(CONVERTER)
	@rm -rf $(UI_DIR)/node_modules
	@rm -rf $(UI_DIST)
	@rm -rf $(EMBED_DIR)/assets $(EMBED_DIR)/index.html $(EMBED_DIR)/vite.svg
	@echo "$(GREEN)✓ Clean complete$(RESET)"

# Clean only Go artifacts (keep frontend)
clean-go:
	@rm -f $(BINARY) $(CONVERTER)
	@echo "$(GREEN)✓ Go artifacts cleaned$(RESET)"

# Install dependencies
deps:
	@echo "$(CYAN)Installing dependencies...$(RESET)"
	@go mod download
	@cd $(UI_DIR) && npm install
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

# ==============================================================================
# Linting
# ==============================================================================

# Run all linters
lint: lint-go lint-frontend
	@echo "$(GREEN)✓ All linting complete$(RESET)"

# Run Go linter (golangci-lint)
lint-go:
	@echo "$(CYAN)Running Go linter (golangci-lint)...$(RESET)"
	@golangci-lint run --timeout=5m ./...
	@echo "$(GREEN)✓ Go lint complete$(RESET)"

# Run frontend linter (Biome)
lint-frontend:
	@echo "$(CYAN)Running frontend linter (Biome)...$(RESET)"
	@cd $(UI_DIR) && npx @biomejs/biome check src/
	@echo "$(GREEN)✓ Frontend lint complete$(RESET)"

# ==============================================================================
# Formatting
# ==============================================================================

# Format all code
fmt: fmt-go fmt-frontend
	@echo "$(GREEN)✓ All formatting complete$(RESET)"

# Format Go code
fmt-go:
	@echo "$(CYAN)Formatting Go code...$(RESET)"
	@gofmt -w -s .
	@goimports -w -local github.com/krisarmstrong .
	@echo "$(GREEN)✓ Go formatting complete$(RESET)"

# Format frontend code
fmt-frontend:
	@echo "$(CYAN)Formatting frontend code (Biome)...$(RESET)"
	@cd $(UI_DIR) && npx @biomejs/biome format --write src/
	@echo "$(GREEN)✓ Frontend formatting complete$(RESET)"

# ==============================================================================
# Testing & Security
# ==============================================================================

# Run tests
test:
	@echo "$(CYAN)Running tests...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)✓ Tests complete$(RESET)"

# Run security scans
security:
	@echo "$(CYAN)Running security scans...$(RESET)"
	@echo "$(CYAN)  → govulncheck...$(RESET)"
	@govulncheck ./...
	@echo "$(CYAN)  → gosec...$(RESET)"
	@gosec -quiet ./...
	@echo "$(GREEN)✓ Security scans complete$(RESET)"

# ==============================================================================
# Release
# ==============================================================================

# Build release binaries
release: build converter
	@echo "$(CYAN)Building release...$(RESET)"
	@mkdir -p releases/$(VERSION)
	@cp $(BINARY) releases/$(VERSION)/$(BINARY)-$(VERSION)-$$(go env GOOS)-$$(go env GOARCH)
	@cp $(CONVERTER) releases/$(VERSION)/$(CONVERTER)-$(VERSION)-$$(go env GOOS)-$$(go env GOARCH)
	@echo "$(GREEN)✓ Release built in releases/$(VERSION)/$(RESET)"

# Show version info (uses common-info from Makefile.common)
version: common-info

# ==============================================================================
# Help
# ==============================================================================

help:
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
