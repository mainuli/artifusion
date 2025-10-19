.PHONY: build test test-coverage lint fmt vet clean docker-build docker-up docker-down run help security

# Binary and image names
BINARY_NAME=artifusion
DOCKER_IMAGE=artifusion:latest

# Version injection
# VERSION: Git tag or "dev" for local builds (e.g., "1.2.3" or "v1.2.3-5-gabcdef")
# GIT_COMMIT: Full git commit SHA (e.g., "abc123f456...")
# BUILD_TIME: ISO 8601 timestamp (e.g., "2025-01-15T10:30:00Z")
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Build flags for optimization and version injection
# These match the variable names in cmd/artifusion/main.go
LDFLAGS=-ldflags "-w -s \
	-X main.version=$(VERSION) \
	-X main.gitCommit=$(GIT_COMMIT) \
	-X main.buildTime=$(BUILD_TIME)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run

# Directories
BUILD_DIR=bin
COVERAGE_DIR=coverage

## help: Display this help message
help:
	@echo "Artifusion - Multi-protocol Artifact Reverse Proxy"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  test            - Run all tests with race detection"
	@echo "  test-coverage   - Run tests and generate HTML coverage report"
	@echo "  lint            - Run linters (vet, fmt check)"
	@echo "  fmt             - Format code with go fmt"
	@echo "  vet             - Run go vet"
	@echo "  clean           - Remove build artifacts"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-up       - Start services with docker-compose"
	@echo "  docker-down     - Stop services with docker-compose"
	@echo "  run             - Run the application locally"
	@echo "  security        - Run security checks"
	@echo "  deps            - Download and verify dependencies"
	@echo "  help            - Display this help message"

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/artifusion
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run all tests with race detection
test:
	@echo "Running tests with race detection..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "Tests complete"

## test-coverage: Generate HTML coverage report
test-coverage: test
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOCMD) tool cover -html=coverage.txt -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

## test-short: Run tests without race detection (faster)
test-short:
	@echo "Running tests (short mode)..."
	$(GOTEST) -v -short ./...

## lint: Run linters
lint: vet fmt-check
	@echo "Linting complete"

## fmt: Format code with go fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## fmt-check: Check if code is formatted
fmt-check:
	@echo "Checking code formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt
	@rm -rf $(COVERAGE_DIR)
	@echo "Clean complete"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-f Dockerfile .
	@echo "Docker image built: $(DOCKER_IMAGE)"

## docker-up: Start services with docker-compose
docker-up:
	@echo "Starting services with docker-compose..."
	@if [ -d "deployments/docker" ]; then \
		cd deployments/docker && docker-compose up -d; \
	else \
		echo "Error: deployments/docker directory not found"; \
		exit 1; \
	fi

## docker-down: Stop services with docker-compose
docker-down:
	@echo "Stopping services with docker-compose..."
	@if [ -d "deployments/docker" ]; then \
		cd deployments/docker && docker-compose down; \
	else \
		echo "Error: deployments/docker directory not found"; \
		exit 1; \
	fi

## run: Run the application locally
run:
	@echo "Running $(BINARY_NAME)..."
	$(GORUN) ./cmd/artifusion --config config/config.yaml

## run-dev: Run with development config
run-dev:
	@echo "Running $(BINARY_NAME) in development mode..."
	CONFIG_PATH=config/config.dev.yaml $(GORUN) ./cmd/artifusion

## security: Run security checks
security:
	@echo "Running security checks..."
	$(GOVET) ./...
	@echo "Security checks complete"
	@echo "Note: For comprehensive security scanning, install and run 'gosec' or 'govulncheck'"

## deps: Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify
	@echo "Dependencies verified"

## tidy: Tidy go.mod
tidy:
	@echo "Tidying go.mod..."
	$(GOMOD) tidy

## install: Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## all: Run fmt, vet, test, and build
all: fmt vet test build
	@echo "All tasks complete"

## ci: Run all CI checks (lint, test, build)
ci: lint test build
	@echo "CI checks complete"
