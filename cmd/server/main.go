package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"grpc-registry/pkg/config"
	"grpc-registry/pkg/manager"
	"grpc-registry/pkg/server"
)

func main() {
	// Create a logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Load configuration
	cfg := config.New()

	// Configure logger level
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.Infof("Starting proxy manager on port %d", cfg.Port)

	// Create a proxy manager
	mgr := manager.New(cfg)

	// Start the server
	if err := server.StartServer(mgr); err != nil {
		logger.Errorf("Server error: %v", err)
		os.Exit(1)
	}
}
