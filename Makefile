.PHONY: all clean proto install-tools help

# Project variables
PROTO_DIR := proto
GO_OUT_DIR := proto/pb
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
GOPATH := $(shell go env GOPATH)
PATH := $(GOPATH)/bin:$(PATH)

# Default target
all: proto

# Install necessary tools
install-tools:
	@echo "Installing protoc-gen-go and protoc-gen-go-grpc..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installed Go protoc plugins to $(GOPATH)/bin"
	@echo "Make sure $(GOPATH)/bin is in your PATH"

# Generate Go code from proto files
proto: $(PROTO_FILES)
	@echo "Generating Go code from proto files..."
	@mkdir -p $(GO_OUT_DIR)
	@echo "Using GOPATH: $(GOPATH)"
	@if [ ! -f "$(GOPATH)/bin/protoc-gen-go" ]; then \
		echo "Error: protoc-gen-go not found in $(GOPATH)/bin. Please run 'make install-tools'."; \
		exit 1; \
	fi
	@if [ ! -f "$(GOPATH)/bin/protoc-gen-go-grpc" ]; then \
		echo "Error: protoc-gen-go-grpc not found in $(GOPATH)/bin. Please run 'make install-tools'."; \
		exit 1; \
	fi
	PATH="$(GOPATH)/bin:$(PATH)" protoc -I=$(PROTO_DIR) \
		--go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_FILES)
	@echo "Proto files generated successfully in $(GO_OUT_DIR)"

# Clean generated files
clean:
	@echo "Cleaning generated proto files..."
	rm -f $(GO_OUT_DIR)/*.pb.go
	@echo "Clean completed"

# Show help
help:
	@echo "Available targets:"
	@echo "  all          : Generate all proto files (default)"
	@echo "  proto        : Generate Go code from proto files"
	@echo "  clean        : Remove generated files from $(GO_OUT_DIR)"
	@echo "  install-tools: Install required protoc plugins"
	@echo "  help         : Show this help message"