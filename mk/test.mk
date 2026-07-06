# =============================================================================
# Test Targets
# =============================================================================
#
# All testing targets:
#   - Unit tests (backend + frontend)
#   - Integration tests
#   - E2E tests (Playwright)
#   - Coverage reports
#
# =============================================================================

.PHONY: test test-all test-backend test-backend-quiet test-frontend test-frontend-quiet \
        test-e2e test-e2e-ui test-e2e-install test-coverage

# =============================================================================
# Main Test Targets
# =============================================================================

test: ## Run unit tests (backend + frontend)
	@printf "$(BOLD)$(CYAN)┌─ Unit Tests ─────────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Backend (Go)                                                          $(CYAN)│$(RESET)\n"
	$(call timer-start,test-backend)
	@$(MAKE) --no-print-directory test-backend-quiet
	$(call timer-end,test-backend,Backend tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) Frontend (Vitest)                                                      $(CYAN)│$(RESET)\n"
	$(call timer-start,test-frontend)
	@$(MAKE) --no-print-directory test-frontend-quiet
	$(call timer-end,test-frontend,Frontend tests)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

test-all: ## Run ALL tests (unit + E2E)
	@printf "$(BOLD)$(CYAN)┌─ Full Test Suite ────────────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/3]$(RESET) Backend unit tests                                                    $(CYAN)│$(RESET)\n"
	$(call timer-start,test-backend)
	@$(MAKE) --no-print-directory test-backend-quiet
	$(call timer-end,test-backend,Backend tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/3]$(RESET) Frontend unit tests                                                   $(CYAN)│$(RESET)\n"
	$(call timer-start,test-frontend)
	@$(MAKE) --no-print-directory test-frontend-quiet
	$(call timer-end,test-frontend,Frontend tests)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/3]$(RESET) E2E tests (Playwright)                                                $(CYAN)│$(RESET)\n"
	$(call timer-start,test-e2e)
	@$(MAKE) --no-print-directory test-e2e
	$(call timer-end,test-e2e,E2E tests)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"

# =============================================================================
# Backend Tests
# =============================================================================

test-backend: ## Run Go tests with progress
	@printf "\n$(BOLD)🧪 Running backend tests...$(RESET)\n"
	@PKGS=$$(go list ./... | grep -v '/ui$$'); \
	PKG_COUNT=$$(echo "$$PKGS" | wc -l | tr -d ' '); \
	printf "   📦 Testing $$PKG_COUNT packages...\n\n"; \
	if command -v gotestsum > /dev/null 2>&1; then \
		gotestsum --format pkgname-and-test-fails -- -race -coverprofile=coverage.out $$PKGS; \
	else \
		go test -race -coverprofile=coverage.out $$PKGS; \
	fi
	@if [ -f coverage.out ]; then \
		COV=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}'); \
		printf "\n   📊 Coverage: %s\n" "$$COV"; \
	fi
	@printf "\n$(GREEN)✓ Backend tests complete$(RESET)\n"

test-backend-quiet:
	@PKGS=$$(go list ./... | grep -v '/ui$$'); \
	PKG_COUNT=$$(echo "$$PKGS" | wc -l | tr -d ' '); \
	printf "   Testing $$PKG_COUNT packages...\n"; \
	OUTPUT=$$(go test -race -coverprofile=coverage.out $$PKGS 2>&1); \
	STATUS=$$?; \
	echo "$$OUTPUT" | grep -E "^(ok|FAIL|---)"; \
	if [ $$STATUS -ne 0 ]; then \
		echo ""; \
		echo "$$OUTPUT" | grep -v -E "^(ok|FAIL|---)" | tail -60; \
		exit $$STATUS; \
	fi
	@if [ -f coverage.out ]; then \
		COV=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}'); \
		printf "   📊 Coverage: %s\n" "$$COV"; \
	fi

# =============================================================================
# Frontend Tests
# =============================================================================

test-frontend: ## Run frontend tests with progress
	@printf "\n$(BOLD)🧪 Running frontend tests...$(RESET)\n"
	@STORY_COUNT=$$(find $(UI_DIR)/src -name "*.test.ts" -o -name "*.test.tsx" 2>/dev/null | wc -l | tr -d ' '); \
	printf "   📦 Running $$STORY_COUNT test files...\n\n"
	@cd $(UI_DIR) && npm test
	@printf "\n$(GREEN)✓ Frontend tests complete$(RESET)\n"

test-frontend-quiet:
	@STORY_COUNT=$$(find $(UI_DIR)/src -name "*.test.ts" -o -name "*.test.tsx" 2>/dev/null | wc -l | tr -d ' '); \
	printf "   Running $$STORY_COUNT test files...\n"; \
	OUTPUT=$$(cd $(UI_DIR) && npm test 2>&1); \
	STATUS=$$?; \
	echo "$$OUTPUT" | grep -E "(PASS|FAIL|Tests:)"; \
	if [ $$STATUS -ne 0 ]; then \
		echo ""; \
		echo "$$OUTPUT" | tail -60; \
		exit $$STATUS; \
	fi

# =============================================================================
# E2E Tests
# =============================================================================

test-e2e: ## Run frontend E2E tests (requires backend running)
	@echo ""
	@echo "🎭 Running E2E tests (Playwright)..."
	@E2E_COUNT=$$(find $(UI_DIR)/e2e -name "*.spec.ts" 2>/dev/null | wc -l | tr -d ' '); \
	echo "   📦 Running $$E2E_COUNT spec files..."
	@echo ""
	@cd $(UI_DIR) && npm run test:e2e
	@echo ""
	@echo "✅ E2E tests complete"

test-e2e-ui: ## Run E2E tests with Playwright UI
	@echo "🎭 Starting Playwright UI mode..."
	cd $(UI_DIR) && npx playwright test --ui

test-e2e-install: ## Install Playwright browsers
	cd $(UI_DIR) && npx playwright install --with-deps chromium

# =============================================================================
# Coverage
# =============================================================================

test-coverage: ## Generate coverage report
	@PKGS=$$(go list ./... | grep -v '/ui$$'); \
	go test -race -coverprofile=coverage.out $$PKGS
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
