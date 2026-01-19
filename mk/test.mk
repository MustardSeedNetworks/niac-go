# =============================================================================
# Test Targets
# =============================================================================
#
# All testing targets for NIAC:
#   - Go unit tests with coverage
#
# =============================================================================

.PHONY: test test-coverage

# Run tests
test: ## Run tests with coverage
	@echo "$(CYAN)Running tests...$(RESET)"
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)✓ Tests complete$(RESET)"

# Run tests with HTML coverage report
test-coverage: test ## Generate HTML coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report: coverage.html$(RESET)"
