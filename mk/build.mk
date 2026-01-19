# =============================================================================
# Build Targets
# =============================================================================
#
# All build-related targets for NIAC:
#   - Frontend build (React/Vite)
#   - Backend build (Go binary)
#   - Converter tool
#   - Release builds
#
# =============================================================================

.PHONY: all build frontend backend converter quick clean clean-go release

# Default target
all: build ## Build everything (frontend + backend)

# Full build: frontend + backend
build: frontend backend ## Build frontend and backend
	@echo "$(GREEN)✓ Build complete: $(BINARY)$(RESET)"

# Build only the frontend
frontend: ## Build React WebUI
	@echo "$(CYAN)Building frontend...$(RESET)"
	@cd $(UI_DIR) && npm install --silent && npm run build
	@echo "$(CYAN)Copying frontend to embed directory...$(RESET)"
	@rm -rf $(EMBED_DIR)/*
	@cp -r $(UI_DIST)/* $(EMBED_DIR)/
	@echo "$(GREEN)✓ Frontend built$(RESET)"

# Build only the backend (assumes frontend is already built)
backend: ## Build Go binary
	@echo "$(CYAN)Building backend...$(RESET)"
	@go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) ./cmd/niac
	@echo "$(GREEN)✓ Backend built: $(BINARY)$(RESET)"

# Build the converter tool (no libpcap dependency)
converter: ## Build the niac-convert tool
	@echo "$(CYAN)Building converter...$(RESET)"
	@go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(CONVERTER) ./cmd/niac-convert
	@echo "$(GREEN)✓ Converter built: $(CONVERTER)$(RESET)"

# Quick backend rebuild (skip frontend)
quick: backend ## Quick rebuild (backend only)
	@echo "$(GREEN)✓ Quick build complete$(RESET)"

# Clean build artifacts
clean: ## Remove all build artifacts
	@echo "$(CYAN)Cleaning build artifacts...$(RESET)"
	@rm -f $(BINARY) $(CONVERTER)
	@rm -rf $(UI_DIR)/node_modules
	@rm -rf $(UI_DIST)
	@rm -rf $(EMBED_DIR)/assets $(EMBED_DIR)/index.html $(EMBED_DIR)/vite.svg
	@echo "$(GREEN)✓ Clean complete$(RESET)"

# Clean only Go artifacts (keep frontend)
clean-go: ## Remove only Go artifacts
	@rm -f $(BINARY) $(CONVERTER)
	@echo "$(GREEN)✓ Go artifacts cleaned$(RESET)"

# Build release binaries
release: build converter ## Build release binaries
	@echo "$(CYAN)Building release...$(RESET)"
	@mkdir -p releases/$(VERSION)
	@cp $(BINARY) releases/$(VERSION)/$(BINARY)-$(VERSION)-$$(go env GOOS)-$$(go env GOARCH)
	@cp $(CONVERTER) releases/$(VERSION)/$(CONVERTER)-$(VERSION)-$$(go env GOOS)-$$(go env GOARCH)
	@echo "$(GREEN)✓ Release built in releases/$(VERSION)/$(RESET)"
