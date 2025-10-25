# Makefile for Enhanced Gauge HTML Report

.PHONY: all build install test clean dist help

# Variables
BINARY_NAME=html-report-enhanced
VERSION?=1.0.0
BUILD_DIR=build
DIST_DIR=dist
THEMES_DIR=web/themes
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")
PLATFORMS=linux darwin windows
ARCHITECTURES=amd64 arm64

# Build tags
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT) -X main.date=$(BUILD_TIME)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: clean build ## Clean and build

deps: ## Download dependencies
	@echo "Downloading Go dependencies..."
	go mod download
	go mod tidy
	@echo "Installing frontend dependencies..."
	cd web && npm install

build: deps ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/html-report-enhanced/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-web: ## Build frontend assets
	@echo "Building frontend assets..."
	cd web && npm run build
	@echo "Frontend build complete"

install: build ## Install the binary
	@echo "Installing $(BINARY_NAME)..."
	gauge install $(BINARY_NAME) --file $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Installation complete"

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linters
	@echo "Running linters..."
	golangci-lint run ./...

format: ## Format code
	@echo "Formatting code..."
	gofmt -s -w $(GO_FILES)
	go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

dist: clean ## Create distribution packages
	@echo "Creating distribution packages..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			output_name=$(BINARY_NAME)-$(VERSION)-$$platform-$$arch; \
			if [ $$platform = "windows" ]; then \
				output_name=$$output_name.exe; \
			fi; \
			echo "Building $$output_name..."; \
			GOOS=$$platform GOARCH=$$arch go build $(LDFLAGS) \
				-o $(DIST_DIR)/$$output_name cmd/html-report-enhanced/main.go; \
		done \
	done
	@echo "Copying themes..."
	@cp -r $(THEMES_DIR) $(DIST_DIR)/themes
	@cp plugin.json $(DIST_DIR)/
	@echo "Distribution packages created in $(DIST_DIR)"

package: dist ## Create installation packages
	@echo "Creating installation packages..."
	@for platform in $(PLATFORMS); do \
		for arch in $(ARCHITECTURES); do \
			package_name=$(BINARY_NAME)-$(VERSION)-$$platform-$$arch; \
			echo "Packaging $$package_name..."; \
			cd $(DIST_DIR) && \
			zip -r $$package_name.zip \
				$$package_name* \
				themes/ \
				plugin.json && \
			cd ..; \
		done \
	done
	@echo "Installation packages created"

run: build ## Build and run
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME) --help

dev: ## Run in development mode
	@echo "Running in development mode..."
	go run cmd/html-report-enhanced/main.go serve --watch

generate: ## Generate code
	@echo "Generating code..."
	go generate ./...

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

docker-run: docker-build ## Run in Docker
	docker run -it --rm -p 8080:8080 $(BINARY_NAME):$(VERSION)

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

.DEFAULT_GOAL := help