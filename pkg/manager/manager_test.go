package manager

import (
	"testing"
	"time"

	"grpc-registry/pkg/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Port:              50051,
		LogLevel:          "info",
		DefaultTimeout:    10 * time.Second,
		DefaultMaxRetries: 3,
	}

	mgr := New(cfg)

	if mgr == nil {
		t.Fatal("Expected non-nil manager")
	}

	if mgr.Config != cfg {
		t.Errorf("Expected Config to be %v, got %v", cfg, mgr.Config)
	}

	if mgr.Connections == nil {
		t.Error("Expected non-nil Connections map")
	}

	if mgr.Logger == nil {
		t.Error("Expected non-nil Logger")
	}
}

func TestRegisterServer(t *testing.T) {
	cfg := &config.Config{
		Port:              50051,
		LogLevel:          "info",
		DefaultTimeout:    10 * time.Millisecond, // Short timeout for testing
		DefaultMaxRetries: 0,                     // No retries for testing
	}

	mgr := New(cfg)

	// Test registering a server with invalid address (should fail quickly)
	err := mgr.RegisterServer("test-server", "invalid-address:1234", WithInsecure())
	if err == nil {
		t.Error("Expected error when registering server with invalid address")
	}

	// Test registering a server with same ID twice
	// This is a bit tricky to test without a real server, so we'll mock it
	// by directly inserting a connection into the map

	// Create a dummy connection
	mgr.Mutex.Lock()
	mgr.Connections["test-server"] = &ServerConnection{
		ID:      "test-server",
		Address: "localhost:1234",
		Conn:    nil, // Nil connection for testing
		Options: &ConnectionOptions{},
	}
	mgr.Mutex.Unlock()

	// Try to register with the same ID
	err = mgr.RegisterServer("test-server", "localhost:5678", WithInsecure())
	if err == nil {
		t.Error("Expected error when registering server with existing ID")
	}
}

func TestListServers(t *testing.T) {
	cfg := &config.Config{
		Port:              50051,
		LogLevel:          "info",
		DefaultTimeout:    10 * time.Second,
		DefaultMaxRetries: 3,
	}

	mgr := New(cfg)

	// Create some dummy connections
	mgr.Mutex.Lock()
	mgr.Connections["server1"] = &ServerConnection{
		ID:      "server1",
		Address: "localhost:1234",
		Conn:    nil, // Nil connection for testing
		Options: &ConnectionOptions{},
	}
	mgr.Connections["server2"] = &ServerConnection{
		ID:      "server2",
		Address: "localhost:5678",
		Conn:    nil, // Nil connection for testing
		Options: &ConnectionOptions{},
	}
	mgr.Mutex.Unlock()

	// List servers
	servers := mgr.ListServers()

	// Check if both servers are in the list
	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}

	// Check if server IDs are in the list
	server1Found := false
	server2Found := false
	for _, id := range servers {
		if id == "server1" {
			server1Found = true
		}
		if id == "server2" {
			server2Found = true
		}
	}

	if !server1Found {
		t.Error("Expected server1 in the list")
	}
	if !server2Found {
		t.Error("Expected server2 in the list")
	}
}

func TestRemoveServer(t *testing.T) {
	cfg := &config.Config{
		Port:              50051,
		LogLevel:          "info",
		DefaultTimeout:    10 * time.Second,
		DefaultMaxRetries: 3,
	}

	mgr := New(cfg)

	// Test removing a non-existent server
	err := mgr.RemoveServer("non-existent")
	if err == nil {
		t.Error("Expected error when removing non-existent server")
	}

	// Create a dummy connection
	mgr.Mutex.Lock()
	mgr.Connections["test-server"] = &ServerConnection{
		ID:      "test-server",
		Address: "localhost:1234",
		Conn:    nil, // Nil connection for testing
		Options: &ConnectionOptions{},
	}
	mgr.Mutex.Unlock()

	// Remove the server
	err = mgr.RemoveServer("test-server")
	// We expect an error because the connection is nil, but the server should still be removed

	// Check if the server was removed
	mgr.Mutex.RLock()
	_, exists := mgr.Connections["test-server"]
	mgr.Mutex.RUnlock()

	if exists {
		t.Error("Expected server to be removed")
	}
}

func TestGetConnection(t *testing.T) {
	cfg := &config.Config{
		Port:              50051,
		LogLevel:          "info",
		DefaultTimeout:    10 * time.Second,
		DefaultMaxRetries: 3,
	}

	mgr := New(cfg)

	// Test getting a non-existent connection
	_, err := mgr.GetConnection("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent connection")
	}

	// Create a dummy connection
	mgr.Mutex.Lock()
	mgr.Connections["test-server"] = &ServerConnection{
		ID:      "test-server",
		Address: "localhost:1234",
		Conn:    nil, // Nil connection for testing
		Options: &ConnectionOptions{},
	}
	mgr.Mutex.Unlock()

	// Get the connection
	conn, err := mgr.GetConnection("test-server")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if conn == nil {
		t.Error("Expected non-nil connection")
	}

	if conn.ID != "test-server" {
		t.Errorf("Expected ID to be test-server, got %s", conn.ID)
	}

	if conn.Address != "localhost:1234" {
		t.Errorf("Expected Address to be localhost:1234, got %s", conn.Address)
	}
}

func TestConnectionOptions(t *testing.T) {
	// Test WithTLS
	options := &ConnectionOptions{}
	WithTLS("cert.pem")(options)

	if !options.UseTLS {
		t.Error("Expected UseTLS to be true")
	}

	if options.TLSCertFile != "cert.pem" {
		t.Errorf("Expected TLSCertFile to be cert.pem, got %s", options.TLSCertFile)
	}

	// Test WithInsecure
	options = &ConnectionOptions{}
	WithInsecure()(options)

	if options.UseTLS {
		t.Error("Expected UseTLS to be false")
	}

	if options.TLSCertFile != "" {
		t.Errorf("Expected TLSCertFile to be empty, got %s", options.TLSCertFile)
	}

	// Test WithTimeout
	options = &ConnectionOptions{}
	WithTimeout(5 * time.Second)(options)

	if options.Timeout != 5*time.Second {
		t.Errorf("Expected Timeout to be 5s, got %v", options.Timeout)
	}

	// Test WithMaxRetries
	options = &ConnectionOptions{}
	WithMaxRetries(5)(options)

	if options.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries to be 5, got %d", options.MaxRetries)
	}
}

func TestShouldRetry(t *testing.T) {
	// Test nil error
	if shouldRetry(nil) {
		t.Error("Expected shouldRetry(nil) to return false")
	}
}
