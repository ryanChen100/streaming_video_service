package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"streaming_video_service/internal/member/app"
	"streaming_video_service/internal/member/domain"
	"streaming_video_service/internal/member/repository"
	"streaming_video_service/pkg/config"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/logger"
	memberpb "streaming_video_service/pkg/proto/member"
)

func main() {
	logger.Log = logger.Initialize(config.EnvConfig.MemberService, config.EnvConfig.MemberServiceLogPath)

	cfg := config.LoadConfig[config.Member](config.EnvConfig.MemberService, config.EnvConfig.MemberServiceYAMLPath)

	sqlParams := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.PostgreSQL.User, cfg.PostgreSQL.Password, cfg.PostgreSQL.Host, cfg.PostgreSQL.Port, cfg.PostgreSQL.Database)
	pool, err := database.NewDatabaseConnection(database.Connection{
		ConnectStr:    sqlParams,
		RetryCount:    cfg.PostgreSQL.RetryCount,
		RetryInterval: time.Duration(cfg.PostgreSQL.RetryInterval),
	})

	if err != nil {
		logger.Log.Fatal(
			"Unable to connect to postgreSQL database after retries",
			zap.String("address", fmt.Sprintf("[%s]", sqlParams)),
			zap.Error(err),
		)
	}
	defer pool.Close()

	memberRepo := repository.NewMemberRepository(pool)
	masterName, sentinel := config.GetRedisSetting()
	redisRepo, err := database.NewRedisRepository[domain.MemberSession](masterName, sentinel, cfg.RedisMember.RedisDB)
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("connect redis err : %v", err))
	}
	usecase := app.NewMemberUseCase(memberRepo, cfg.SessionTTL*time.Minute, redisRepo)

	lis, err := net.Listen("tcp", cfg.IP+":"+cfg.Port)
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("Failed to listen Port(%s): ", cfg.Port), zap.Error(err))
	}

	// 建立 gRPC 伺服器
	grpcServer := grpc.NewServer()
	memberpb.RegisterMemberServiceServer(grpcServer, &app.MemberGRPCServer{Usecase: usecase})
	logger.Log.Info(fmt.Sprintf("MemberService gRPC server listening on : %s", cfg.Port))

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
