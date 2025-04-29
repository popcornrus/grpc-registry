# gRPC Proxy Manager

A Go package that serves as a proxy manager between a main application and its extensions, facilitating communication via gRPC. This package manages connections to multiple gRPC servers representing extensions and allows the main application to send/receive data to/from specific gRPC servers.

## Features

- Manage connections to multiple gRPC servers
- Send requests to specific gRPC servers and receive responses
- Support dynamic registration and removal of gRPC servers at runtime
- Handle connection errors and retries gracefully
- Thread-safe management of gRPC connections
- Support for secure (TLS) and insecure gRPC connections
- Configuration via environment variables
- Docker support for easy deployment

## Architecture

The gRPC Proxy Manager acts as an intermediary between a main application and multiple extension servers:

```
                 ┌────────────┐
                 │   Main     │
                 │ Application│
                 └─────┬──────┘
                       │
                       │ gRPC
                       │
                 ┌─────▼──────┐
                 │   Proxy    │
                 │  Manager   │
                 └──┬───┬───┬─┘
                    │   │   │
                    │   │   │ gRPC
                    │   │   │
         ┌──────────┴┐  │  ┌┴──────────┐
         │ Extension │  │  │ Extension │
         │ Server 1  │  │  │ Server N  │
         └───────────┘  │  └───────────┘
                        │
                    ┌───▼────────┐
                    │ Extension  │
                    │ Server 2   │
                    └────────────┘
```

## Installation

### Using Go

```bash
go get grpc-registry
```

### Using Docker

```bash
# Build the Docker image
docker build -t grpc-registry .

# Run the Docker container
docker run -p 50051:50051 grpc-registry
```

### Using Docker Compose

```bash
# Start both the proxy manager and an example extension server
docker-compose up

# Run in the background
docker-compose up -d
```

## Usage

### As a Library

#### Starting the Proxy Manager

```go
import (
    "grpc-registry/pkg/config"
    "grpc-registry/pkg/manager"
    "grpc-registry/pkg/server"
)

func main() {
    // Load configuration
    cfg := config.New()
    
    // Create a proxy manager
    mgr := manager.New(cfg)
    
    // Start the proxy manager server
    if err := server.StartServer(mgr); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

#### Registering a gRPC Server

```go
// Register a new gRPC server with insecure connection
err := mgr.RegisterServer("extension1", "localhost:53000", manager.WithInsecure())
if err != nil {
    log.Fatalf("Failed to register server: %v", err)
}

// Register a new gRPC server with TLS
err := mgr.RegisterServer("secure-extension", "localhost:53001", manager.WithTLS("/path/to/cert.pem"))
if err != nil {
    log.Fatalf("Failed to register server: %v", err)
}
```

#### Sending a Request

```go
// Send a request to a specific server
resp, err := mgr.SendRequest("extension1", "extension.ExtensionService", "Track", trackRequest)
if err != nil {
    log.Fatalf("Failed to send request: %v", err)
}

// Use the response
log.Printf("Response: %+v", resp)
```

#### Removing a Server

```go
// Remove a server
err := mgr.RemoveServer("extension1")
if err != nil {
    log.Fatalf("Failed to remove server: %v", err)
}
```

#### Listing Registered Servers

```go
// List all registered servers
servers := mgr.ListServers()
for _, server := range servers {
    log.Printf("Server: %s", server)
}
```

### Using the gRPC API

#### Registering a Server

```go
// Create a gRPC client
conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
if err != nil {
    log.Fatalf("Failed to connect: %v", err)
}
defer conn.Close()

client := proxypb.NewProxyServiceClient(conn)

// Register a server
resp, err := client.RegisterServer(context.Background(), &proxypb.RegisterServerRequest{
    ServerId:       "extension1",
    Address:        "localhost:53000",
    UseTls:         false,
    TimeoutSeconds: 10,
    MaxRetries:     3,
})
if err != nil {
    log.Fatalf("Failed to register server: %v", err)
}
if !resp.Success {
    log.Fatalf("Failed to register server: %s", resp.Error)
}
```

#### Sending a Request via Proxy

```go
// Create a request
req := &extensionpb.TrackRequest{
    UserId: "user123",
    Data:   []byte("Example tracking data"),
}

