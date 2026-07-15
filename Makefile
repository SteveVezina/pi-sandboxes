# PI Agent Sandbox Runtime — Build & Test

# ─── Variables ────────────────────────────────────────────────────────────────

BINARY_BOX           = pi-box
BINARY_SANDBOXD      = pi-sandboxd
BINARY_AGENTD        = pi-agentd
BINARY_INIT          = pi-init
BINARY_VMM_MANAGER   = pi-vmm-manager
ALL_BINARIES         = $(BINARY_BOX) $(BINARY_SANDBOXD) $(BINARY_AGENTD) $(BINARY_INIT) $(BINARY_VMM_MANAGER)

INSTALL_DIR          = $(HOME)/bin
BUILD_DIR            = bin

VERSION              = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE           = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS              = -ldflags "-s -w -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)"

MAIN_BOX             = cmd/pi-box/main.go
MAIN_SANDBOXD        = cmd/pi-sandboxd/main.go
MAIN_AGENTD          = cmd/pi-agentd/main.go
MAIN_INIT            = cmd/pi-init/main.go
MAIN_VMM_MANAGER     = cmd/pi-vmm-manager/main.go

TEST_TIMEOUT         = 30s
TEST_RACE            = -race

# ─── Phony targets ────────────────────────────────────────────────────────────

.PHONY: build help test test-short test-coverage clean lint format version \
        build-box build-sandboxd build-agentd build-init build-vmm-manager \
        install docker docker-run docker-stop docker-build \
        mock-up mock-down \
        gui-dev gui-build gui-desktop-dev gui-desktop-build gui-install \
        template-build template-pull template-push template-install

# ─── Help ─────────────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ─── Build ────────────────────────────────────────────────────────────────────

build: build-sandboxd build-box build-agentd build-init build-vmm-manager ## Build all binaries

build-box: ## Build CLI binary
	@echo "Building $(BINARY_BOX)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_BOX) $(MAIN_BOX)

build-sandboxd: ## Build sandbox daemon
	@echo "Building $(BINARY_SANDBOXD)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_SANDBOXD) $(MAIN_SANDBOXD)

build-agentd: ## Build agent-side daemon
	@echo "Building $(BINARY_AGENTD)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_AGENTD) $(MAIN_AGENTD)

build-init: ## Build MicroVM guest init
	@echo "Building $(BINARY_INIT)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_INIT) $(MAIN_INIT)

build-vmm-manager: ## Build MicroVM manager
	@echo "Building $(BINARY_VMM_MANAGER)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_VMM_MANAGER) $(MAIN_VMM_MANAGER)

# ─── Install ──────────────────────────────────────────────────────────────────

install: build ## Build and install all binaries to ~/bin
	@mkdir -p $(INSTALL_DIR)
	@echo "Installing to $(INSTALL_DIR)..."
	install $(addprefix $(BUILD_DIR)/,$(ALL_BINARIES)) $(INSTALL_DIR)/
	@echo "Done: $(ALL_BINARIES)"

# ─── Test ─────────────────────────────────────────────────────────────────────

test: ## Run all tests with race detector
	go test ./tests/... $(TEST_RACE) -count=1 -timeout=$(TEST_TIMEOUT)

test-short: ## Run tests without race detector (faster)
	go test ./tests/... -count=1 -timeout=$(TEST_TIMEOUT)

test-coverage: ## Run tests with coverage report
	go test ./tests/... $(TEST_RACE) -count=1 -timeout=$(TEST_TIMEOUT) -coverprofile=coverage.out -coverpkg=./pkg/...
	go tool cover -html=coverage.out -o coverage.html

# ─── Clean ────────────────────────────────────────────────────────────────────

clean: ## Remove build artifacts and coverage files
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# ─── Lint & Format ────────────────────────────────────────────────────────────

lint: ## Run Go linting
	@echo "Checking code style..."
	go vet ./...
	@golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed, skipping"

format: ## Format code
	go fmt ./...

# ─── Docker ───────────────────────────────────────────────────────────────────

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t pi-sandbox:$(VERSION) .

docker-run: ## Run sandbox daemon in Docker
	@echo "Running pi-sandboxd in Docker..."
	docker run -d \
		--name pi-sandbox \
		-v $(HOME)/.pi-box:/home/pi/.pi-box \
		-p 9001:9001 \
		pi-sandbox:$(VERSION)

docker-stop: ## Stop and remove Docker container
	docker stop pi-sandbox 2>/dev/null || true
	docker rm pi-sandbox 2>/dev/null || true

# ─── Version ──────────────────────────────────────────────────────────────────

# ─── Mock Services ──────────────────────────────────────────────────────────

