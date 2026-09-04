# =============================================================================
# Build Targets
# =============================================================================
#
# Local build targets for NIAC:
#   - Frontend build (React/Vite)
#   - Backend build for the current host with embedded assets
#
# GitHub Actions owns cross-platform artifacts, provenance, and checksums.
#
# =============================================================================

.PHONY: build build-frontend build-frontend-quiet build-backend build-backend-quiet \
        run dev dev-frontend frontend-deps converter quick \
        check-frontend-exists verify-build

# =============================================================================
# Main Build Target
# =============================================================================

# Build complete application (frontend embedded in Go binary)
build: ## Build current host binary with embedded frontend
	@printf "$(BOLD)$(CYAN)┌─ Building Application ───────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/2]$(RESET) Frontend (React/TypeScript)                                           $(CYAN)│$(RESET)\n"
	$(call timer-start,build-frontend)
	@$(MAKE) --no-print-directory build-frontend-quiet
	$(call timer-end,build-frontend,Frontend build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/2]$(RESET) Backend (Go)                                                          $(CYAN)│$(RESET)\n"
	$(call timer-start,build-backend)
	@$(MAKE) --no-print-directory build-backend-quiet
	$(call timer-end,build-backend,Backend build)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "$(GREEN)✓ Build complete: $(BINARY_NAME) ($(VERSION))$(RESET)\n"

# =============================================================================
# Frontend Build
# =============================================================================

frontend-deps: ## Install frontend dependencies (cached)
	@if [ ! -d $(UI_DIR)/node_modules ] || [ $(UI_DIR)/package-lock.json -nt $(UI_DIR)/node_modules/.package-lock.json ]; then \
		echo "Installing frontend dependencies..."; \
		cd $(UI_DIR) && npm ci; \
	else \
		echo "Frontend dependencies up to date"; \
	fi

build-frontend: frontend-deps ## Build React frontend (outputs directly to embed dir)
	@printf "$(BOLD)🔨 Building frontend...$(RESET)\n"
	@cd $(UI_DIR) && npm run build
	@printf "$(GREEN)✓ Frontend build complete$(RESET)\n"

build-frontend-quiet: frontend-deps
	@printf "   Bundling React application...\n"
	@command -v npm >/dev/null 2>&1 || { printf "$(RED)ERROR: npm not found. Frontend build requires npm.$(RESET)\n"; exit 1; }
	@cd $(UI_DIR) && npm run build
	@SIZE=$$(du -sh $(EMBED_DIR) 2>/dev/null | cut -f1 || echo "unknown"); \
	printf "   Output: $(EMBED_DIR) ($$SIZE)\n"

# =============================================================================
# Backend Build
# =============================================================================

# =============================================================================
# Build Verification
# =============================================================================

check-frontend-exists: ## Verify frontend assets exist before backend build
	@if [ ! -f "$(EMBED_DIR)/index.html" ]; then \
		printf "$(RED)ERROR: Frontend not built. Run 'make build-frontend' first.$(RESET)\n"; \
		exit 1; \
	fi
	@if [ ! -d "$(EMBED_DIR)/assets" ]; then \
		printf "$(RED)ERROR: Frontend assets missing. Run 'make build-frontend' first.$(RESET)\n"; \
		exit 1; \
	fi

verify-build: build ## Verify build produces valid binary with embedded UI
	@printf "$(BOLD)$(CYAN)=== Build Verification ===$(RESET)\n"
	@printf "Checking binary version...\n"
	@./$(BINARY_NAME) version || ./$(BINARY_NAME) --version || true
	@printf "\n"
	@printf "Checking embedded UI...\n"
	@if strings ./$(BINARY_NAME) 2>/dev/null | grep -q "index.html"; then \
		printf "$(GREEN)✓ UI embedded correctly$(RESET)\n"; \
	else \
		printf "$(RED)ERROR: UI not embedded!$(RESET)\n"; \
		exit 1; \
	fi
	@printf "\n"
	@printf "Checking version injection...\n"
	@if ./$(BINARY_NAME) version 2>&1 | grep -q "$(VERSION)" || ./$(BINARY_NAME) --version 2>&1 | grep -q "$(VERSION)"; then \
		printf "$(GREEN)✓ Version embedded correctly$(RESET)\n"; \
	else \
		printf "$(YELLOW)⚠ Version may not match exactly (expected: $(VERSION))$(RESET)\n"; \
	fi
	@printf "\n$(GREEN)✓ Build verification complete$(RESET)\n"

