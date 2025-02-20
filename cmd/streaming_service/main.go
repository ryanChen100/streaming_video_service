package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"streaming_video_service/internal/streaming/app"
	"streaming_video_service/internal/streaming/domain"
	"streaming_video_service/internal/streaming/repository"
	"streaming_video_service/pkg/config"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/logger"
	streaming_pb "streaming_video_service/pkg/proto/streaming"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger.Log = logger.Initialize(config.EnvConfig.Streaming, config.EnvConfig.StreamingLogPath)

	cfg := config.LoadConfig[config.Streaming](config.EnvConfig.Streaming, config.EnvConfig.StreamingYAMLPath)

	// 1. 連線 PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.PostgreSQL.Host, cfg.PostgreSQL.User, cfg.PostgreSQL.Password, cfg.PostgreSQL.Database, cfg.PostgreSQL.Port)
	db, err := database.NewPGConnection(database.Connection{
		ConnectStr: dsn,

		RetryCount:    cfg.PostgreSQL.RetryCount,
		RetryInterval: time.Duration(cfg.PostgreSQL.RetryInterval),
	})
	if err != nil {
		logger.Log.Fatal(
			"Unable to connect to postgreSQL database after retries",
			zap.String("address", fmt.Sprintf("[%s]", dsn)),
			zap.Error(err),
		)
	}

	// 自動遷移影片資料表
	videoRepo := repository.NewVideoRepo(db)
	if err := videoRepo.AutoMigrate(); err != nil {
		log.Fatalf("資料表遷移失敗: %v", err)
	}

	// 2. 初始化 MinIO 客戶端
	minioClient, err := database.NewMinIOConnection(database.MinIOConnection{
		Endpoint:   fmt.Sprintf("%s:%d", cfg.MinIO.Host, cfg.MinIO.Port),
		User:       cfg.MinIO.User,
		Password:   cfg.MinIO.Password,
		BucketName: cfg.MinIO.BucketName,
		UseSSL:     cfg.MinIO.UseSSL,

		RetryCount:    cfg.MinIO.RetryCount,
		RetryInterval: cfg.MinIO.RetryInterval,
	})
	if err != nil {
		logger.Log.Fatal(
			"Unable to connect to minio after retries",
			zap.String("address", fmt.Sprintf("[%s]", dsn)),
			zap.Error(err),
		)
	}

	rabbitURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.RabbitMQ.User, cfg.RabbitMQ.Password, cfg.RabbitMQ.IP, cfg.RabbitMQ.Port)
	conn, err := database.ConnectRabbitMQWithRetry(database.Connection{
		ConnectStr:    rabbitURL,
		RetryCount:    cfg.RabbitMQ.RetryCount,
		RetryInterval: time.Duration(cfg.RabbitMQ.RetryInterval),
	})
	if err != nil {
		log.Fatalf("RabbitMQ 連線失敗: %v", err)
	}
	defer conn.Close()

	rabbitChannel, err := database.GetRabbitMQChannelWithRetry(conn, cfg.RabbitMQ.RetryCount, cfg.RabbitMQ.RetryInterval)
	if err != nil {
		log.Fatalf("取得 RabbitMQ Channel 失敗: %v", err)
	}
	defer rabbitChannel.Close()

	//先初始化一個queue name = transcode
	if _, err := rabbitChannel.QueueDeclare(
		domain.QueueName, // queue name
		true,             // durable
		false,            // autoDelete
		false,            // exclusive
		false,            // noWait
		nil,              // arguments
	); err != nil {
		log.Fatalf("Queue Declare failed: %v", err)
	}

	// 建立 Kafka Writer 使用重試機制
	// kafkaWriter, err := database.NewKafkaWriterWithRetry(database.KafkaConnection{
	// 	Brokers:       cfg.KafKa.Brokers,
	// 	Topic:         cfg.KafKa.Topic,
	// 	RetryCount:    cfg.KafKa.RetryCount,
	// 	RetryInterval: cfg.KafKa.RetryInterval,
	// })
	// if err != nil {
	// 	log.Fatalf("Kafka Writer 建立失敗: %v", err)
	// }
	// defer kafkaWriter.Close()

	// 假設已初始化 rabbitChannel, minioClient, videoRepo
	consumer := app.NewConsumer(rabbitChannel, minioClient, videoRepo, domain.QueueName)
	// 使用 context 控制 Consumer 的生命週期
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 啟動 Consumer（通常以 goroutine 執行）
	go consumer.StartConsumer(ctx)

	// // 3. 建立 Fiber 應用
	// // 创建 Fiber 应用
	// r := fiber.New()
	// // 添加日志中间件
	// file, err := os.OpenFile(fmt.Sprintf("%s/access.log", config.EnvConfig.StreamingLogPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	// if err != nil {
	// 	log.Fatalf("Failed to open log file: %v", err)
	// }
	// defer file.Close()

	// r.Use(fiber_log.New(fiber_log.Config{
	// 	Output: file, // 将日志输出到文件
	// }))

	// // 4. 設定 API 路由
	// videoHandler := &handlers.VideoHandler{
	// 	MinioClient:   minioClient,
	// 	VideoRepo:     videoRepo,
	// 	RabbitChannel: rabbitChannel,
	// }
	// router.RegisterRoutes(r, videoHandler)

	// // 5. 啟動 API 服務（預設埠號）
	// // 启动服务器
	// if err := r.Listen(":" + cfg.Port); err != nil {
	// 	// 执行清理操作
	// 	cleanup()
	// 	logger.Log.Fatal("Server failed to start", zap.Error(err))
	// }

	usecase := app.NewStreamingUseCase(minioClient, videoRepo, rabbitChannel)

	lis, err := net.Listen("tcp", cfg.IP+":"+cfg.Port)
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("Failed to listen Port(%s): ", cfg.Port), zap.Error(err))
	}

	// 建立 gRPC 伺服器
	grpcServer := grpc.NewServer()

	streaming_pb.RegisterStreamingServiceServer(grpcServer, &app.StreamingGRPCServer{Usecase: usecase})
	logger.Log.Info(fmt.Sprintf("MemberService gRPC server listening on : %s", cfg.Port))

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}

func cleanup() {
	// 释放资源，例如关闭数据库连接、清理文件等
	log.Println("Performing cleanup tasks...")
}
