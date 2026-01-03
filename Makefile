.PHONY: help build run test clean docker-build docker-run docker-stop docker-clean \
        proto cluster-up cluster-down cluster-logs cluster-status demo

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags="-w -s -X 'kv-service/internal/config.Version=$(VERSION)' -X 'kv-service/internal/config.GitCommit=$(GIT_COMMIT)' -X 'kv-service/internal/config.BuildTime=$(BUILD_TIME)'"

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "  Build & Run:"
	@echo "    build         - Build the application"
	@echo "    run           - Run the application (standalone)"
	@echo "    clean         - Clean build artifacts"
	@echo "    proto         - Generate protobuf code"
	@echo "    deps          - Install dependencies"
	@echo ""
	@echo "  Docker:"
	@echo "    docker-build  - Build Docker image"
	@echo "    docker-run    - Run single instance with Docker"
	@echo "    docker-stop   - Stop Docker containers"
	@echo "    docker-clean  - Remove Docker containers and volumes"
	@echo ""
	@echo "  Cluster (1 master + 2 slaves):"
	@echo "    cluster-up    - Start the cluster"
	@echo "    cluster-down  - Stop the cluster"
	@echo "    cluster-logs  - View cluster logs"
	@echo "    cluster-status- Check cluster health"
	@echo "    demo          - Run replication demo"

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/replication.proto

# Build the application
build:
	@echo "Building kv-service $(VERSION)..."
	@go build $(LDFLAGS) -o kv-service cmd/main.go

# Run the application (standalone master mode)
run: build
	@echo "Starting kv-service as standalone master..."
	@ROLE=master NODE_ID=standalone ./kv-service

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f kv-service
	@rm -rf data/

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy

# Build Docker image
docker-build:
	@echo "Building Docker image $(VERSION)..."
	@docker build --build-arg VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg BUILD_TIME=$(BUILD_TIME) -t kv-service:$(VERSION) -t kv-service:latest .

# Run single instance with Docker
docker-run: docker-build
	@echo "Starting single instance..."
	@docker run -d --name kv-service -p 3300:3300 -e ROLE=master kv-service:latest

# Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	@docker stop kv-service 2>/dev/null || true
	@docker rm kv-service 2>/dev/null || true

# Remove Docker containers and volumes
docker-clean: cluster-down
	@echo "Cleaning Docker resources..."
	@docker rmi kv-service:latest 2>/dev/null || true
	@docker rmi kv-service:$(VERSION) 2>/dev/null || true
	@rm -rf data/

# Start cluster (1 master + 2 slaves)
cluster-up: docker-build
	@echo "Starting cluster (1 master + 2 slaves)..."
	@docker-compose up -d
	@echo ""
	@echo "Cluster started:"
	@echo "  Master:  http://localhost:3300"
	@echo "  Slave 1: http://localhost:3301"
	@echo "  Slave 2: http://localhost:3302"
	@echo ""
	@echo "Run 'make cluster-status' to check health"
	@echo "Run 'make demo' to see replication in action"

# Stop cluster
cluster-down:
	@echo "Stopping cluster..."
	@docker-compose down -v

# View cluster logs
cluster-logs:
	@docker-compose logs -f

# Check cluster health
cluster-status:
	@echo "Checking cluster status..."
	@echo ""
	@echo "=== Master (localhost:3300) ==="
	@curl -s http://localhost:3300/cluster 2>/dev/null | jq . || echo "Master not responding"
	@echo ""
	@echo "=== Slave 1 (localhost:3301) ==="
	@curl -s http://localhost:3301/health 2>/dev/null | jq . || echo "Slave 1 not responding"
	@echo ""
	@echo "=== Slave 2 (localhost:3302) ==="
	@curl -s http://localhost:3302/health 2>/dev/null | jq . || echo "Slave 2 not responding"

# Run replication demo
demo:
	@echo "Running replication demo..."
	@./scripts/replication_demo.sh
