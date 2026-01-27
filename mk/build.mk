# =============================================================================
# Build Targets
# =============================================================================
#
# All build-related targets for NIAC:
#   - Frontend build (React/Vite)
#   - Backend build (Go with embedded assets)
#   - Cross-compilation (Linux, macOS, ARM64)
#   - Docker image builds
#
# =============================================================================

.PHONY: build build-frontend build-frontend-quiet build-backend build-backend-quiet \
        build-linux-amd64 build-linux-arm64 build-linux-docker \
        build-darwin build-windows build-all \
        docker-build docker-test docker docker-push \
        run dev dev-frontend frontend-deps converter quick \
        check-frontend-exists verify-build

# =============================================================================
# Main Build Target
# =============================================================================

# Build complete application (frontend embedded in Go binary)
build: ## Build everything (frontend embedded in binary)
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

build-frontend: frontend-deps ## Build React frontend
	@printf "$(BOLD)🔨 Building frontend...$(RESET)\n"
	@cd $(UI_DIR) && npm run build
	@rm -rf $(EMBED_DIR)/*
	@cp -r $(UI_DIST)/* $(EMBED_DIR)/
	@printf "$(GREEN)✓ Frontend build complete$(RESET)\n"

build-frontend-quiet: frontend-deps
	@printf "   Bundling React application...\n"
	@command -v npm >/dev/null 2>&1 || { printf "$(RED)ERROR: npm not found. Frontend build requires npm.$(RESET)\n"; exit 1; }
	@cd $(UI_DIR) && npm run build
	@rm -rf $(EMBED_DIR)/* 2>/dev/null || true
	@mkdir -p $(EMBED_DIR)
	@cp -r $(UI_DIST)/* $(EMBED_DIR)/
	@SIZE=$$(du -sh $(UI_DIST) 2>/dev/null | cut -f1 || echo "unknown"); \
	printf "   Output: $(UI_DIST) ($$SIZE)\n"

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
	@printf "$(YELLOW)  Use 'make build' for release builds$(RESET)\n"
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/niac
	@printf "$(GREEN)✓ Quick build complete: $(BINARY_NAME)$(RESET)\n"

converter: ## Build the niac-convert tool
	@printf "$(BOLD)🔨 Building converter...$(RESET)\n"
	@go build -ldflags="$(LDFLAGS)" -o $(CONVERTER_NAME) ./cmd/niac-convert
	@printf "$(GREEN)✓ Converter built: $(CONVERTER_NAME)$(RESET)\n"

# =============================================================================
# Cross-Platform Builds
# =============================================================================

build-linux-amd64: build-frontend ## Build for Linux AMD64 (requires cross-compiler)
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 ./cmd/niac

build-linux-arm64: build-frontend ## Build for Linux ARM64 (Raspberry Pi, ARM servers)
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-linux-arm64 ./cmd/niac

build-linux-docker: build-frontend ## Build for Linux AMD64 using Docker (cross-platform)
	@echo "Building for Linux AMD64 using Docker..."
	docker run --rm -v "$(PWD):/build" -w /build golang:1.25.5 bash -c "\
		apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
		go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-amd64 ./cmd/niac"
	@echo "Built: $(BINARY_NAME)-linux-amd64"

build-darwin: build-frontend ## Build for macOS (native architecture)
	@echo "Building for macOS ($(shell uname -m))..."
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-darwin-$(shell uname -m) ./cmd/niac
	@echo "Built: $(BINARY_NAME)-darwin-$(shell uname -m)"

build-windows: build-frontend ## Build for Windows AMD64 (cross-compile)
	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-windows-amd64.exe ./cmd/niac
	@echo "Built: $(BINARY_NAME)-windows-amd64.exe"

build-all: build-frontend ## Build for all platforms (native + Linux via Docker)
	@printf "$(BOLD)$(CYAN)┌─ Building All Platforms ────────────────────────────────────────────────────┐$(RESET)\n"
	@printf "$(CYAN)│$(RESET) $(BOLD)[1/3]$(RESET) macOS (native)                                                       $(CYAN)│$(RESET)\n"
	$(call timer-start,build-darwin)
	@CGO_ENABLED=1 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY_NAME)-darwin-$(shell uname -m) ./cmd/niac
	$(call timer-end,build-darwin,macOS build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[2/3]$(RESET) Linux AMD64 (Docker)                                                 $(CYAN)│$(RESET)\n"
	$(call timer-start,build-linux-amd64)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		docker run --rm -v "$(PWD):/build" -w /build golang:1.25.5 bash -c "\
			apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
			go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-amd64 ./cmd/niac"; \
	else \
		printf "$(YELLOW)   SKIP: Docker not available$(RESET)\n"; \
	fi
	$(call timer-end,build-linux-amd64,Linux AMD64 build)
	@printf "$(CYAN)│$(RESET) $(BOLD)[3/3]$(RESET) Linux ARM64 (Docker)                                                 $(CYAN)│$(RESET)\n"
	$(call timer-start,build-linux-arm64)
	@if command -v docker > /dev/null 2>&1 && docker info > /dev/null 2>&1; then \
		docker run --rm -v "$(PWD):/build" -w /build --platform linux/arm64 golang:1.25.5 bash -c "\
			apt-get update -qq && apt-get install -y -qq libpcap-dev > /dev/null && \
			go build $(GOFLAGS) -ldflags=\"$(LDFLAGS)\" -o $(BINARY_NAME)-linux-arm64 ./cmd/niac" 2>/dev/null || \
		printf "$(YELLOW)   SKIP: ARM64 emulation not available$(RESET)\n"; \
	else \
		printf "$(YELLOW)   SKIP: Docker not available$(RESET)\n"; \
	fi
	$(call timer-end,build-linux-arm64,Linux ARM64 build)
	@printf "$(CYAN)└──────────────────────────────────────────────────────────────────────────────┘$(RESET)\n"
	@printf "\n$(GREEN)✓ Multi-platform build complete$(RESET)\n"
	@ls -lh $(BINARY_NAME)-* 2>/dev/null || true

# =============================================================================
# Docker Targets
# =============================================================================

docker-build: ## Build Docker image
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-test: docker-build ## Test Docker image starts and responds
	@echo "Testing Docker image..."
	@docker run --rm -d --name niac-test -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)
	@sleep 5
	@curl -sf http://localhost:8080/api/health || (docker stop niac-test && exit 1)
	@docker stop niac-test
	@echo "Docker test passed"

docker: docker-test ## Build and test Docker image

docker-push: docker-build ## Push Docker image to registry
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "ERROR: DOCKER_REGISTRY not set"; \
		exit 1; \
	fi
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)

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
