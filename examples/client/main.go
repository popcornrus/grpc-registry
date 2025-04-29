package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	extensionpb "grpc-registry/proto/extension"
	proxypb "grpc-registry/proto/proxy"
)

var (
	proxyAddr     = flag.String("proxy", "localhost:50051", "Proxy manager address")
	extensionID   = flag.String("extension-id", "rrweb-extension", "Extension ID")
	extensionAddr = flag.String("extension-addr", "localhost:53000", "Extension server address")
	action        = flag.String("action", "register", "Action to perform: register, track, get-data, or list")
	userID        = flag.String("user-id", "user123", "User ID for tracking")
)

func main() {
	flag.Parse()

	// Connect to the proxy manager
	conn, err := grpc.Dial(*proxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer conn.Close()

	// Create a proxy service client
	proxyClient := proxypb.NewProxyServiceClient(conn)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform the requested action
	switch *action {
	case "register":
		// Register the extension server
		resp, err := proxyClient.RegisterServer(ctx, &proxypb.RegisterServerRequest{
			ServerId:       *extensionID,
			Address:        *extensionAddr,
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
		fmt.Printf("Successfully registered server %s at %s\n", *extensionID, *extensionAddr)

	case "list":
		// List registered servers
		resp, err := proxyClient.ListServers(ctx, &proxypb.ListServersRequest{})
		if err != nil {
			log.Fatalf("Failed to list servers: %v", err)
		}
		fmt.Println("Registered servers:")
		for _, id := range resp.ServerIds {
			fmt.Printf("- %s\n", id)
		}

	case "track":
		// Send a track request to the extension via the proxy
		trackReq := &extensionpb.TrackRequest{
			UserId: *userID,
			Data:   []byte("Example tracking data for user behavior"),
		}

		// Serialize the request
		reqData, err := proto.Marshal(trackReq)
		if err != nil {
			log.Fatalf("Failed to serialize request: %v", err)
		}

		// Send the request to the proxy
		proxyReq := &proxypb.ProxyRequest{
			ServerId: *extensionID,
			Service:  "extension.ExtensionService",
			Method:   "Track",
			Data:     reqData,
		}

		proxyResp, err := proxyClient.SendRequest(ctx, proxyReq)
		if err != nil {
			log.Fatalf("Failed to send request: %v", err)
		}

		if !proxyResp.Success {
			log.Fatalf("Request failed: %s", proxyResp.Error)
		}

		// Deserialize the response
		trackResp := &extensionpb.TrackResponse{}
		if err := proto.Unmarshal(proxyResp.Data, trackResp); err != nil {
			log.Fatalf("Failed to deserialize response: %v", err)
		}

		fmt.Printf("Track response: success=%v, session-id=%s\n", trackResp.Success, trackResp.SessionId)

	case "get-data":
		// Send a get-data request to the extension via the proxy
		dataReq := &extensionpb.GetDataRequest{
			UserId:    *userID,
			SessionId: flag.Arg(0), // Get session ID from the first non-flag argument
		}

		// Serialize the request
		reqData, err := proto.Marshal(dataReq)
		if err != nil {
			log.Fatalf("Failed to serialize request: %v", err)
		}

		// Send the request to the proxy
		proxyReq := &proxypb.ProxyRequest{
			ServerId: *extensionID,
			Service:  "extension.ExtensionService",
			Method:   "GetData",
			Data:     reqData,
		}

		proxyResp, err := proxyClient.SendRequest(ctx, proxyReq)
		if err != nil {
			log.Fatalf("Failed to send request: %v", err)
		}

		if !proxyResp.Success {
			log.Fatalf("Request failed: %s", proxyResp.Error)
		}

		// Deserialize the response
		dataResp := &extensionpb.GetDataResponse{}
		if err := proto.Unmarshal(proxyResp.Data, dataResp); err != nil {
			log.Fatalf("Failed to deserialize response: %v", err)
		}

		fmt.Printf("GetData response: success=%v, file-location=%s\n", dataResp.Success, dataResp.FileLocation)
		fmt.Printf("Data: %s\n", string(dataResp.Data))

	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}
