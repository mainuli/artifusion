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
	@echo "  Build & Test:"
	@echo "    build           - Build the binary"
	@echo "    test            - Run all tests with race detection"
	@echo "    test-coverage   - Run tests and generate HTML coverage report"
	@echo "    lint            - Run linters (vet, fmt check)"
	@echo "    fmt             - Format code with go fmt"
	@echo "    vet             - Run go vet"
	@echo "    clean           - Remove build artifacts"
	@echo ""
	@echo "  Docker:"
	@echo "    docker-build    - Build Docker image"
	@echo "    docker-up       - Start services with docker-compose"
	@echo "    docker-down     - Stop services with docker-compose"
	@echo ""
	@echo "  Helm:"
	@echo "    helm-lint       - Lint Helm chart"
	@echo "    helm-template   - Render chart templates"
	@echo "    helm-package    - Package chart into .tgz"
	@echo "    helm-push       - Push chart to GHCR"
	@echo "    helm-install    - Install chart to cluster"
	@echo "    helm-test       - Run Helm tests"
	@echo "    helm-all        - Full pipeline (lint/package/push)"
	@echo ""
	@echo "  Other:"
	@echo "    run             - Run the application locally"
	@echo "    security        - Run security checks"
	@echo "    deps            - Download and verify dependencies"
	@echo "    help            - Display this help message"

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

# Helm configuration
HELM_VERSION ?= $(VERSION)
HELM_CHART_DIR = deployments/helm/artifusion
HELM_REGISTRY = oci://ghcr.io/mainuli
HELM_CHART_NAME = artifusion
GITHUB_USER ?= mainuli

## helm-deps: Check Helm and dependencies
.PHONY: helm-deps
helm-deps:
	@command -v helm >/dev/null 2>&1 || { echo "❌ helm not found, install from https://helm.sh"; exit 1; }
	@command -v helm-docs >/dev/null 2>&1 || echo "⚠️  helm-docs not found (optional), install: brew install norwoodj/tap/helm-docs"
	@command -v kubeconform >/dev/null 2>&1 || echo "⚠️  kubeconform not found (optional), install: brew install kubeconform"

## helm-lint: Lint Helm chart with strict validation
.PHONY: helm-lint
helm-lint: helm-deps
	@echo "Linting Helm chart..."
	helm lint $(HELM_CHART_DIR) --strict
	@if command -v kubeconform >/dev/null 2>&1; then \
		echo "Validating templates with kubeconform..."; \
		helm template test $(HELM_CHART_DIR) | kubeconform -strict -summary; \
	fi

## helm-template: Render chart templates for inspection
.PHONY: helm-template
helm-template: helm-deps
	helm template $(HELM_CHART_NAME) $(HELM_CHART_DIR) \
		--namespace artifusion \
		--values $(HELM_CHART_DIR)/values.yaml

## helm-template-dev: Render with development values
.PHONY: helm-template-dev
helm-template-dev: helm-deps
	helm template $(HELM_CHART_NAME) $(HELM_CHART_DIR) \
		--values $(HELM_CHART_DIR)/values-dev.yaml

## helm-template-prod: Render with production values
.PHONY: helm-template-prod
helm-template-prod: helm-deps
	helm template $(HELM_CHART_NAME) $(HELM_CHART_DIR) \
		--values $(HELM_CHART_DIR)/values-prod.yaml

## helm-package: Package chart into .tgz
.PHONY: helm-package
helm-package: helm-deps
	@echo "Packaging Helm chart version $(HELM_VERSION)..."
	@mkdir -p ./dist
	helm package $(HELM_CHART_DIR) \
		--version $(HELM_VERSION) \
		--app-version $(VERSION) \
		--destination ./dist
	@echo "✓ Chart packaged: dist/$(HELM_CHART_NAME)-$(HELM_VERSION).tgz"

## helm-login: Login to GHCR for Helm chart push
.PHONY: helm-login
helm-login:
	@echo "Logging in to GHCR..."
	@echo "$$GITHUB_TOKEN" | helm registry login ghcr.io -u $(GITHUB_USER) --password-stdin

## helm-push: Push chart to GHCR
.PHONY: helm-push
helm-push: helm-package helm-login
	@echo "Pushing Helm chart to $(HELM_REGISTRY)..."
	helm push ./dist/$(HELM_CHART_NAME)-$(HELM_VERSION).tgz $(HELM_REGISTRY)
	@echo "✓ Chart pushed to $(HELM_REGISTRY)/$(HELM_CHART_NAME):$(HELM_VERSION)"

## helm-install: Install chart to current kubectl context
.PHONY: helm-install
helm-install: helm-deps
	helm install $(HELM_CHART_NAME) $(HELM_CHART_DIR) \
		--namespace artifusion --create-namespace \
		--values $(HELM_CHART_DIR)/values.yaml

## helm-upgrade: Upgrade existing installation
.PHONY: helm-upgrade
helm-upgrade: helm-deps
	helm upgrade $(HELM_CHART_NAME) $(HELM_CHART_DIR) \
		--namespace artifusion \
		--values $(HELM_CHART_DIR)/values.yaml \
		--wait

## helm-test: Run Helm tests
.PHONY: helm-test
helm-test:
	helm test $(HELM_CHART_NAME) --namespace artifusion

## helm-uninstall: Uninstall chart from cluster
.PHONY: helm-uninstall
helm-uninstall:
	helm uninstall $(HELM_CHART_NAME) --namespace artifusion

## helm-docs: Generate README.md from values.yaml
.PHONY: helm-docs
helm-docs:
	@if command -v helm-docs >/dev/null 2>&1; then \
		helm-docs $(HELM_CHART_DIR); \
		echo "✓ Documentation generated"; \
	else \
		echo "⚠️  helm-docs not installed, skipping"; \
	fi

## helm-all: Full pipeline: lint → package → push
.PHONY: helm-all
helm-all: helm-lint helm-package helm-push
	@echo "✓ Helm pipeline complete"

## helm-dry-run: Dry-run install for validation
.PHONY: helm-dry-run
helm-dry-run: helm-deps
	helm install $(HELM_CHART_NAME) $(HELM_CHART_DIR) \
		--namespace artifusion --create-namespace \
		--dry-run --debug
