# =============================================================================
# Dependency Management
# =============================================================================
#
# All dependency-related targets:
#   - Tool installation (Go + Node)
#   - Dependency updates
#   - Version checking
#
# =============================================================================

.PHONY: deps tools tools-go tools-frontend \
        update update-go update-npm version-check

# =============================================================================
# Install All Dependencies
# =============================================================================

deps: ## Install all dependencies (Go modules + npm packages)
	@printf "$(BOLD)$(CYAN)┌─ Installing Dependencies ────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Go modules                                                             $(CYAN)│$(RESET)\n"
	$(call timer-start,deps-go)
	@go mod download
	@go mod verify
	$(call timer-end,deps-go,Go modules)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) npm packages                                                           $(CYAN)│$(RESET)\n"
	$(call timer-start,deps-npm)
	@cd $(UI_DIR) && npm ci
	$(call timer-end,deps-npm,npm packages)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "$(GREEN)✓ All dependencies installed$(RESET)\n"

# =============================================================================
# Development Tools Installation
# =============================================================================

tools: tools-go tools-frontend ## Install all development tools
	@printf "$(GREEN)✓ All development tools installed$(RESET)\n"

tools-go: ## Install Go development tools
	@printf "$(BOLD)📦 Installing Go development tools...$(RESET)\n"
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
	@echo "Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Installing gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Installing gofumpt..."
	@go install mvdan.cc/gofumpt@latest
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing gotestsum..."
	@go install gotest.tools/gotestsum@latest
	@echo "Installing gitleaks..."
	@go install github.com/zricethezav/gitleaks/v8@latest
	@echo "Installing go-licenses..."
	@go install github.com/google/go-licenses@latest
	@printf "$(GREEN)✓ Go development tools installed$(RESET)\n"

tools-frontend: ## Install frontend development tools
	@printf "$(BOLD)📦 Installing frontend development tools...$(RESET)\n"
	@cd $(UI_DIR) && npm ci
	@echo "Installing Playwright browsers..."
	@cd $(UI_DIR) && npx playwright install chromium
	@printf "$(GREEN)✓ Frontend development tools installed$(RESET)\n"

# =============================================================================
# Dependency Updates
# =============================================================================

update: update-go update-npm ## Update all dependencies
	@printf "$(GREEN)✓ All dependencies updated$(RESET)\n"

update-go: ## Update Go dependencies
	@printf "$(BOLD)📦 Updating Go dependencies...$(RESET)\n"
	@go get -u ./...
	@go mod tidy
	@printf "$(GREEN)✓ Go dependencies updated$(RESET)\n"

update-npm: ## Update npm dependencies
	@printf "$(BOLD)📦 Updating npm dependencies...$(RESET)\n"
	@cd $(UI_DIR) && npm update
	@printf "$(GREEN)✓ npm dependencies updated$(RESET)\n"

# =============================================================================
# Version Checking
# =============================================================================

version-check: ## Show version info and check for outdated dependencies
	@printf "$(BOLD)$(CYAN)┌─ Version Information ────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)Runtime Versions$(RESET)                                                           $(CYAN)│$(RESET)\n"
	@printf "$(CYAN)│$(RESET)   Go:     %s\n" "$$(go version | awk '{print $$3}')"
	@printf "$(CYAN)│$(RESET)   Node:   %s\n" "$$(node --version)"
	@printf "$(CYAN)│$(RESET)   npm:    %s\n" "$$(npm --version)"
	@printf "$(CYAN)│$(RESET)                                                                              $(CYAN)│$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)Go Tools$(RESET)                                                                   $(CYAN)│$(RESET)\n"
	@printf "$(CYAN)│$(RESET)   golangci-lint: %s\n" "$$(golangci-lint --version 2>/dev/null | head -1 || echo 'not installed')"
	@printf "$(CYAN)│$(RESET)   govulncheck:   %s\n" "$$(govulncheck -version 2>/dev/null | head -1 || echo 'not installed')"
	@printf "$(CYAN)│$(RESET)   gosec:         %s\n" "$$(gosec --version 2>/dev/null | head -1 || echo 'not installed')"
	@printf "$(CYAN)│$(RESET)   gitleaks:      %s\n" "$$(gitleaks version 2>/dev/null || echo 'not installed')"
	@printf "$(CYAN)│$(RESET)                                                                              $(CYAN)│$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)Outdated Go Packages$(RESET)                                                       $(CYAN)│$(RESET)\n"
	@go list -u -m -f '{{if .Update}}$(CYAN)│$(RESET)   {{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' all 2>/dev/null | head -10 || printf "$(CYAN)│$(RESET)   (unable to check)\n"
	@printf "$(CYAN)│$(RESET)                                                                              $(CYAN)│$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)Outdated npm Packages$(RESET)                                                      $(CYAN)│$(RESET)\n"
	@cd $(UI_DIR) && npm outdated 2>/dev/null | head -10 | sed 's/^/│   /' || printf "$(CYAN)│$(RESET)   All packages up to date\n"
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
