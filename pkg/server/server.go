package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"

	"grpc-registry/pkg/manager"
	proxypb "grpc-registry/proto/proxy"
)

// ProxyServer implements the ProxyService gRPC service
type ProxyServer struct {
	proxypb.UnimplementedProxyServiceServer
	Manager *manager.ProxyManager
	Logger  *logrus.Logger
}

// NewProxyServer creates a new ProxyServer
func NewProxyServer(mgr *manager.ProxyManager) *ProxyServer {
	return &ProxyServer{
		Manager: mgr,
		Logger:  mgr.Logger,
	}
}

// RegisterServer implements the RegisterServer RPC method
func (s *ProxyServer) RegisterServer(ctx context.Context, req *proxypb.RegisterServerRequest) (*proxypb.RegisterServerResponse, error) {
	s.Logger.Infof("Received RegisterServer request for server %q at %s", req.ServerId, req.Address)

	// Create connection options based on the request
	var opts []manager.Option
	if req.UseTls {
		opts = append(opts, manager.WithTLS(req.TlsCertFile))
	} else {
		opts = append(opts, manager.WithInsecure())
	}

	// Set timeout if specified
	if req.TimeoutSeconds > 0 {
		opts = append(opts, manager.WithTimeout(time.Duration(req.TimeoutSeconds) * time.Second))
	}

	// Set max retries if specified
	if req.MaxRetries > 0 {
		opts = append(opts, manager.WithMaxRetries(int(req.MaxRetries)))
	}

	// Register the server
	err := s.Manager.RegisterServer(req.ServerId, req.Address, opts...)
	if err != nil {
		s.Logger.Errorf("Failed to register server %q: %v", req.ServerId, err)
		return &proxypb.RegisterServerResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &proxypb.RegisterServerResponse{
		Success: true,
	}, nil
}

// RemoveServer implements the RemoveServer RPC method
func (s *ProxyServer) RemoveServer(ctx context.Context, req *proxypb.RemoveServerRequest) (*proxypb.RemoveServerResponse, error) {
	s.Logger.Infof("Received RemoveServer request for server %q", req.ServerId)

	// Remove the server
	err := s.Manager.RemoveServer(req.ServerId)
	if err != nil {
		s.Logger.Errorf("Failed to remove server %q: %v", req.ServerId, err)
		return &proxypb.RemoveServerResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &proxypb.RemoveServerResponse{
		Success: true,
	}, nil
}

// ListServers implements the ListServers RPC method
func (s *ProxyServer) ListServers(ctx context.Context, req *proxypb.ListServersRequest) (*proxypb.ListServersResponse, error) {
	s.Logger.Info("Received ListServers request")

	// List all registered servers
	serverIDs := s.Manager.ListServers()

	return &proxypb.ListServersResponse{
		ServerIds: serverIDs,
	}, nil
}

// SendRequest implements the SendRequest RPC method
func (s *ProxyServer) SendRequest(ctx context.Context, req *proxypb.ProxyRequest) (*proxypb.ProxyResponse, error) {
	s.Logger.Infof("Received SendRequest request for server %q, service %q, method %q",
		req.ServerId, req.Service, req.Method)

	// Create a generic proto.Message from the request data
	// In a real implementation, you'd need to know the expected request type
	// For this example, we'll use a simple Any message to represent the request data
	reqMsg := &proxypb.ProxyRequest{
		Data: req.Data,
	}

	// Send the request to the target server
	respMsg, err := s.Manager.SendRequest(req.ServerId, req.Service, req.Method, reqMsg)
	if err != nil {
		s.Logger.Errorf("Failed to send request to server %q: %v", req.ServerId, err)
		return &proxypb.ProxyResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Extract the response data
	var respData []byte
	if respMsg != nil {
		respData, err = proto.Marshal(respMsg)
		if err != nil {
			s.Logger.Errorf("Failed to marshal response: %v", err)
			return &proxypb.ProxyResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to marshal response: %v", err),
			}, nil
		}
	}

	return &proxypb.ProxyResponse{
		Success: true,
		Data:    respData,
	}, nil
}

// StartServer starts the gRPC server
func StartServer(mgr *manager.ProxyManager) error {
	// Create a new gRPC server
	var opts []grpc.ServerOption

	// Configure TLS if enabled
	if mgr.Config.TLSCertFile != "" && mgr.Config.TLSKeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(mgr.Config.TLSCertFile, mgr.Config.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to create server TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	// Create the gRPC server
	server := grpc.NewServer(opts...)

	// Create the proxy service server
	proxyServer := NewProxyServer(mgr)

	// Register the service
	proxypb.RegisterProxyServiceServer(server, proxyServer)

	// Create a TCP listener
	addr := fmt.Sprintf(":%d", mgr.Config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Start a goroutine to handle shutdown signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		mgr.Logger.Info("Received shutdown signal, gracefully stopping server")
		server.GracefulStop()
	}()

	// Start the server
	mgr.Logger.Infof("Starting gRPC server on %s", addr)
	if err := server.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}
