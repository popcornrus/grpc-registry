# Build stage
FROM golang:1.23-alpine AS builder

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

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/grpc-registry ./cmd/server

# Development stage (for live reloading with air)
FROM golang:1.23-alpine AS dev

# Install essential tools and air
RUN apk add --no-cache git && \
    go install github.com/air-verse/air@latest

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Expose the gRPC port
EXPOSE 50051

# Run air for live reloading
ENTRYPOINT ["air", "-c", ".air.toml"]

# Final stage (production)
FROM alpine:3.18

# Add non-root user for security
RUN adduser -D -g '' appuser

# Install certificates for TLS support
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/grpc-registry .

# Create a directory for TLS certificates
RUN mkdir -p /app/certs && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose the gRPC port
EXPOSE 50051

# Run the proxy manager
ENTRYPOINT ["/app/grpc-registry"]