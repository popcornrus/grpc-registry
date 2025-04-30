package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/popcornrus/grpc-registry/pkg/config"
)

// RouteInfo represents a gRPC route (service+method)
type RouteInfo struct {
	// Service name
	Service string

	// Method name
	Method string
}

// ServerConnection represents a connection to a gRPC server
type ServerConnection struct {
	// Server ID
	ID string

	// Server address
	Address string

	// gRPC connection
	Conn *grpc.ClientConn

	// Connection options
	Options *ConnectionOptions

	// Known routes for this server
	Routes []RouteInfo

	// Mutex for thread-safe access to routes
	RoutesMutex sync.RWMutex
}

// ConnectionOptions represents options for a gRPC connection
type ConnectionOptions struct {
	// Whether to use TLS
	UseTLS bool

	// TLS certificate file path (if UseTLS is true)
	TLSCertFile string

	// Connection timeout
	Timeout time.Duration

	// Max retry attempts
	MaxRetries int
}

// Option is a function that configures a ConnectionOptions object
type Option func(*ConnectionOptions)

// WithTLS configures a connection to use TLS with the given certificate file
func WithTLS(certFile string) Option {
	return func(o *ConnectionOptions) {
		o.UseTLS = true
		o.TLSCertFile = certFile
	}
}

// WithInsecure configures a connection to use insecure transport
func WithInsecure() Option {
	return func(o *ConnectionOptions) {
		o.UseTLS = false
		o.TLSCertFile = ""
	}
}

// WithTimeout configures a connection timeout
func WithTimeout(timeout time.Duration) Option {
	return func(o *ConnectionOptions) {
		o.Timeout = timeout
	}
}

// WithMaxRetries configures the maximum retry attempts
func WithMaxRetries(maxRetries int) Option {
	return func(o *ConnectionOptions) {
		o.MaxRetries = maxRetries
	}
}

// ProxyManager manages connections to gRPC servers
type ProxyManager struct {
	// Configuration
	Config *config.Config

	// Connections to gRPC servers (server ID -> connection)
	Connections map[string]*ServerConnection

	// Mutex for thread-safe access to the connections map
	Mutex sync.RWMutex

	// Logger
	Logger *logrus.Logger
}

// New creates a new ProxyManager
func New(cfg *config.Config) *ProxyManager {
	logger := logrus.New()

	// Configure logger based on log level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Enable JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &ProxyManager{
		Config:      cfg,
		Connections: make(map[string]*ServerConnection),
		Logger:      logger,
	}
}