# =============================================================================
# Backend Build
# =============================================================================

build-backend: check-frontend-exists ## Build Go backend with embedded frontend
	@printf "$(BOLD)🔨 Building backend...$(RESET)\n"
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/niac
	@printf "$(GREEN)✓ Backend build complete: $(BINARY_NAME)$(RESET)\n"

build-backend-quiet: check-frontend-exists
	@printf "   Compiling Go binary...\n"
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/niac
	@SIZE=$$(ls -lh $(BINARY_NAME) | awk '{print $$5}'); \
	printf "   Output: $(BINARY_NAME) ($$SIZE)\n"

quick: ## Quick backend rebuild (skip frontend) - FOR DEVELOPMENT ONLY
	@printf "$(YELLOW)⚠ Quick build - frontend NOT updated$(RESET)\n"
	@printf "$(YELLOW)  Use 'make build' for production builds$(RESET)\n"
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/niac
	@printf "$(GREEN)✓ Quick build complete: $(BINARY_NAME)$(RESET)\n"

converter: ## Build the niac-convert tool
	@printf "$(BOLD)🔨 Building converter...$(RESET)\n"
	@go build -ldflags="$(LDFLAGS)" -o $(CONVERTER_NAME) ./cmd/niac-convert
	@printf "$(GREEN)✓ Converter built: $(CONVERTER_NAME)$(RESET)\n"

schema: ## Regenerate docs/schemas/niac.schema.json from converter.Config
	@printf "$(BOLD)Generating JSON Schema for the YAML config...$(RESET)\n"
	@go run ./cmd/niac-schema -o docs/schemas/niac.schema.json
	@printf "$(GREEN)Wrote docs/schemas/niac.schema.json ($$(wc -l < docs/schemas/niac.schema.json) lines)$(RESET)\n"
	@$(MAKE) --no-print-directory ui-sections
	@$(MAKE) --no-print-directory openapi

openapi: ## Regenerate docs/openapi.yaml from the capability registry
	@printf "$(BOLD)Generating the OpenAPI description from the route registry...$(RESET)\n"
	@go run ./cmd/niac-openapi -o docs/openapi.yaml
	@printf "$(GREEN)Wrote docs/openapi.yaml ($$(wc -l < docs/openapi.yaml) lines)$(RESET)\n"

ui-sections: ## Regenerate the device-editor form manifest from the schema
	@printf "$(BOLD)Generating device-editor sections from the schema...$(RESET)\n"
	@./scripts/gen-device-editor-sections.py

cli-docs: ## Regenerate the CLI sections of README.md and docs/CLI_REFERENCE.md
	@printf "$(BOLD)Generating the CLI reference from the command tree...$(RESET)\n"
	@go run ./cmd/niac docs

man: ## Regenerate the man pages under docs/man
	@printf "$(BOLD)Generating man pages...$(RESET)\n"
	@go run ./cmd/niac man

.PHONY: schema ui-sections openapi cli-docs man

# =============================================================================
# Development Targets
# =============================================================================

run: ## Run the application
	@echo "Running $(BINARY_NAME) (use Ctrl+C to stop)..."
	@./$(BINARY_NAME) serve

dev: ## Run backend in development mode
	@echo "Running backend in development mode..."
	@go run ./cmd/niac serve

dev-frontend: ## Run frontend in development mode
	@echo "Starting Vite dev server..."
	@cd $(UI_DIR) && npm run dev
