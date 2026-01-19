# =============================================================================
# Dependency Management
# =============================================================================
#
# Dependency management targets for NIAC:
#   - Install dependencies
#   - Development mode
#
# =============================================================================

.PHONY: deps dev

# Install dependencies
deps: ## Install all dependencies
	@echo "$(CYAN)Installing dependencies...$(RESET)"
	@go mod download
	@cd $(UI_DIR) && npm install
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

# Development mode: show instructions
dev: ## Show development mode instructions
	@echo "$(CYAN)Starting development mode...$(RESET)"
	@echo "Run these in separate terminals:"
	@echo "  Terminal 1: cd $(UI_DIR) && npm run dev"
	@echo "  Terminal 2: make quick && sudo ./$(BINARY) run <interface> <config> --web"
