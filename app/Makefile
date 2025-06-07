.PHONY: build test clean run-cli run-server dev fmt vet lint

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build targets
BUILD_DIR=../build
CLI_BINARY=$(BUILD_DIR)/cli
SERVER_BINARY=$(BUILD_DIR)/server

# Default target
all: test build

# Build CLI binary
build-cli:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(CLI_BINARY) ./cmd/cli

# Build server binary
build-server:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(SERVER_BINARY) ./cmd/server

# Build both binaries
build: build-cli build-server

# Test
test:
	$(GOTEST) -v ./...

# Test with race detection
test-race:
	$(GOTEST) -race -v ./...

# Test with coverage
test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run CLI
run-cli:
	$(GOCMD) run ./cmd/cli

# Run server
run-server:
	$(GOCMD) run ./cmd/server

# Development server with hot reload (requires air)
dev:
	air -c .air.toml || $(GOCMD) run ./cmd/server

# Format code
fmt:
	$(GOFMT) ./...

# Vet code
vet:
	$(GOCMD) vet ./...

# Lint (requires golangci-lint)
lint:
	golangci-lint run

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
update:
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

# Generate (if you have code generation)
generate:
	$(GOCMD) generate ./...

# Security check (requires gosec)
security:
	gosec ./...

# All checks
check: fmt vet lint test-race security

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build both CLI and server binaries"
	@echo "  build-cli   - Build CLI binary only"
	@echo "  build-server- Build server binary only"
	@echo "  test        - Run tests"
	@echo "  test-race   - Run tests with race detection"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean       - Clean build artifacts"
	@echo "  run-cli     - Run CLI application"
	@echo "  run-server  - Run server application"
	@echo "  dev         - Run development server with hot reload"
	@echo "  fmt         - Format code"
	@echo "  vet         - Vet code"
	@echo "  lint        - Lint code (requires golangci-lint)"
	@echo "  deps        - Install and tidy dependencies"
	@echo "  update      - Update dependencies"
	@echo "  generate    - Run go generate"
	@echo "  security    - Run security checks (requires gosec)"
	@echo "  check       - Run all checks (fmt, vet, lint, test-race, security)"
	@echo "  check-env   - Check development environment setup"
	@echo "  help        - Show this help message"

# Environment check
.PHONY: check-env
check-env:
	@echo "🔍 Checking development environment..."
	@echo ""
	@echo "1. Go version:"
	@eval "$$(mise activate bash)" && go version || echo "❌ Go not found"
	@echo ""
	@echo "2. Google Cloud authentication:"
	@gcloud auth list --format="value(account)" | head -1 | sed 's/^/✅ Authenticated as: /' || echo "❌ Not authenticated to Google Cloud"
	@echo ""
	@echo "3. GCS bucket access:"
	@gsutil ls gs://article-summarizer-processed-articles/ >/dev/null 2>&1 && echo "✅ GCS bucket accessible" || echo "❌ Cannot access GCS bucket"
	@echo ""
	@echo "Environment check complete!"
