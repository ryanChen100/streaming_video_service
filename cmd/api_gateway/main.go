package main

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "streaming_video_service/cmd/api_gateway/docs" // 引入生成的 Swagger 文档
	"streaming_video_service/internal/api/handlers"
	"streaming_video_service/internal/api/router"
	"streaming_video_service/pkg/config"
	"streaming_video_service/pkg/logger"
	memberpb "streaming_video_service/pkg/proto/member"

	"github.com/gofiber/fiber/v2"
	fiber_log "github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

func main() {
	logger.Log = logger.Initialize(config.EnvConfig.APIGateway, config.EnvConfig.APIGatewayLogPath)
	cfg := config.LoadConfig[config.APIGateway](config.EnvConfig.APIGateway, config.EnvConfig.APIGatewayYAMLPath)
	// 建立 gRPC 连接
	client, err := grpc.Dial(cfg.MemberService.Name+":"+cfg.MemberService.Port, grpc.WithInsecure())
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("Failed to connect: %v", err))
	}
	defer client.Close()

	// 检查连接状态
	go func() {
		for {
			state := client.GetState()
			logger.Log.Info(fmt.Sprintf("Connection state: %s", state))
			if state == connectivity.Ready {
				logger.Log.Info("Connection is READY")
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	logger.Log.Info(fmt.Sprintf("grpc connect :%v", client.GetState()))
	// 使用 gRPC 连接创建 UserService 客户端
	memberClient := memberpb.NewMemberServiceClient(client)

	// 初始化 UserHandler
	memberHandler := handlers.NewMemberHandler(memberClient)

	// 创建 Fiber 应用
	r := fiber.New()
	// 添加日志中间件
	file, err := os.OpenFile(fmt.Sprintf("%s/access.log", config.EnvConfig.APIGatewayLogPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	r.Use(fiber_log.New(fiber_log.Config{
		Output: file, // 将日志输出到文件
	}))

	// 注册路由
	router.RegisterRoutes(r, memberHandler)

	// 启动服务器
	if err := r.Listen(":" + cfg.Port); err != nil {
		// 执行清理操作
		cleanup()
		logger.Log.Fatal("Server failed to start", zap.Error(err))
	}
}

func cleanup() {
	// 释放资源，例如关闭数据库连接、清理文件等
	log.Println("Performing cleanup tasks...")
}