// RegisterServer registers a new gRPC server
func (m *ProxyManager) RegisterServer(serverID, address string, opts ...Option) error {
	// Create default connection options from config
	options := &ConnectionOptions{
		UseTLS:      false,
		TLSCertFile: "",
		Timeout:     m.Config.DefaultTimeout,
		MaxRetries:  m.Config.DefaultMaxRetries,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// Check if server ID already exists
	m.Mutex.RLock()
	_, exists := m.Connections[serverID]
	m.Mutex.RUnlock()

	if exists {
		return fmt.Errorf("server with ID %q already registered", serverID)
	}

	// Create dial options for the connection
	dialOpts := []grpc.DialOption{
		grpc.WithBlock(),
	}

	// Configure transport security
	if options.UseTLS {
		creds, err := credentials.NewClientTLSFromFile(options.TLSCertFile, "")
		if err != nil {
			return fmt.Errorf("failed to create TLS credentials: %w", err)
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Create a context with timeout for connection
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// Connect to the server with retry
	var conn *grpc.ClientConn
	var err error

	for attempt := 0; attempt <= options.MaxRetries; attempt++ {
		conn, err = grpc.DialContext(ctx, address, dialOpts...)
		if err == nil {
			break
		}

		if attempt < options.MaxRetries {
			backoff := time.Duration(attempt+1) * time.Second
			m.Logger.Infof("Failed to connect to %s (attempt %d/%d), retrying in %v", address, attempt+1, options.MaxRetries, backoff)
			time.Sleep(backoff)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to server %s after %d attempts: %w", address, options.MaxRetries+1, err)
	}

	// Create a server connection
	serverConn := &ServerConnection{
		ID:      serverID,
		Address: address,
		Conn:    conn,
		Options: options,
		Routes:  []RouteInfo{},
	}

	// Store the connection
	m.Mutex.Lock()
	m.Connections[serverID] = serverConn
	m.Mutex.Unlock()

	m.Logger.Infof("Registered server %q at %s", serverID, address)

	// Try to discover routes (don't fail registration if this fails)
	go func() {
		if err := m.DiscoverServerRoutes(serverID); err != nil {
			m.Logger.Warnf("Failed to discover routes for server %q: %v", serverID, err)
		}
	}()

	return nil
}

// RemoveServer removes a registered gRPC server
func (m *ProxyManager) RemoveServer(serverID string) error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	// Check if server exists
	conn, exists := m.Connections[serverID]
	if !exists {
		return fmt.Errorf("server with ID %q not found", serverID)
	}

	// Close the connection
	if err := conn.Conn.Close(); err != nil {
		m.Logger.Warnf("Error closing connection to server %q: %v", serverID, err)
	}

	// Remove the connection from the map
	delete(m.Connections, serverID)

	m.Logger.Infof("Removed server %q", serverID)

	return nil
}

// ListServers lists all registered gRPC servers
func (m *ProxyManager) ListServers() []string {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	serverIDs := make([]string, 0, len(m.Connections))
	for id := range m.Connections {
		serverIDs = append(serverIDs, id)
	}

	return serverIDs
}

// DiscoverServerRoutes attempts to discover available routes for a server
func (m *ProxyManager) DiscoverServerRoutes(serverID string) error {
	// Get the connection
	conn, err := m.GetConnection(serverID)
	if err != nil {
		return err
	}

	// Note: In a real implementation, you'd use gRPC reflection to discover services
	// We're using a simplified approach here where routes are discovered as they're used

	// Try getting service info directly from the ClientConn
	// This is a best-effort approach as some servers may not support reflection
	// We'll add service and method pairs we know are supported
	// In a real implementation, you'd want to use the reflection service if available

	// For now, we'll record the service and method from each successful request
	// This will gradually build up the routes list as requests are made

	// Add some basic routes if we know them (from the proto files)
	knownRoutes := []RouteInfo{}

	// Record these routes
	conn.RoutesMutex.Lock()
	conn.Routes = append(conn.Routes, knownRoutes...)
	conn.RoutesMutex.Unlock()

	m.Logger.Infof("Discovered %d routes for server %q", len(knownRoutes), serverID)
	return nil
}

// AddServerRoute adds a route to a server's known routes if it doesn't already exist
func (m *ProxyManager) AddServerRoute(serverID, service, method string) error {
	// Get the connection
	conn, err := m.GetConnection(serverID)
	if err != nil {
		return err
	}

	// Create the route
	route := RouteInfo{
		Service: service,
		Method:  method,
	}

	// Check if route already exists
	conn.RoutesMutex.RLock()
	for _, r := range conn.Routes {
		if r.Service == service && r.Method == method {
			// Route already exists
			conn.RoutesMutex.RUnlock()
			return nil
		}
	}
	conn.RoutesMutex.RUnlock()

	// Add the route
	conn.RoutesMutex.Lock()
	conn.Routes = append(conn.Routes, route)
	conn.RoutesMutex.Unlock()

	m.Logger.Debugf("Added route %s/%s to server %q", service, method, serverID)
	return nil
}

// ProtoServerInfo represents detailed information about a server (matches ServerInfo in proto)
type ProtoServerInfo struct {
	ServerID  string            // Server ID
	Address   string            // Server address
	Connected bool              // Connection status
	Routes    []*ProtoRouteInfo // Server routes/methods
}

// ProtoRouteInfo represents information about a route/method (matches RouteInfo in proto)
type ProtoRouteInfo struct {
	Service string // Service name
	Method  string // Method name
}

// GetServerDetails gets detailed information about servers
func (m *ProxyManager) GetServerDetails(serverID string) []*ProtoServerInfo {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	var result []*ProtoServerInfo

	// If serverID is provided, get details for that server only
	if serverID != "" {
		conn, exists := m.Connections[serverID]
		if !exists {
			return result
		}

		info := m.createServerInfo(conn)
		result = append(result, info)
		return result
	}

	// Otherwise, get details for all servers
	result = make([]*ProtoServerInfo, 0, len(m.Connections))
	for _, conn := range m.Connections {
		info := m.createServerInfo(conn)
		result = append(result, info)
	}

	return result
}

// createServerInfo creates a ServerInfo protobuf message from a ServerConnection
func (m *ProxyManager) createServerInfo(conn *ServerConnection) *ProtoServerInfo {
	// Check connection status
	connected := conn.Conn.GetState() != connectivity.Shutdown

	// Create the server info
	info := &ProtoServerInfo{
		ServerID:  conn.ID,
		Address:   conn.Address,
		Connected: connected,
		Routes:    make([]*ProtoRouteInfo, 0, len(conn.Routes)),
	}

	// Add routes
	conn.RoutesMutex.RLock()
	for _, route := range conn.Routes {
		route := &ProtoRouteInfo{
			Service: route.Service,
			Method:  route.Method,
		}
		info.Routes = append(info.Routes, route)
	}
	conn.RoutesMutex.RUnlock()

	return info
}

// GetConnection gets a connection by server ID
func (m *ProxyManager) GetConnection(serverID string) (*ServerConnection, error) {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()

	conn, exists := m.Connections[serverID]
	if !exists {
		return nil, fmt.Errorf("server with ID %q not found", serverID)
	}

	return conn, nil
}

// SendRequest sends a request to a specific gRPC server and receives a response
func (m *ProxyManager) SendRequest(serverID string, service, method string, request proto.Message) (proto.Message, error) {
	// Look up the server in the registry
	conn, err := m.GetConnection(serverID)
	if err != nil {
		return nil, err
	}

	// Set up the context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// We'll create a response buffer when we make the call

	// Ensure the service name includes a package
	formattedService := service
	if !strings.Contains(service, ".") {
		// If no package prefix is provided, use "extension" as the default package
		formattedService = "extension." + service
		m.Logger.Infof("Added default package to service name: %s", formattedService)
	}

	// Create the full method string in gRPC format: /package.Service/Method
	fullMethod := fmt.Sprintf("/%s/%s", formattedService, method)

	m.Logger.Infof("Using gRPC method: %s", fullMethod)
	// Send the request and get the response
	var resp proto.Message
	var lastErr error

	for attempt := 0; attempt <= conn.Options.MaxRetries; attempt++ {
		// Create a buffer to store the response
		outBuffer := &anypb.Any{}

		// Invoke needs proto.Message, not raw bytes
		// The request must be a proto.Message and the response will be populated in outBuffer
		lastErr = conn.Conn.Invoke(ctx, fullMethod, request, outBuffer)
		if lastErr == nil {
			// Use the populated outBuffer as our response
			resp = outBuffer

			// Record this successful route
			m.AddServerRoute(serverID, service, method)
			break
		}

		// Check if we should retry
		if shouldRetry(lastErr) && attempt < conn.Options.MaxRetries {
			backoff := time.Duration(attempt+1) * time.Second
			m.Logger.Infof("Failed to send request to %s (attempt %d/%d), retrying in %v: %v",
				conn.Address, attempt+1, conn.Options.MaxRetries, backoff, lastErr)
			time.Sleep(backoff)
		} else {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to send request to server %s after %d attempts: last err from Invoke: %w",
			conn.Address, conn.Options.MaxRetries+1, lastErr)
	}

	return resp, nil
}

// shouldRetry determines if a request should be retried based on the error
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	// Get the gRPC status code
	s, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error, don't retry
		return false
	}

	// List of status codes that should be retried
	retryableCodes := []codes.Code{
		codes.Unavailable,
		codes.ResourceExhausted,
		codes.DeadlineExceeded,
	}

	for _, code := range retryableCodes {
		if s.Code() == code {
			return true
		}
	}

	return false
}

// Close closes all connections
func (m *ProxyManager) Close() error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	var errs []error

	for id, conn := range m.Connections {
		if err := conn.Conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("error closing connection to server %q: %w", id, err))
		}
	}

	// Clear the connections map
	m.Connections = make(map[string]*ServerConnection)

	if len(errs) > 0 {
		// Return the first error
		return errs[0]
	}

	return nil
}
