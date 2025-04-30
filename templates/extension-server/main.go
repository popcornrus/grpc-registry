package main

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
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"my-extension-server/internal/config"
	"my-extension-server/internal/service"
	pb "my-extension-server/proto/extension"
)

var (
	cfg               = config.NewConfig()
	logger            = logrus.New()
	extensionAddr     string
	extensionHostname string
)

// registerWithProxyManager registers this extension server with the proxy manager
func registerWithProxyManager() error {
	logger.Infof("Attempting to register with proxy manager at %s", cfg.ProxyAddr)

	// Connect to the proxy manager
	conn, err := grpc.Dial(cfg.ProxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to proxy manager: %v", err)
	}
	defer conn.Close()

	// Create a proxy service client
	proxyClient := pb.NewProxyServiceClient(conn)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.RegistrationTimeoutSeconds)*time.Second)
	defer cancel()

	// Create the registration address
	registerAddr := extensionAddr
	if cfg.RegistrationAddr != "" {
		registerAddr = cfg.RegistrationAddr
	}

	logger.Infof("Registering as %s with address %s", cfg.ExtensionID, registerAddr)

	// Register the extension server
	resp, err := proxyClient.RegisterServer(ctx, &pb.RegisterServerRequest{
		ServerId:       cfg.ExtensionID,
		Address:        registerAddr,
		UseTls:         cfg.UseTLS,
		TlsCertFile:    cfg.TLSCertFile,
		TimeoutSeconds: int32(cfg.ClientTimeoutSeconds),
		MaxRetries:     int32(cfg.MaxRetries),
	})

	if err != nil {
		return fmt.Errorf("failed to register with proxy manager: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration failed: %s", resp.Error)
	}

	logger.Infof("Successfully registered with proxy manager")
	return nil
}

func main() {
	// Parse config from flags/env
	cfg.ParseFlags()

	// Configure logger
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Determine hostname and address for registration
	extensionHostname, err = os.Hostname()
	if err != nil {
		extensionHostname = "localhost"
		logger.Warnf("Could not get hostname, using 'localhost': %v", err)
	}

	// Create data directory if needed
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		logger.Fatalf("Failed to create data directory: %v", err)
	}

	// Create a TCP listener
	addr := fmt.Sprintf(":%d", cfg.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	extensionAddr = fmt.Sprintf("%s:%d", extensionHostname, cfg.Port)

	// Create a gRPC server
	var grpcServer *grpc.Server
	if cfg.UseTLS {
		// Configure TLS (implementation omitted for brevity)
		logger.Fatalf("TLS not implemented in template")
	} else {
		grpcServer = grpc.NewServer()
	}

	// Create the extension service implementation
	extensionService := service.NewExtensionService(logger, cfg)
	pb.RegisterExtensionServiceServer(grpcServer, extensionService)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Auto-register with the proxy manager if enabled
	if cfg.AutoRegister {
		go func() {
			// Add a delay to ensure proxy manager is ready
			time.Sleep(cfg.RegistrationDelaySeconds * time.Second)

			// Attempt registration with retry
			for attempt := 1; attempt <= cfg.RegistrationMaxRetries; attempt++ {
				err := registerWithProxyManager()
				if err == nil {
					break
				}

				logger.Warnf("Registration attempt %d/%d failed: %v",
					attempt, cfg.RegistrationMaxRetries, err)

				if attempt < cfg.RegistrationMaxRetries {
					retryDelay := time.Duration(attempt) * time.Second
					logger.Infof("Retrying in %v...", retryDelay)
					time.Sleep(retryDelay)
				} else {
					logger.Errorf("Failed to register with proxy manager after %d attempts",
						cfg.RegistrationMaxRetries)
				}
			}
		}()
	}

	// Start a goroutine for graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh

		logger.Info("Received shutdown signal, gracefully stopping server")
		grpcServer.GracefulStop()
	}()

	// Start serving requests
	logger.Infof("Extension server starting on %s", addr)
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatalf("Failed to serve: %v", err)
	}
}
