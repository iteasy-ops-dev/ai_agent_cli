#  AI Agent Makefile
# Cross-platform build automation

.PHONY: help build clean test run dev install deps build-all build-windows build-macos build-linux fmt vet lint

# Default target
.DEFAULT_GOAL := help

# Build configuration
APP_NAME := syseng-agent
BUILD_DIR := dist
VERSION ?= dev
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)
BUILD_FLAGS := -ldflags="$(LDFLAGS)"

# Colors for output
BLUE := \033[34m
GREEN := \033[32m
YELLOW := \033[33m
RED := \033[31m
NC := \033[0m

help: ## Show this help message
	@echo "$(BLUE) AI Agent Build System$(NC)"
	@echo ""
	@echo "$(GREEN)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(GREEN)Environment variables:$(NC)"
	@echo "  $(YELLOW)VERSION$(NC)     Set build version (default: dev)"
	@echo "  $(YELLOW)GOOS$(NC)        Target operating system"
	@echo "  $(YELLOW)GOARCH$(NC)      Target architecture"

build: ## Build for current platform
	@echo "$(BLUE)Building $(APP_NAME) for current platform...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "$(GREEN)✓ Build complete: $(BUILD_DIR)/$(APP_NAME)$(NC)"

build-all: ## Build for all platforms using build script
	@echo "$(BLUE)Building for all platforms...$(NC)"
	@if [ -x "./build.sh" ]; then \
		./build.sh; \
	else \
		echo "$(RED)Error: build.sh not found or not executable$(NC)"; \
		exit 1; \
	fi

build-windows: ## Build for Windows (amd64 and 386)
	@echo "$(BLUE)Building for Windows...$(NC)"
	@mkdir -p $(BUILD_DIR)/windows
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/windows/$(APP_NAME)-windows-amd64.exe .
	GOOS=windows GOARCH=386 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/windows/$(APP_NAME)-windows-386.exe .
	@echo "$(GREEN)✓ Windows builds complete$(NC)"

build-macos: ## Build for macOS (Intel and Apple Silicon)
	@echo "$(BLUE)Building for macOS...$(NC)"
	@mkdir -p $(BUILD_DIR)/macos
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/macos/$(APP_NAME)-macos-amd64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/macos/$(APP_NAME)-macos-arm64 .
	@echo "$(GREEN)✓ macOS builds complete$(NC)"

build-linux: ## Build for Linux (amd64, arm64, and 386)
	@echo "$(BLUE)Building for Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)/linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/linux/$(APP_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/linux/$(APP_NAME)-linux-arm64 .
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/linux/$(APP_NAME)-linux-386 .
	@echo "$(GREEN)✓ Linux builds complete$(NC)"

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f $(APP_NAME) $(APP_NAME).exe
	rm -f coverage.out coverage.html
	@echo "$(GREEN)✓ Clean complete$(NC)"

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	go test -v ./...
	@echo "$(GREEN)✓ Tests complete$(NC)"

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"

run: ## Run the application locally
	@echo "$(BLUE)Running $(APP_NAME)...$(NC)"
	go run . $(ARGS)

dev: build ## Build and run for development
	@echo "$(BLUE)Starting development mode...$(NC)"
	$(BUILD_DIR)/$(APP_NAME) $(ARGS)

install: ## Install the application to $GOPATH/bin
	@echo "$(BLUE)Installing $(APP_NAME)...$(NC)"
	go install $(BUILD_FLAGS) .
	@echo "$(GREEN)✓ Installed to $(shell go env GOPATH)/bin/$(APP_NAME)$(NC)"

deps: ## Download and tidy dependencies
	@echo "$(BLUE)Managing dependencies...$(NC)"
	go mod download
	go mod tidy
	@echo "$(GREEN)✓ Dependencies updated$(NC)"

fmt: ## Format Go code
	@echo "$(BLUE)Formatting code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)✓ Vet check complete$(NC)"

lint: ## Run golangci-lint (requires golangci-lint to be installed)
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "$(GREEN)✓ Lint check complete$(NC)"; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

check: fmt vet test ## Run all checks (format, vet, test)
	@echo "$(GREEN)✓ All checks passed$(NC)"

release: clean build-all ## Prepare a release build
	@echo "$(BLUE)Preparing release...$(NC)"
	@echo "$(GREEN)✓ Release build complete in $(BUILD_DIR)/$(NC)"

info: ## Show build information
	@echo "$(BLUE)Build Information:$(NC)"
	@echo "  App Name: $(APP_NAME)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Go Version: $(shell go version)"
	@echo "  Build Dir: $(BUILD_DIR)"

# Development helpers
watch: ## Watch for changes and rebuild (requires entr)
	@echo "$(BLUE)Watching for changes...$(NC)"
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -r make build; \
	else \
		echo "$(RED)Error: entr not installed. Install with your package manager.$(NC)"; \
		exit 1; \
	fi

docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t $(APP_NAME):$(VERSION) .
	@echo "$(GREEN)✓ Docker image built: $(APP_NAME):$(VERSION)$(NC)"

# Platform-specific builds with custom naming
windows: build-windows ## Alias for build-windows
macos: build-macos ## Alias for build-macos  
linux: build-linux ## Alias for build-linux
all: build-all ## Alias for build-all