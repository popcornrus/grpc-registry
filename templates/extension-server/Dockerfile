# Build stage
FROM golang:1.21-alpine AS builder

# Install essential build tools
RUN apk add --no-cache git

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Generate code from protocol buffers
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2 && \
    mkdir -p /usr/lib/protoc/bin && \
    apk add --no-cache protoc && \
    protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/extension/extension.proto

# Build the extension server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/extension-server .

# Final stage
FROM alpine:3.18

# Install certificates for TLS support
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/extension-server .

# Create a data directory
RUN mkdir -p /app/data

# Expose the gRPC port
EXPOSE 53000

# Run the extension server with auto-registration
ENTRYPOINT ["/app/extension-server"]
CMD ["--port=53000", "--data-dir=/app/data", "--auto-register=true", "--proxy-addr=proxy-manager:50051", "--extension-id=my-extension-server"]
