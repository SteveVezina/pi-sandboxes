.PHONY: build install test clean lint mock-up mock-down help

# Variables
BINARY_SANDBOXD=pi-sandboxd
BINARY_PI=pi
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)"

# Go paths
MAIN_SANDBOXD=cmd/pi-sandboxd/main.go
MAIN_PI=cmd/pi/main.go

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: build-sandboxd build-pi ## Build all binaries

build-sandboxd: ## Build sandbox daemon
	@echo "Building $(BINARY_SANDBOXD)..."
	go build $(LDFLAGS) -o $(BINARY_SANDBOXD) $(MAIN_SANDBOXD)

build-pi: ## Build CLI binary
	@echo "Building $(BINARY_PI)..."
	go build $(LDFLAGS) -o $(BINARY_PI) $(MAIN_PI)

install: build ## Build and install binaries
	@echo "Installing to $(GOPATH)/bin..."
	install $(BINARY_SANDBOXD) $(GOPATH)/bin/
	install $(BINARY_PI) $(GOPATH)/bin/

test: ## Run all tests
	go test ./tests/... -count=1 -race -timeout=30s

test-short: ## Run tests without race detector (faster)
	go test ./tests/... -count=1 -timeout=30s

test-coverage: ## Run tests with coverage
	go test ./tests/... -count=1 -race -coverprofile=coverage.out -coverpkg=./pkg/... -timeout=30s
	go tool cover -html=coverage.out -o coverage.html

clean: ## Remove build artifacts
	rm -f $(BINARY_SANDBOXD) $(BINARY_PI)
	rm -f coverage.out coverage.html

mock-up: ## Start mock services
	docker compose -f mocks/docker-compose.mocks.yml up -d
	@echo "Mocks started: Orchestrator:9001 Gateway:9002 SessionManager:9003 SecretManager:9004"

mock-down: ## Stop mock services
	docker compose -f mocks/docker-compose.mocks.yml down

lint: ## Run Go linting
	@echo "Checking code style..."
	go vet ./...
	@golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed, skipping"

format: ## Format code
	go fmt ./...

.PHONY: version
version: ## Show version
	@echo "Version: $(VERSION)"
	@echo "Build: $(BUILD_DATE)"
