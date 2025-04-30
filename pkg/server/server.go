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
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/popcornrus/grpc-registry/pkg/manager"
	"github.com/popcornrus/grpc-registry/proto/pb"
)

// These are our local types that mirror what would be in the generated protobuf files

// LocalServerDetailsRequest represents a request to get server details
type LocalServerDetailsRequest struct {
	ServerId string // Optional server ID to filter by
}

// LocalServerDetailsResponse represents a response with server details
type LocalServerDetailsResponse struct {
	Servers []*LocalServerInfo // List of server details
}

// LocalServerInfo represents information about a server
type LocalServerInfo struct {
	ServerId  string            // Server ID
	Address   string            // Server address
	Connected bool              // Connection status
	Routes    []*LocalRouteInfo // Server routes/methods
}

// LocalRouteInfo represents information about a route/method
type LocalRouteInfo struct {
	Service string // Service name
	Method  string // Method name
}

// ProxyServer implements the ProxyService gRPC service
type ProxyServer struct {
	pb.UnimplementedProxyServiceServer
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
func (s *ProxyServer) RegisterServer(ctx context.Context, req *pb.RegisterServerRequest) (*pb.RegisterServerResponse, error) {
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
		opts = append(opts, manager.WithTimeout(time.Duration(req.TimeoutSeconds)*time.Second))
	}

	// Set max retries if specified
	if req.MaxRetries > 0 {
		opts = append(opts, manager.WithMaxRetries(int(req.MaxRetries)))
	}

	// Register the server
	err := s.Manager.RegisterServer(req.ServerId, req.Address, opts...)
	if err != nil {
		s.Logger.Errorf("Failed to register server %q: %v", req.ServerId, err)
		return &pb.RegisterServerResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.RegisterServerResponse{
		Success: true,
	}, nil
}

// RegisterRoute implements the RegisterRoute RPC method
func (s *ProxyServer) RegisterRoute(ctx context.Context, req *pb.RegisterRouteRequest) (*pb.RegisterRouteResponse, error) {
	s.Logger.Infof("Received RegisterRoute request for server %q with %d routes", req.ServerId, len(req.Routes))
	
	// Check if the server is registered
	_, err := s.Manager.GetConnection(req.ServerId)
	if err != nil {
		s.Logger.Errorf("Failed to get connection for server %q: %v", req.ServerId, err)
		return &pb.RegisterRouteResponse{
			Success: false,
			Error:   fmt.Sprintf("Server with ID %q not found", req.ServerId),
		}, nil
	}
	
	// Add each route to the server
	successCount := 0
	for _, route := range req.Routes {
		err := s.Manager.AddServerRoute(req.ServerId, route.Service, route.Method)
		if err != nil {
			s.Logger.Warnf("Failed to add route %s/%s to server %q: %v", 
				route.Service, route.Method, req.ServerId, err)
			continue
		}
		successCount++
	}
	
	// Return success if at least one route was added
	success := successCount > 0
	var errorMsg string
	if !success {
		errorMsg = "Failed to add any routes"
	} else if successCount < len(req.Routes) {
		errorMsg = fmt.Sprintf("Added %d of %d routes", successCount, len(req.Routes))
	}
	
	s.Logger.Infof("Added %d routes for server %q", successCount, req.ServerId)
	
	return &pb.RegisterRouteResponse{
		Success:    success,
		Error:      errorMsg,
		RouteCount: int32(successCount),
	}, nil
}

// RemoveServer implements the RemoveServer RPC method
func (s *ProxyServer) RemoveServer(ctx context.Context, req *pb.RemoveServerRequest) (*pb.RemoveServerResponse, error) {
	s.Logger.Infof("Received RemoveServer request for server %q", req.ServerId)

	// Remove the server
	err := s.Manager.RemoveServer(req.ServerId)
	if err != nil {
		s.Logger.Errorf("Failed to remove server %q: %v", req.ServerId, err)
		return &pb.RemoveServerResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.RemoveServerResponse{
		Success: true,
	}, nil
}

// ListServers implements the ListServers RPC method
func (s *ProxyServer) ListServers(ctx context.Context, req *pb.ListServersRequest) (*pb.ListServersResponse, error) {
	s.Logger.Info("Received ListServers request")

	// List all registered servers
	serverIDs := s.Manager.ListServers()

	return &pb.ListServersResponse{
		ServerIds: serverIDs,
	}, nil
}

// ManagerProtoServerInfo represents information about a server from the manager
type ManagerProtoServerInfo struct {
	ServerID  string                   // Server ID
	Address   string                   // Server address
	Connected bool                     // Connection status
	Routes    []*ManagerProtoRouteInfo // Server routes/methods
}

// ManagerProtoRouteInfo represents information about a gRPC route from the manager
type ManagerProtoRouteInfo struct {
	Service string // Service name
	Method  string // Method name
}

// Temporary struct definitions to make the code compile
// GetServerDetails implements the GetServerDetails RPC method
func (s *ProxyServer) GetServerDetails(ctx context.Context, req *pb.ServerDetailsRequest) (*pb.ServerDetailsResponse, error) {
	// Get the server ID from the request
	serverID := req.ServerId
	if serverID == "" {
		s.Logger.Info("Received GetServerDetails request for all servers")
	} else {
		s.Logger.Infof("Received GetServerDetails request for server %q", serverID)
	}

	// Get server details from the manager
	customServerInfos := s.Manager.GetServerDetails(serverID)

	// Convert manager server infos to pb.ServerInfo objects
	servers := make([]*pb.ServerInfo, 0, len(customServerInfos))
	for _, customInfo := range customServerInfos {
		pbInfo := &pb.ServerInfo{
			ServerId:  customInfo.ServerID,
			Address:   customInfo.Address,
			Connected: customInfo.Connected,
			Routes:    make([]*pb.RouteInfo, 0, len(customInfo.Routes)),
		}

		// Convert routes
		for _, customRoute := range customInfo.Routes {
			pbRoute := &pb.RouteInfo{
				Service: customRoute.Service,
				Method:  customRoute.Method,
			}
			pbInfo.Routes = append(pbInfo.Routes, pbRoute)
		}

		servers = append(servers, pbInfo)
	}

	// Return a response with the server details
	return &pb.ServerDetailsResponse{
		Servers: servers,
	}, nil
}

// SendRequest implements the SendRequest RPC method
func (s *ProxyServer) SendRequest(ctx context.Context, req *pb.ProxyRequest) (*pb.ProxyResponse, error) {
	s.Logger.Infof("Received SendRequest request for server %q, service %q, method %q",
		req.ServerId, req.Service, req.Method)

	// For raw binary data, we need a more robust approach to create a valid proto.Message
	// We'll create a properly formatted anypb.Any message with the correct type URL
	// This fixes the 'message is []uint8, want proto.Message' error
	typeURL := fmt.Sprintf("type.googleapis.com/%s.%s", req.Service, req.Method)

	// Create a new Any message that properly wraps the raw data
	anyMsg := &anypb.Any{
		TypeUrl: typeURL,
		Value:   req.Data,
	}

	// anyMsg itself implements the proto.Message interface
	// This ensures we pass a proper protobuf message to SendRequest

	// Send the request to the target server
	respMsg, err := s.Manager.SendRequest(req.ServerId, req.Service, req.Method, anyMsg)
	if err != nil {
		s.Logger.Errorf("Failed to send request to server %q: %v", req.ServerId, err)
		return &pb.ProxyResponse{
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
			return &pb.ProxyResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to marshal response: %v", err),
			}, nil
		}
	}

	return &pb.ProxyResponse{
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
	pb.RegisterProxyServiceServer(server, proxyServer)

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
