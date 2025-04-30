package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	extensionpb "github.com/popcornrus/grpc-registry/proto/extension"
	proxypb "github.com/popcornrus/grpc-registry/proto/proxy"
)

var (
	port          = flag.Int("port", 53000, "The server port")
	dataDir       = flag.String("data-dir", "/tmp/rrweb-data", "Directory to store tracking data")
	autoRegister  = flag.Bool("auto-register", true, "Whether to auto-register with the proxy manager")
	proxyAddr     = flag.String("proxy-addr", "proxy-manager:50051", "Proxy manager address")
	extensionID   = flag.String("extension-id", "rrweb-extension", "Extension ID for registration")
	extensionAddr = flag.String("extension-addr", "", "Extension server address for registration (defaults to hostname:port)")
)

// extensionServer implements the ExtensionService
type extensionServer struct {
	extensionpb.UnimplementedExtensionServiceServer
	logger  *logrus.Logger
	dataDir string
}

// Track implements the Track method
func (s *extensionServer) Track(ctx context.Context, req *extensionpb.TrackRequest) (*extensionpb.TrackResponse, error) {
	s.logger.Infof("Received Track request for user %s", req.UserId)

	// Create a session ID based on time
	sessionID := fmt.Sprintf("%s-%d", req.UserId, time.Now().Unix())

	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		s.logger.Errorf("Failed to create data directory: %v", err)
		return &extensionpb.TrackResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create data directory: %v", err),
		}, nil
	}

	// Create a file to store the tracking data
	filePath := filepath.Join(s.dataDir, fmt.Sprintf("%s.bin", sessionID))
	file, err := os.Create(filePath)
	if err != nil {
		s.logger.Errorf("Failed to create file: %v", err)
		return &extensionpb.TrackResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create file: %v", err),
		}, nil
	}
	defer file.Close()

	// Write the tracking data to the file
	if _, err := file.Write(req.Data); err != nil {
		s.logger.Errorf("Failed to write data: %v", err)
		return &extensionpb.TrackResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to write data: %v", err),
		}, nil
	}

	s.logger.Infof("Tracking data saved to %s", filePath)

	return &extensionpb.TrackResponse{
		Success:   true,
		SessionId: sessionID,
	}, nil
}

// GetData implements the GetData method
func (s *extensionServer) GetData(ctx context.Context, req *extensionpb.GetDataRequest) (*extensionpb.GetDataResponse, error) {
	s.logger.Infof("Received GetData request for user %s, session %s", req.UserId, req.SessionId)

	// If session ID is provided, get data for that session
	if req.SessionId != "" {
		filePath := filepath.Join(s.dataDir, fmt.Sprintf("%s.bin", req.SessionId))

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			s.logger.Errorf("File not found: %s", filePath)
			return &extensionpb.GetDataResponse{
				Success: false,
				Error:   fmt.Sprintf("Session data not found: %s", req.SessionId),
			}, nil
		}

		// Read the file for small files (for large files, you would just return the file location)
		data, err := os.ReadFile(filePath)
		if err != nil {
			s.logger.Errorf("Failed to read file: %v", err)
			return &extensionpb.GetDataResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to read file: %v", err),
			}, nil
		}

		return &extensionpb.GetDataResponse{
			Success:      true,
			FileLocation: filePath,
			Data:         data,
		}, nil
	}

	// For this example, just return the file location for the user's latest session
	// In a real implementation, you would scan the directory for matching files
	// and potentially return a list or the latest one

	return &extensionpb.GetDataResponse{
		Success: false,
		Error:   "Session ID is required",
	}, nil
}

// registerWithProxyManager registers this extension server with the proxy manager
func registerWithProxyManager(logger *logrus.Logger, address string) error {
	// Connect to the proxy manager
	conn, err := grpc.Dial(*proxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to proxy manager: %v", err)
	}
	defer conn.Close()

	// Create a proxy service client
	proxyClient := proxypb.NewProxyServiceClient(conn)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Register the extension server
	resp, err := proxyClient.RegisterServer(ctx, &proxypb.RegisterServerRequest{
		ServerId:       *extensionID,
		Address:        address,
		UseTls:         false,
		TimeoutSeconds: 30,
		MaxRetries:     5,
	})

	if err != nil {
		return fmt.Errorf("failed to register with proxy manager: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration failed: %s", resp.Error)
	}

	return nil
}

func main() {
	flag.Parse()

	// Create a logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Create the data directory if it doesn't exist
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		logger.Fatalf("Failed to create data directory: %v", err)
	}

	// Create a TCP listener
	addr := fmt.Sprintf(":%d", *port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	// Determine registration address
	registerAddr := *extensionAddr
	if registerAddr == "" {
		// Get hostname for registration
		hostname, err := os.Hostname()
		if err != nil {
			logger.Warnf("Could not get hostname for registration, using 'extension-server': %v", err)
			hostname = "extension-server"
		}
		registerAddr = fmt.Sprintf("%s:%d", hostname, *port)
	}

	// Create a gRPC server
	server := grpc.NewServer()

	// Create and register the extension service
	extensionService := &extensionServer{
		logger:  logger,
		dataDir: *dataDir,
	}
	extensionpb.RegisterExtensionServiceServer(server, extensionService)

	// Enable reflection for debugging
	reflection.Register(server)

	// Auto-register with the proxy manager if enabled
	if *autoRegister {
		logger.Infof("Attempting to auto-register with proxy manager at %s", *proxyAddr)
		logger.Infof("Registration address: %s", registerAddr)

		// Start a goroutine to handle registration
		go func() {
			// Add a small delay to ensure proxy manager is ready
			time.Sleep(5 * time.Second)

			// Attempt registration with retry
			maxRetries := 5
			retryInterval := 5 * time.Second

			for attempt := 1; attempt <= maxRetries; attempt++ {
				err := registerWithProxyManager(logger, registerAddr)
				if err == nil {
					logger.Infof("Successfully registered with proxy manager (attempt %d/%d)", attempt, maxRetries)
					break
				}

				logger.Warnf("Registration attempt %d/%d failed: %v", attempt, maxRetries, err)
				if attempt < maxRetries {
					logger.Infof("Retrying in %v...", retryInterval)
					time.Sleep(retryInterval)
				} else {
					logger.Errorf("Failed to register with proxy manager after %d attempts", maxRetries)
				}
			}
		}()
	}

	// Start a goroutine to handle shutdown signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		logger.Info("Received shutdown signal, gracefully stopping server")
		server.GracefulStop()
	}()

	// Start the server
	logger.Infof("Starting extension server on %s", addr)
	if err := server.Serve(lis); err != nil {
		logger.Fatalf("Failed to serve: %v", err)
	}
}
