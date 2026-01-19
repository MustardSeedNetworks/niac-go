# =============================================================================
# Security Scanning Targets
# =============================================================================
#
# Security targets for NIAC:
#   - Go vulnerability scanning (govulncheck)
#   - Security analysis (gosec)
#
# =============================================================================

.PHONY: security

# Run security scans
security: ## Run security scans (govulncheck + gosec)
	@echo "$(CYAN)Running security scans...$(RESET)"
	@echo "$(CYAN)  → govulncheck...$(RESET)"
	@govulncheck ./...
	@echo "$(CYAN)  → gosec...$(RESET)"
	@gosec -quiet ./...
	@echo "$(GREEN)✓ Security scans complete$(RESET)"
