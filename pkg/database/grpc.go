package database

import (
	"fmt"
	"streaming_video_service/pkg/logger"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// CreateGRPCClient create grpc client
func CreateGRPCClient(grpcIP string) (*grpc.ClientConn, error) {
	client, err := grpc.Dial(grpcIP, grpc.WithInsecure())
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("Failed to connect: %v", err))
	}

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("connection did not become READY within 5 minutes")
		case <-ticker.C:
			state := client.GetState()
			logger.Log.Info(fmt.Sprintf("Connection[%s] state: %s", grpcIP, state))
			if state == connectivity.Ready {
				logger.Log.Info("Connection is READY")
				return client, nil
			}
		}
	}
}