mock-up: ## Start mock services (skip if docker-compose not found)
	@if [ -f mocks/docker-compose.mocks.yml ]; then \
		docker compose -f mocks/docker-compose.mocks.yml up -d; \
		echo "Mocks started: Orchestrator:9001 Gateway:9002 SessionManager:9003 SecretManager:9004"; \
	else \
		echo "No mock services configured (mocks/docker-compose.mocks.yml not found)"; \
	fi

mock-down: ## Stop mock services
	@if [ -f mocks/docker-compose.mocks.yml ]; then \
		docker compose -f mocks/docker-compose.mocks.yml down; \
	else \
		echo "No mock services configured (mocks/docker-compose.mocks.yml not found)"; \
	fi

# ─── GUI (Tauri + Vite/React) ────────────────────────────────────────────────

GUI_DIR    = apps/gui
GUI_SRC    = $(GUI_DIR)/src
GUI_DIST   = $(GUI_DIR)/dist
GUI_BIN    = $(GUI_DIR)/src-tauri/target/release/pi-sandbox

GUI_DEPS   = $(GUI_DIR)/package.json
GUI_SRC_TS = $(wildcard $(GUI_SRC)/**/*.ts $(GUI_SRC)/**/*.tsx)

.PHONY: gui gui-dev gui-build gui-desktop-dev gui-desktop-build gui-install

gui: gui-build ## Build GUI frontend only

gui-install: ## Install GUI dependencies
	@echo "Installing GUI dependencies..."
	cd $(GUI_DIR) && npm install

gui-dev: gui-install ## Start GUI dev server (Vite)
	@echo "Starting GUI dev server..."
	cd $(GUI_DIR) && npm run dev

gui-build: gui-install ## Build GUI frontend (TypeScript + Vite)
	@echo "Building GUI frontend..."
	cd $(GUI_DIR) && npm run build
	@echo "Frontend built to $(GUI_DIST)/"

gui-desktop-dev: gui-install ## Start Tauri desktop app in dev mode
	@echo "Starting Tauri desktop dev..."
	cd $(GUI_DIR) && npm run desktop:dev

gui-desktop-build: gui-install ## Build Tauri desktop app (macOS/Windows/Linux)
	@echo "Building Tauri desktop app..."
	cd $(GUI_DIR) && npm run desktop:build
	@echo "Desktop app built to $(GUI_DIR)/src-tauri/target/release/"

# ─── Template Images (OCI/Docker) ─────────────────────────────────────────────

TEMPLATE_DIR  = specs/templates
TEMPLATE_IMAGES := pi-template-base \
                   pi-template-node \
                   pi-template-python \
                   pi-template-go \
                   pi-template-rust \
                   pi-template-node-python \
                   pi-template-polyglot

.PHONY: template template-build template-pull template-push template-install

template: template-build ## Build all template images

template-build: ## Build OCI images for all templates
	@if [ ! -d $(TEMPLATE_DIR) ]; then \
		echo "No template images configured (specs/templates/ not found)"; \
		echo "Create specs/templates/<name>/Dockerfile for each template"; \
	else \
		echo "Building template images..."; \
		for tpl in $(TEMPLATE_IMAGES); do \
			if [ -f $(TEMPLATE_DIR)/$$tpl/Dockerfile ]; then \
				echo "  → $$tpl"; \
				docker build -t "$$tpl:$(VERSION)" -f $(TEMPLATE_DIR)/$$tpl/Dockerfile . ; \
			else \
				echo "  → $$tpl (no Dockerfile, skipping)"; \
			fi; \
		done; \
		echo "Done. Images: $(addsuffix :$(VERSION),$(TEMPLATE_IMAGES))"; \
	fi

template-pull: ## Pull base images for templates
	@echo "Pulling base images..."
	@for img in debian:bookworm-slim node:22-bookworm python:3.13-bookworm golang:1.24-bookworm rust:1.80-bookworm; do \
		echo "  → $$img"; \
		docker pull "$$img"; \
	done

template-push: ## Push all template images to registry
	@REGISTRY ?= ghcr.io/pi-sandbox
	@echo "Pushing template images to $$REGISTRY..."
	@for tpl in $(TEMPLATE_IMAGES); do \
		TAG="$$REGISTRY/$$tpl:$(VERSION)"; \
		echo "  → $$TAG"; \
		docker tag "$$tpl:$(VERSION)" "$$TAG"; \
		docker push "$$TAG"; \
	done

template-install: template-build ## Build and load template images into local Docker
	@echo "Loading template images into local Docker..."
	@for tpl in $(TEMPLATE_IMAGES); do \
		echo "  → $$tpl"; \
		docker tag "$$tpl:$(VERSION)" "$$tpl:latest"; \
	done

# ─── Version ──────────────────────────────────────────────────────────────────

version: ## Show version
	@echo "Version: $(VERSION)"
	@echo "Build: $(BUILD_DATE)"
