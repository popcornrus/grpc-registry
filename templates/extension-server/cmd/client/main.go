package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	extensionpb "my-extension-server/proto/extension"
)

var (
	serverAddr = flag.String("server", "localhost:53000", "Extension server address")
	action     = flag.String("action", "health", "Action to perform: track, get, health")
	userId     = flag.String("user-id", "test-user", "User ID for tracking")
	sessionId  = flag.String("session-id", "", "Session ID for getting data")
	data       = flag.String("data", "Test tracking data", "Data to track")
)

func main() {
	flag.Parse()

	// Connect to the extension server
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to extension server: %v", err)
	}
	defer conn.Close()

	// Create an extension service client
	client := extensionpb.NewExtensionServiceClient(conn)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform the requested action
	switch *action {
	case "track":
		// Send a tracking request
		resp, err := client.Track(ctx, &extensionpb.TrackRequest{
			UserId: *userId,
			Data:   []byte(*data),
		})
		if err != nil {
			log.Fatalf("Track request failed: %v", err)
		}

		if !resp.Success {
			log.Fatalf("Track request failed: %s", resp.Error)
		}

		fmt.Printf("Successfully tracked data for user %s\n", *userId)
		fmt.Printf("Session ID: %s\n", resp.SessionId)

	case "get":
		// Check if session ID is provided
		if *sessionId == "" {
			log.Fatalf("Session ID is required for get action")
		}

		// Send a get data request
		resp, err := client.GetData(ctx, &extensionpb.GetDataRequest{
			SessionId: *sessionId,
		})
		if err != nil {
			log.Fatalf("GetData request failed: %v", err)
		}

		if !resp.Success {
			log.Fatalf("GetData request failed: %s", resp.Error)
		}

		fmt.Printf("Successfully retrieved data for session %s\n", *sessionId)
		fmt.Printf("Data: %s\n", string(resp.Data))

	case "health":
		// Send a healthcheck request
		resp, err := client.Healthcheck(ctx, &extensionpb.HealthcheckRequest{})
		if err != nil {
			log.Fatalf("Healthcheck request failed: %v", err)
		}

		fmt.Printf("Extension server health status: %s\n", resp.Status)
		fmt.Printf("Version: %s\n", resp.Version)

	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}
