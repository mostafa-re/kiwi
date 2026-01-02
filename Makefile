.PHONY: help build run test clean docker-build docker-run docker-stop docker-clean

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags="-w -s -X 'kv-service/internal/config.Version=$(VERSION)' -X 'kv-service/internal/config.GitCommit=$(GIT_COMMIT)' -X 'kv-service/internal/config.BuildTime=$(BUILD_TIME)'"

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  docker-stop   - Stop Docker containers"
	@echo "  docker-clean  - Remove Docker containers and volumes"

# Build the application
build:
	@echo "Building kv-service $(VERSION)..."
	@go build $(LDFLAGS) -o kv-service cmd/main.go

# Run the application
run: build
	@echo "Starting kv-service..."
	@./kv-service

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f kv-service
	@rm -rf data/

# Build Docker image
docker-build:
	@echo "Building Docker image $(VERSION)..."
	@docker build --build-arg VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg BUILD_TIME=$(BUILD_TIME) -t kv-service:$(VERSION) -t kv-service:latest .

# Run with Docker Compose
docker-run:
	@echo "Starting with Docker Compose..."
	@docker-compose up -d

# Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

# Remove Docker containers and volumes
docker-clean:
	@echo "Cleaning Docker resources..."
	@docker-compose down -v
	@docker rmi kv-service:latest 2>/dev/null || true

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
