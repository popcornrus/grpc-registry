package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"my-extension-server/internal/config"
	extensionpb "my-extension-server/proto/extension"
)

// ExtensionService implements the extension service gRPC interface
type ExtensionService struct {
	extensionpb.UnimplementedExtensionServiceServer
	logger *logrus.Logger
	config *config.Config
}

// NewExtensionService creates a new extension service instance
func NewExtensionService(logger *logrus.Logger, cfg *config.Config) *ExtensionService {
	return &ExtensionService{
		logger: logger,
		config: cfg,
	}
}

// Track implements the Track method for recording user behavior data
func (s *ExtensionService) Track(ctx context.Context, req *extensionpb.TrackRequest) (*extensionpb.TrackResponse, error) {
	s.logger.Infof("Received Track request for user %s", req.UserId)

	// Input validation
	if req.UserId == "" {
		return &extensionpb.TrackResponse{
			Success: false,
			Error:   "User ID is required",
		}, status.Error(codes.InvalidArgument, "User ID is required")
	}

	if len(req.Data) == 0 {
		return &extensionpb.TrackResponse{
			Success: false,
			Error:   "Track data is required",
		}, status.Error(codes.InvalidArgument, "Track data is required")
	}

	// Create a session ID based on time
	sessionID := fmt.Sprintf("%s-%d", req.UserId, time.Now().Unix())

	// Ensure data directory exists
	if err := os.MkdirAll(s.config.DataDir, 0755); err != nil {
		s.logger.Errorf("Failed to create data directory: %v", err)
		return &extensionpb.TrackResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create data directory: %v", err),
		}, status.Error(codes.Internal, "Failed to create data directory")
	}

	// Create a file to store the tracking data
	filePath := filepath.Join(s.config.DataDir, fmt.Sprintf("%s.bin", sessionID))
	file, err := os.Create(filePath)
	if err != nil {
		s.logger.Errorf("Failed to create data file: %v", err)
		return &extensionpb.TrackResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create data file: %v", err),
		}, status.Error(codes.Internal, "Failed to create data file")
	}
	defer file.Close()

	// Write the data to the file
	_, err = file.Write(req.Data)
	if err != nil {
		s.logger.Errorf("Failed to write data: %v", err)
		return &extensionpb.TrackResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to write data: %v", err),
		}, status.Error(codes.Internal, "Failed to write data")
	}

	s.logger.Infof("Successfully stored tracking data for user %s with session %s", req.UserId, sessionID)

	// Return success response with session ID
	return &extensionpb.TrackResponse{
		Success:   true,
		SessionId: sessionID,
	}, nil
}

// GetData implements the GetData method for retrieving user behavior data
func (s *ExtensionService) GetData(ctx context.Context, req *extensionpb.GetDataRequest) (*extensionpb.GetDataResponse, error) {
	s.logger.Infof("Received GetData request for session %s", req.SessionId)

	// Input validation
	if req.SessionId == "" {
		return &extensionpb.GetDataResponse{
			Success: false,
			Error:   "Session ID is required",
		}, status.Error(codes.InvalidArgument, "Session ID is required")
	}

	// Construct the file path
	filePath := filepath.Join(s.config.DataDir, fmt.Sprintf("%s.bin", req.SessionId))

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		s.logger.Warnf("Session data not found: %s", req.SessionId)
		return &extensionpb.GetDataResponse{
			Success: false,
			Error:   fmt.Sprintf("Session data not found: %s", req.SessionId),
		}, status.Error(codes.NotFound, "Session data not found")
	}

	// Read the data from the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		s.logger.Errorf("Failed to read data: %v", err)
		return &extensionpb.GetDataResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to read data: %v", err),
		}, status.Error(codes.Internal, "Failed to read data")
	}

	s.logger.Infof("Successfully retrieved data for session %s (%d bytes)", req.SessionId, len(data))

	// Return the data
	return &extensionpb.GetDataResponse{
		Success: true,
		Data:    data,
	}, nil
}

// Healthcheck implements a simple health check method
func (s *ExtensionService) Healthcheck(ctx context.Context, req *extensionpb.HealthcheckRequest) (*extensionpb.HealthcheckResponse, error) {
	return &extensionpb.HealthcheckResponse{
		Status:  "OK",
		Version: "1.0.0",
	}, nil
}
