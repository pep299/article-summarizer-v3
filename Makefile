# Article Summarizer v3 Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
SERVER_BINARY=article-summarizer-server
CLI_BINARY=article-summarizer-cli

# Build directories
BUILD_DIR=./build
CMD_SERVER_DIR=./cmd/server
CMD_CLI_DIR=./cmd/cli

# Version
VERSION?=v3.0.0
COMMIT?=$(shell git rev-parse --short HEAD)
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# LDFLAGS for version info
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: all build clean test deps dev run-server run-cli help

# Default target
all: clean deps test build

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Build all binaries
build: build-server build-cli

# Build server binary
build-server:
	@echo "Building server..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY) $(CMD_SERVER_DIR)

# Build CLI binary
build-cli:
	@echo "Building CLI..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY) $(CMD_CLI_DIR)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -race -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Development mode
dev: deps
	@echo "Running in development mode..."
	$(GOCMD) run $(CMD_SERVER_DIR)/main.go

# Run server binary
run-server: build-server
	@echo "Running server..."
	./$(BUILD_DIR)/$(SERVER_BINARY)

# Run CLI binary
run-cli: build-cli
	@echo "Running CLI..."
	./$(BUILD_DIR)/$(CLI_BINARY)

# Process RSS feeds using CLI
process:
	$(GOCMD) run $(CMD_CLI_DIR)/main.go -cmd=process

# Test RSS feeds
test-rss:
	$(GOCMD) run $(CMD_CLI_DIR)/main.go -cmd=test-rss

# Test Gemini API
test-gemini:
	$(GOCMD) run $(CMD_CLI_DIR)/main.go -cmd=test-gemini -url="https://example.com" -title="Test Article"

# Test Slack integration
test-slack:
	$(GOCMD) run $(CMD_CLI_DIR)/main.go -cmd=test-slack -message="Test message from Article Summarizer v3"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Run all quality checks
check: fmt vet test

# Create .env template
env-template:
	@echo "Creating .env template..."
	@echo "# Article Summarizer v3 Configuration" > .env.template
	@echo "PORT=8080" >> .env.template
	@echo "HOST=0.0.0.0" >> .env.template
	@echo "" >> .env.template
	@echo "# Gemini API Configuration" >> .env.template
	@echo "GEMINI_API_KEY=your_gemini_api_key_here" >> .env.template
	@echo "GEMINI_MODEL=gemini-1.5-flash" >> .env.template
	@echo "" >> .env.template
	@echo "# Slack Configuration" >> .env.template
	@echo "SLACK_WEBHOOK_URL=your_slack_webhook_url_here" >> .env.template
	@echo "SLACK_CHANNEL=#general" >> .env.template
	@echo "" >> .env.template
	@echo "# RSS Configuration" >> .env.template
	@echo "RSS_FEEDS=https://b.hatena.ne.jp/hotentry/it.rss,https://lobste.rs/rss" >> .env.template
	@echo "UPDATE_INTERVAL_MINUTES=30" >> .env.template
	@echo "" >> .env.template
	@echo "# Cache Configuration" >> .env.template
	@echo "CACHE_TYPE=memory" >> .env.template
	@echo "CACHE_DURATION_HOURS=24" >> .env.template
	@echo "" >> .env.template
	@echo "# Rate Limiting" >> .env.template
	@echo "MAX_CONCURRENT_REQUESTS=5" >> .env.template
	@echo "Template created at .env.template"

# Help
help:
	@echo "Article Summarizer v3 - Available Commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build         - Build all binaries"
	@echo "  make build-server  - Build server binary only"
	@echo "  make build-cli     - Build CLI binary only"
	@echo ""
	@echo "Development Commands:"
	@echo "  make dev           - Run in development mode"
	@echo "  make deps          - Install dependencies"
	@echo "  make clean         - Clean build artifacts"
	@echo ""
	@echo "Testing Commands:"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make test-race     - Run tests with race detection"
	@echo ""
	@echo "Quality Commands:"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Vet code"
	@echo "  make check         - Run all quality checks"
	@echo ""
	@echo "Runtime Commands:"
	@echo "  make run-server    - Run server"
	@echo "  make run-cli       - Run CLI"
	@echo "  make process       - Process RSS feeds"
	@echo "  make test-rss      - Test RSS feeds"
	@echo "  make test-gemini   - Test Gemini API"
	@echo "  make test-slack    - Test Slack integration"
	@echo ""
	@echo "Configuration:"
	@echo "  make env-template  - Create .env template file"
