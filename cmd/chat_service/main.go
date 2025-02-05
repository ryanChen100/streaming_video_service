package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"streaming_video_service/internal/chat/app"
	"streaming_video_service/internal/chat/repository"
	"streaming_video_service/internal/chat/router"
	"streaming_video_service/pkg/config"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/logger"
	memberpb "streaming_video_service/pkg/proto/member"

	"github.com/gofiber/fiber/v2"
	fiber_log "github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

func main() {
	logger.Log = logger.Initialize(config.EnvConfig.ChatService, config.EnvConfig.ChatServiceLogPath)
	cfg := config.LoadConfig[config.Chat](config.EnvConfig.ChatService, config.EnvConfig.ChatServiceYAMLPath)

	// 2. 建立 Mongo 連線 (存訊息)
	ctx := context.Background()
	// uri := "mongodb://myuser:mypassword@localhost:27017"
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d", cfg.MongoSQL.User, cfg.MongoSQL.Password, cfg.MongoSQL.Host, cfg.MongoSQL.Port)
	mongo, err := database.NewMongoDB(ctx,
		database.Connection{
			ConnectStr:    uri,
			RetryCount:    cfg.MongoSQL.RetryCount,
			RetryInterval: time.Duration(cfg.MongoSQL.RetryInterval),
		},
		cfg.MongoSQL.Database)
	if err != nil {
		logger.Log.Fatal(
			"Unable to connect to mongoDB database after retries",
			zap.String("address", fmt.Sprintf("[%s]", uri)),
			zap.Error(err),
		)
	}
	defer mongo.Close(ctx)

	// 3. 建立 Redis 連線 (Pub/Sub)
	masterName, sentinel := config.GetRedisSetting()
	redisClient, err := database.NewRedisClient(masterName, sentinel, cfg.Redis.RedisDB)
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("connect redis err : %v", err))
	}

	// 4.建立 gRPC 连接
	client, err := grpc.Dial(cfg.MemberService.Name+":"+cfg.MemberService.Port, grpc.WithInsecure())
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("Failed to connect: %v", err))
	}
	defer client.Close()

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
	memberClient := memberpb.NewMemberServiceClient(client)

	// 5. 初始化 Repository
	roomRepo := repository.NewMongoChatRepository(mongo.Database)         // PostgreSQL
	inviteRepo := repository.NewMongoInvitationRepository(mongo.Database) // PostgreSQL
	msgRepo := repository.NewMongoChatMessageRepository(mongo.Database)   // MongoDB
	pub := repository.NewRedisPubSub(redisClient)

	// 6. 初始化 UseCases
	roomUC := app.NewRoomUseCase(inviteRepo, roomRepo)
	sendMessageUC := app.NewSendMessageUseCase(roomRepo, msgRepo, pub)
	// memberHub := app.NewEphemeralHub()

	// 7. 啟動 Fiber
	// 创建 Fiber 应用
	r := fiber.New()
	file, err := os.OpenFile(fmt.Sprintf("%s/access.log", config.EnvConfig.ChatServiceLogPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	r.Use(fiber_log.New(fiber_log.Config{
		Output: file, // 将日志输出到文件
	}))

	// 注册路由
	router.RegisterRoutes(r, app.NewChatWebsocketHandler(roomUC, sendMessageUC, &memberClient))

	// Listen
	port := ":" + cfg.Port
	log.Printf("Chat Service listening on %s", port)
	if err := r.Listen(port); err != nil {
		log.Fatalf("Failed to start Fiber: %v", err)
	}
}
