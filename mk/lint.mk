# =============================================================================
# Linting & Formatting Targets
# =============================================================================
#
# Code quality and formatting:
#   - Go linting (golangci-lint v2)
#   - Frontend linting (Biome)
#   - Formatting (gofmt, goimports, Biome)
#
# =============================================================================

.PHONY: lint lint-go lint-frontend fmt fmt-go fmt-frontend

# Run all linters
lint: lint-go lint-frontend ## Run all linters
	@echo "$(GREEN)✓ All linting complete$(RESET)"

# Run Go linter (golangci-lint)
lint-go: ## Run Go linter
	@echo "$(CYAN)Running Go linter (golangci-lint)...$(RESET)"
	@golangci-lint run --timeout=5m ./...
	@echo "$(GREEN)✓ Go lint complete$(RESET)"

# Run frontend linter (Biome)
lint-frontend: ## Run frontend linter (Biome)
	@echo "$(CYAN)Running frontend linter (Biome)...$(RESET)"
	@cd $(UI_DIR) && npx @biomejs/biome check src/
	@echo "$(GREEN)✓ Frontend lint complete$(RESET)"

# Format all code
fmt: fmt-go fmt-frontend ## Format all code
	@echo "$(GREEN)✓ All formatting complete$(RESET)"

# Format Go code
fmt-go: ## Format Go code
	@echo "$(CYAN)Formatting Go code...$(RESET)"
	@gofmt -w -s .
	@goimports -w -local github.com/krisarmstrong .
	@echo "$(GREEN)✓ Go formatting complete$(RESET)"

# Format frontend code
fmt-frontend: ## Format frontend code (Biome)
	@echo "$(CYAN)Formatting frontend code (Biome)...$(RESET)"
	@cd $(UI_DIR) && npx @biomejs/biome format --write src/
	@echo "$(GREEN)✓ Frontend formatting complete$(RESET)"
