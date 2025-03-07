package testtool

import (
	"context"
	"log"
	"net"

	memberpb "streaming_video_service/pkg/proto/member"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"google.golang.org/grpc"
)

// SetupContainer 通用函式來啟動測試容器
func SetupContainer(ctx context.Context, req testcontainers.ContainerRequest) (testcontainers.Container, string, string, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", "", err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, "", "", err
	}

	// 轉換 ExposedPorts[0] 為 nat.Port
	natPort, err := nat.NewPort("tcp", req.ExposedPorts[0][:len(req.ExposedPorts[0])-4]) // 去掉 "/tcp"
	if err != nil {
		return nil, "", "", err
	}

	port, err := container.MappedPort(ctx, natPort)
	if err != nil {
		return nil, "", "", err
	}

	return container, host, port.Port(), nil
}

// StartMockMemberGRPCServer 通用函式來啟動測試grpc容器
func StartMockMemberGRPCServer() (*grpc.Server, string) {
	listener, err := net.Listen("tcp", ":0") // 隨機取得可用 Port
	if err != nil {
		log.Fatalf("❌ Failed to start gRPC listener: %v", err)
	}

	grpcServer := grpc.NewServer()
	mockMemberService := &MockMemberService{}
	// f(grpcServer, mockMemberService)
	memberpb.RegisterMemberServiceServer(grpcServer, mockMemberService)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("❌ Failed to start Mock gRPC Member Service: %v", err)
		}
	}()

	return grpcServer, listener.Addr().String()
}

//MockMemberService  Mock Member gRPC 服務
type MockMemberService struct {
	memberpb.UnimplementedMemberServiceServer
}