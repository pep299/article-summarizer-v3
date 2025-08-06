.PHONY: build test clean run dev fmt vet lint check-env config

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Version parameters
GO_VERSION := $(shell grep "^go " go.mod | cut -d' ' -f2)
GO_RUNTIME_ID := go$(shell echo $(GO_VERSION) | tr -d '.')

# Build targets
BUILD_DIR=../build
CLI_BINARY=$(BUILD_DIR)/cli
SERVER_BINARY=$(BUILD_DIR)/server

# Default target
all: test build

# Check required environment variables
check-env:
	@if [ -z "$(FUNCTION_TARGET)" ]; then \
		echo "❌ Error: FUNCTION_TARGET is not set"; \
		echo "Please set FUNCTION_TARGET in your .env file or environment"; \
		exit 1; \
	fi
	@echo "✅ FUNCTION_TARGET = $(FUNCTION_TARGET)"

# Build server binary
build-server:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(SERVER_BINARY) ./cmd/server

# Build both binaries
build: build-server

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

# E2E tests (uses existing API keys)
test-e2e:
	@echo "🚀 Running E2E tests with existing API keys..."
	$(GOTEST) -v ./test -run TestE2E_ -timeout 10m

# E2E test setup instructions
test-e2e-setup:
	$(GOTEST) -v ./test -run TestE2E_SetupInstructions

# Clean
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run server (Cloud Function mode)
run: check-env
	@echo "Starting server with FUNCTION_TARGET=$(FUNCTION_TARGET)"
	$(GOCMD) run ./cmd/server

# Development server with hot reload (Cloud Function mode)
dev: check-env
	@echo "Starting dev server with FUNCTION_TARGET=$(FUNCTION_TARGET)"
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

# Docker image configuration
PROJECT_ID=gen-lang-client-0715048106
IMAGE_NAME=asia-northeast1-docker.pkg.dev/$(PROJECT_ID)/article-summarizer/article-summarizer-v3

# Build for deployment (Linux target)
build-for-deploy:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd/server

# Docker build
docker-build: build-for-deploy
	docker build -t $(IMAGE_NAME) .

# Docker push
docker-push: docker-build
	@echo "🔐 Configuring Docker authentication..."
	gcloud auth configure-docker asia-northeast1-docker.pkg.dev --quiet
	docker push $(IMAGE_NAME)

# Deploy using pre-built image (fast method)
deploy-image: docker-push
	@echo "🚀 Deploying to Cloud Run service..."
	gcloud run deploy article-summarizer-v3 \
		--image $(IMAGE_NAME) \
		--memory 512Mi \
		--timeout 540s \
		--region asia-northeast1 \
		--service-account article-summarizer-sa@gen-lang-client-0715048106.iam.gserviceaccount.com \
		--update-secrets "GEMINI_API_KEY=GEMINI_API_KEY:latest,SLACK_BOT_TOKEN=SLACK_BOT_TOKEN:latest,WEBHOOK_AUTH_TOKEN=WEBHOOK_AUTH_TOKEN:latest" \
		--set-env-vars "FUNCTION_TARGET=SummarizeArticles" \
		--allow-unauthenticated \
		--project gen-lang-client-0715048106
	@echo "✅ Deployment completed!"

# All checks
check: fmt vet lint test-race security

# Show current configuration
config: check-env
	@echo "Current configuration:"
	@echo "  FUNCTION_TARGET = $(FUNCTION_TARGET)"
	@echo "  GO_ENV = $(GO_ENV)"
	@echo "  PROJECT_ID = $(PROJECT_ID)"

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build server binary"
	@echo "  test        - Run tests"
	@echo "  test-race   - Run tests with race detection"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-e2e    - Run E2E tests (uses existing API keys)"
	@echo "  test-e2e-setup - Show E2E test setup instructions"
	@echo "  clean       - Clean build artifacts"
	@echo "  run         - Run server in Cloud Function mode"
	@echo "  dev         - Run development server with hot reload"
	@echo "  deploy      - Deploy to Google Cloud Functions"
	@echo "  check-env   - Check required environment variables"
	@echo "  config      - Show current configuration"
	@echo "  fmt         - Format code"
	@echo "  vet         - Vet code"
	@echo "  lint        - Lint code (requires golangci-lint)"
	@echo "  deps        - Install and tidy dependencies"
	@echo "  update      - Update dependencies"
	@echo "  generate    - Run go generate"
	@echo "  security    - Run security checks (requires gosec)"
	@echo "  check       - Run all checks (fmt, vet, lint, test-race, security)"
	@echo "  help        - Show this help message"