// Serialize the request
reqData, err := req.Marshal()
if err != nil {
    log.Fatalf("Failed to serialize request: %v", err)
}

// Send the request through the proxy
proxyReq := &proxypb.ProxyRequest{
    ServerId: "extension1",
    Service:  "extension.ExtensionService",
    Method:   "Track",
    Data:     reqData,
}

proxyResp, err := client.SendRequest(context.Background(), proxyReq)
if err != nil {
    log.Fatalf("Failed to send request: %v", err)
}

if !proxyResp.Success {
    log.Fatalf("Request failed: %s", proxyResp.Error)
}

// Deserialize the response
trackResp := &extensionpb.TrackResponse{}
if err := trackResp.Unmarshal(proxyResp.Data); err != nil {
    log.Fatalf("Failed to deserialize response: %v", err)
}

log.Printf("Track response: success=%v, session-id=%s", trackResp.Success, trackResp.SessionId)
```

## Configuration

The proxy manager can be configured using environment variables:

| Variable                 | Description                                | Default   |
|--------------------------|--------------------------------------------|-----------| 
| `PROXY_PORT`             | Port for the proxy manager's gRPC server   | 50051     |
| `LOG_LEVEL`              | Logging level (debug, info, warn, error)   | info      |
| `TLS_CERT`               | Path to TLS certificate file               | ""        |
| `TLS_KEY`                | Path to TLS key file                       | ""        |
| `DEFAULT_TIMEOUT_SECONDS`| Default timeout for gRPC connections       | 10        |
| `DEFAULT_MAX_RETRIES`    | Default max retries for gRPC connections   | 3         |

## Docker Compose

An example `docker-compose.yml` is provided to demonstrate running the proxy manager alongside a mock extension gRPC server. The compose file:

1. Starts the proxy manager on port 50051
2. Starts an example extension server on port 53000
3. Creates a Docker network for communication between services
4. Configures environment variables for both services
5. Sets up volumes for persistent data storage

To run:

```bash
docker-compose up
```

## Running the Examples

### Running the Example Extension Server

```bash
go run examples/extension/main.go --port=53000 --data-dir=/tmp/rrweb-data
```

### Using the Example Client

```bash
# Register the extension server
go run examples/client/main.go --action=register --extension-id=rrweb-extension --extension-addr=localhost:53000

# List registered servers
go run examples/client/main.go --action=list

# Send a tracking request
go run examples/client/main.go --action=track --user-id=user123

# Get tracking data (provide session ID as argument)
go run examples/client/main.go --action=get-data --user-id=user123 session-id-here
```

## API Documentation

### ProxyManager

- `New(config *config.Config) *ProxyManager`: Create a new proxy manager
- `RegisterServer(serverID string, address string, opts ...Option) error`: Register a gRPC server
- `SendRequest(serverID string, service, method string, request proto.Message) (proto.Message, error)`: Send a request to a specific server
- `RemoveServer(serverID string) error`: Remove a registered server
- `ListServers() []string`: List all registered servers
- `GetConnection(serverID string) (*ServerConnection, error)`: Get a connection by server ID
- `Close() error`: Close all connections

### Connection Options

- `WithTLS(certFile string) Option`: Configure a connection to use TLS
- `WithInsecure() Option`: Configure a connection to use insecure transport
- `WithTimeout(timeout time.Duration) Option`: Configure connection timeout
- `WithMaxRetries(maxRetries int) Option`: Configure maximum retry attempts

## Testing

To run the tests:

```bash
go test -v ./...
```

## Caddy Integration

A sample Caddyfile is provided to demonstrate how to route traffic to the proxy manager's gRPC server. This Caddyfile extends the provided configuration that routes `/grpc/*` to `go-app:53000` by adding a new route for the proxy manager:

```
# Route requests to the proxy manager
route /proxy/* {
    uri strip_prefix /proxy
    reverse_proxy proxy-manager:50051 {
        transport http {
            versions h2c
        }
    }
}
```

## Security Considerations

- The Docker image runs as a non-root user for improved security
- TLS support is included for secure communication
- Configuration via environment variables avoids hardcoded secrets
- The Docker image is built using multi-stage builds to minimize size and attack surface

## License

MIT
