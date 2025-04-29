.PHONY: all clean proto install-tools

# Project variables
PROTO_DIR := proto
GO_OUT_DIR := .
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
GOPATH := $(shell go env GOPATH)
PATH := $(GOPATH)/bin:$(PATH)

# Default target
all: proto

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

# Clean generated files
clean:
	@echo "Cleaning generated proto files..."
	rm -f $(PROTO_DIR)/*.pb.go
	@echo "Clean completed"

# Show help
help:
	@echo "Available targets:"
	@echo "  all          : Generate all proto files (default)"
	@echo "  proto        : Generate Go code from proto files"
	@echo "  clean        : Remove generated files"
	@echo "  install-tools: Install required protoc plugins"
	@echo "  help         : Show this help message"
