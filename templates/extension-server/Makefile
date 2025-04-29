.PHONY: all build clean proto test run install-tools docker docker-compose

# Project variables
PROTO_DIR := proto/extension
GO_OUT_DIR := .
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
GOPATH := $(shell go env GOPATH)
PATH := $(GOPATH)/bin:$(PATH)
BINARY_NAME := extension-server
REGISTRY_REPO := github.com/popcornrus/grpc-registry

# Docker variables
DOCKER_IMAGE := my-extension-server
DOCKER_TAG := latest

# Default target
all: proto build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .

# Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install necessary tools
install-tools:
	@echo "Installing protoc-gen-go and protoc-gen-go-grpc..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
	@echo "Installed Go protoc plugins to $(GOPATH)/bin"
	@echo "Make sure $(GOPATH)/bin is in your PATH"

# Generate Go code from proto files
proto: $(PROTO_FILES)
	@echo "Generating Go code from proto files..."
	@mkdir -p $(GO_OUT_DIR)
	@echo "Using GOPATH: $(GOPATH)"
	@if [ ! -f "$(GOPATH)/bin/protoc-gen-go" ]; then \
		echo "protoc-gen-go not found in GOPATH/bin, install with 'make install-tools'"; \
		exit 1; \
	fi
	@if [ ! -f "$(GOPATH)/bin/protoc-gen-go-grpc" ]; then \
		echo "protoc-gen-go-grpc not found in GOPATH/bin, install with 'make install-tools'"; \
		exit 1; \
	fi
	PATH="$(GOPATH)/bin:$(PATH)" protoc -I=$(PROTO_DIR) \
		--go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)
	@echo "Proto files generated successfully"

# Build Docker image
docker:
	@echo "Building Docker image $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Run with Docker Compose
docker-compose:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

# Clean generated files
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f proto/extension/*.pb.go
	go clean

# Initialize a new project from this template
init:
	@echo "Initializing new project..."
	@read -p "Enter your module name (e.g., github.com/username/repo): " MODULE_NAME; \
	read -p "Enter your extension ID: " EXTENSION_ID; \
	read -p "Enter proxy registry repo (default: $(REGISTRY_REPO)): " REGISTRY_REPO_INPUT; \
	REGISTRY_REPO_FINAL=$${REGISTRY_REPO_INPUT:-$(REGISTRY_REPO)}; \
	find . -type f -name "*.go" -o -name "*.mod" -o -name "Dockerfile" -o -name "docker-compose.yml" | xargs sed -i "s|my-extension-server|$${MODULE_NAME}|g"; \
	find . -type f -name "*.go" -o -name "Dockerfile" -o -name "docker-compose.yml" | xargs sed -i "s|my-extension-server|$${EXTENSION_ID}|g"; \
	find . -type f -name "*.go" -o -name "*.mod" | xargs sed -i "s|github.com/popcornrus/grpc-registry|$${REGISTRY_REPO_FINAL}|g"; \
	go mod tidy
	@echo "Project initialized successfully. Don't forget to update the proto definitions if needed."
