# gRPC Extension Server Template

This is a template for creating extension servers that can register with the gRPC Registry proxy manager.

## Project Structure

```
.
├── Dockerfile              # Docker build file for containerization
├── docker-compose.yml      # Docker Compose for local development
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
├── main.go                 # Main application entry point
├── cmd/                    # Command-line applications
│   └── client/             # Registration client for testing
│       └── main.go         # Client implementation
├── internal/               # Private application code
│   ├── config/             # Configuration handling
│   │   └── config.go       # Configuration implementation
│   └── service/            # Service implementation
│       └── extension.go    # Extension service implementation
└── proto/                  # Protocol buffer definitions
    └── extension/          # Extension service proto files
        └── extension.proto # Extension service definition
```

## Quick Start

1. Clone this template repository
2. Update the proto definitions if needed
3. Implement your extension service logic in `internal/service/extension.go`
4. Run locally: `go run main.go`
5. Build container: `docker build -t my-extension-server .`

## Auto-Registration

This template includes auto-registration with the gRPC Registry proxy manager. By default, it will attempt to register with the proxy manager at startup.

To configure registration:

```
go run main.go --proxy-addr=proxy-manager:50051 --extension-id=my-extension
```

## Docker Deployment

The template includes Docker configuration for easy deployment. You can run:

```
docker-compose up -d
```

Which will build and start the extension server, automatically registering it with the proxy manager.
