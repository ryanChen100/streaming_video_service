package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"

	"os"
	"os/exec"
	"testing"
	"time"

	"streaming_video_service/internal/streaming/domain"
	"streaming_video_service/internal/streaming/repository"
	"streaming_video_service/pkg/config"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/logger"

	streaming_pb "streaming_video_service/pkg/proto/streaming"
	testtool "streaming_video_service/pkg/test_tool"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
)

// **測試用的容器**
var postgresContainer testcontainers.Container
var minioContainer testcontainers.Container
var rabbitmqContainer testcontainers.Container

// **Handler**
var streamingHandler *StreamingGRPCServer
var minioClient database.MinIOClientRepo

var (
	minioUser     = "minioadmin"
	minioPassword = "minioadmin"
	minioBucket   = "video-bucket"

	rabbitUser     = "rabbitadmin"
	rabbitPassword = "rabbitadmin"

	integrationFilePath = "./tmp"
)

// **TestMain - 初始化測試環境**
func TestMain(m *testing.M) {
	ctx := context.Background()
	var err error
	logger.SetNewNop()

	// **啟動 PostgreSQL**
	postgresContainer, postgresHost, postgresPort, err := testtool.SetupContainer(ctx, testcontainers.ContainerRequest{
		Image: "postgres:latest",
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "streamingdb",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	})
	if err != nil {
		log.Fatalf("❌ Failed to start PostgreSQL: %v", err)
	}
	fmt.Printf("✅ PostgreSQL running at %s:%s\n", postgresHost, postgresPort)

	// **啟動 MinIO**
	minioContainer, minioHost, minioPort, err := testtool.SetupContainer(ctx, testcontainers.ContainerRequest{
		Image: "minio/minio:latest",
		Cmd:   []string{"server", "/data"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     minioUser,
			"MINIO_ROOT_PASSWORD": minioPassword,
		},
		ExposedPorts: []string{"9000/tcp"},
		WaitingFor:   wait.ForListeningPort("9000/tcp"),
	})
	if err != nil {
		log.Fatalf("❌ Failed to start MinIO: %v", err)
	}
	fmt.Printf("✅ MinIO running at %s:%s\n", minioHost, minioPort)

	// **啟動 RabbitMQ**
	rabbitmqContainer, rabbitmqHost, rabbitmqPort, err := testtool.SetupContainer(ctx, testcontainers.ContainerRequest{
		Image: "rabbitmq:3-management",
		Env: map[string]string{
			"RABBITMQ_DEFAULT_USER": rabbitUser,
			"RABBITMQ_DEFAULT_PASS": rabbitPassword,
		},
		ExposedPorts: []string{"5672/tcp", "15672/tcp"},
		WaitingFor:   wait.ForListeningPort("5672/tcp"),
	})
	if err != nil {
		log.Fatalf("❌ Failed to start RabbitMQ: %v", err)
	}
	fmt.Printf("✅ RabbitMQ running at %s:%s\n", rabbitmqHost, rabbitmqPort)

	// **設定環境變數**
	os.Setenv("DATABASE_URL", fmt.Sprintf("postgres://test:test@%s:%s/streamingdb?sslmode=disable", postgresHost, postgresPort))
	os.Setenv("MINIO_URL", fmt.Sprintf("%s:%s", minioHost, minioPort))
	os.Setenv("RABBITMQ_URL", fmt.Sprintf("amqp://%s:%s@%s:%s/", rabbitUser, rabbitPassword, rabbitmqHost, rabbitmqPort))

	fmt.Printf("🔹 DATABASE_URL=%s\n", os.Getenv("DATABASE_URL"))
	fmt.Printf("🔹 MINIO_URL=%s\n", os.Getenv("MINIO_URL"))
	fmt.Printf("🔹 RABBITMQ_URL=%s\n", os.Getenv("RABBITMQ_URL"))

	// **執行 Migrations**
	migrationsPath, err := config.GetPath("Makefile/migrations", 5)
	if err != nil {
		log.Fatalf("get migrations path Error : %v", err)
	}
	fmt.Printf("🔹 migrations path = %s\n", migrationsPath)
	cmd := exec.Command("migrate", "-database", os.Getenv("DATABASE_URL"), "-path", migrationsPath, "up")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// **等待 Redis 確保已經準備好**
	time.Sleep(5 * time.Second)

	// **初始化 PostgreSQL**
	db, err := database.NewPGConnection(database.Connection{
		ConnectStr:    os.Getenv("DATABASE_URL"),
		RetryCount:    5,
		RetryInterval: 5,
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}

	// **初始化 MinIO**
	minioClient, err = database.NewMinIOConnection(database.MinIOConnection{
		Endpoint:   os.Getenv("MINIO_URL"),
		User:       minioUser,
		Password:   minioPassword,
		BucketName: minioBucket,
		UseSSL:     false,

		RetryCount:    5,
		RetryInterval: 5,
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to MinIO: %v", err)
	}

	// **初始化 RabbitMQ**
	mqClient, err := database.ConnectRabbitMQWithRetry(database.Connection{
		ConnectStr:    os.Getenv("RABBITMQ_URL"),
		RetryCount:    5,
		RetryInterval: 5,
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to RabbitMQ: %v", err)
	}
	defer mqClient.Close()

	rabbitChannel, err := database.GetRabbitMQChannelWithRetry(mqClient, 5, 5)
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

	videoRepo := repository.NewVideoRepo(db)
	if err := videoRepo.AutoMigrate(); err != nil {
		log.Fatalf("資料表遷移失敗: %v", err)
	}

	rabbitRepo := database.NewRabbitRepository(rabbitChannel)

	usecase := NewStreamingUseCase(minioClient, videoRepo, rabbitRepo)

	// **初始化 Handler**
	streamingHandler = new(StreamingGRPCServer)
	streamingHandler.Usecase = usecase

	fmt.Println("✅ All services connected successfully!")

	// **執行測試**
	code := m.Run()

	// **清理測試環境**
	_ = postgresContainer.Terminate(ctx)
	_ = minioContainer.Terminate(ctx)
	_ = rabbitmqContainer.Terminate(ctx)

	os.Exit(code)
}

type MockUploadVideoServer struct {
	grpc.ServerStream
	recvQueue []*streaming_pb.UploadVideoReq
	response  *streaming_pb.UploadVideoRes
}

func NewMockUploadVideoServer() *MockUploadVideoServer {
	return &MockUploadVideoServer{recvQueue: []*streaming_pb.UploadVideoReq{}}
}

func (m *MockUploadVideoServer) Recv() (*streaming_pb.UploadVideoReq, error) {
	if len(m.recvQueue) == 0 {
		return nil, io.EOF
	}
	req := m.recvQueue[0]
	m.recvQueue = m.recvQueue[1:]
	return req, nil
}

func (m *MockUploadVideoServer) RecvData(req *streaming_pb.UploadVideoReq) {
	m.recvQueue = append(m.recvQueue, req)
}

func (m *MockUploadVideoServer) SendAndClose(res *streaming_pb.UploadVideoRes) error {
	m.response = res
	return nil
}

func (m *MockUploadVideoServer) GetResponse() *streaming_pb.UploadVideoRes {
	if m.response == nil {
		return &streaming_pb.UploadVideoRes{Success: false, Message: "no response"}
	}
	return m.response
}

type MockUploadVideoServerWithWriteError struct {
	MockUploadVideoServer
}

func NewMockUploadVideoServerWithWriteError() *MockUploadVideoServerWithWriteError {
	return &MockUploadVideoServerWithWriteError{}
}

func (m *MockUploadVideoServerWithWriteError) Recv() (*streaming_pb.UploadVideoReq, error) {
	req, err := m.MockUploadVideoServer.Recv()
	if req != nil && req.GetChunk() != nil {
		// **模擬 fileBuffer.Write() 發生錯誤**
		req.GetChunk().Content = nil
	}
	return req, err
}

// ✅1️⃣測試 UploadVideo
func TestIntegrationUploadVideo(t *testing.T) {
	// **確認 handler 初始化**
	assert.NotNil(t, streamingHandler, "❌ streamingHandler 未初始化")

	t.Run("成功上傳影片", func(t *testing.T) {
		mockStream := NewMockUploadVideoServer()

		// 傳送 Metadata
		mockStream.RecvData(&streaming_pb.UploadVideoReq{
			Data: &streaming_pb.UploadVideoReq_Metadata{
				Metadata: &streaming_pb.VideoMetadata{
					Title:       "Test Video",
					Description: "Integration Test",
					Type:        "mp4",
					FileName:    "test_video.mp4",
				},
			},
		})

		// 傳送影片 Chunk
		mockStream.RecvData(&streaming_pb.UploadVideoReq{
			Data: &streaming_pb.UploadVideoReq_Chunk{
				Chunk: &streaming_pb.VideoChunk{
					Content: []byte("dummy_video_data"),
				},
			},
		})

		// 執行 `UploadVideo`
		err := streamingHandler.UploadVideo(mockStream)
		assert.NoError(t, err, "❌ UploadVideo 執行失敗")

		// 確保成功回應
		resp := mockStream.GetResponse()
		assert.True(t, resp.Success)
		fmt.Println("✅ 成功上傳影片:", resp.Message)
	})

	t.Run("缺少影片 Metadata", func(t *testing.T) {
		mockStream := NewMockUploadVideoServer()

		// 直接傳送影片 Chunk，**沒有 Metadata**
		mockStream.RecvData(&streaming_pb.UploadVideoReq{
			Data: &streaming_pb.UploadVideoReq_Chunk{
				Chunk: &streaming_pb.VideoChunk{
					Content: []byte("dummy_video_data"),
				},
			},
		})

		// **執行 UploadVideo**
		err := streamingHandler.UploadVideo(mockStream)
		assert.NoError(t, err, "❌ UploadVideo 應該回傳錯誤，但沒有發生錯誤")

		// **確保 resp 不為 nil**
		resp := mockStream.GetResponse()
		assert.NotNil(t, resp, "❌ Response 應該不為 nil")
		assert.False(t, resp.Success)
		assert.Equal(t, "缺少影片元資料", resp.Message)

		fmt.Println("✅ 缺少 Metadata 測試通過")
	})

	t.Run("影片 Chunk 寫入錯誤", func(t *testing.T) {
		mockStream := NewMockUploadVideoServer()

		// 傳送 Metadata
		mockStream.RecvData(&streaming_pb.UploadVideoReq{
			Data: &streaming_pb.UploadVideoReq_Metadata{
				Metadata: &streaming_pb.VideoMetadata{
					Title:       "Test Video",
					Description: "Integration Test",
					Type:        "mp4",
					FileName:    "test_video.mp4",
				},
			},
		})

		// 執行 `UploadVideo`
		err := streamingHandler.UploadVideo(mockStream)
		assert.NoError(t, err, "❌ UploadVideo 不應該回傳 gRPC error，而應該回應寫入錯誤")

		// **確保回應為錯誤**
		resp := mockStream.GetResponse()
		assert.NotNil(t, resp, "❌ Response 不應為 nil")
		assert.False(t, resp.Success)
		assert.Equal(t, "缺少寫入檔案區塊", resp.Message)

		fmt.Println("✅ 影片 Chunk 寫入錯誤測試通過")
	})
}

// ✅2️⃣TestIntegrationGetVideo
func TestIntegrationGetVideo(t *testing.T) {
	ctx := context.Background()

	// 確保 `streamingHandler` 已經初始化
	assert.NotNil(t, streamingHandler, "❌ streamingHandler 未初始化")

	t.Run("取得有效影片資訊", func(t *testing.T) {
		// **準備請求**
		req := &streaming_pb.GetVideoReq{VideoId: "1"} // 這應該對應到 `INSERT` 的第一筆數據

		// **執行 `GetVideo`**
		resp, err := streamingHandler.GetVideo(ctx, req)

		// **確認回應**
		assert.NoError(t, err, "❌ GetVideo 應該成功但發生錯誤")
		assert.NotNil(t, resp, "❌ GetVideo 回應不應為 nil")
		assert.True(t, resp.Success, "❌ GetVideo 回應應該為成功")
		assert.Equal(t, "Sample Video 1", resp.Title, "❌ 影片標題錯誤")
		assert.NotEmpty(t, resp.HlsUrl, "❌ HLS URL 不應為空")

		fmt.Println("✅ 取得影片資訊成功:", resp.Title, resp.HlsUrl)
	})

	t.Run("查詢不存在的影片", func(t *testing.T) {
		videoID := "999"
		// **準備請求**
		req := &streaming_pb.GetVideoReq{VideoId: videoID} // 假設 999 是不存在的影片 ID

		// **執行 `GetVideo`**
		resp, err := streamingHandler.GetVideo(ctx, req)

		// **確認回應**
		assert.Error(t, err, "❌ 應該回傳錯誤")
		assert.NotNil(t, resp, "❌ GetVideo 回應不應為 nil")
		assert.False(t, resp.Success, "❌ GetVideo 回應應該為失敗")
		assert.Equal(t, err.Error(), fmt.Sprintf("videoID[%s] 找不到影片: record not found", videoID))
		fmt.Println("✅ 查詢不存在的影片測試通過")
	})

	t.Run("查詢未準備好", func(t *testing.T) {
		videoID := "12"
		// **準備請求**
		req := &streaming_pb.GetVideoReq{VideoId: videoID} // 假設 999 是不存在的影片 ID

		// **執行 `GetVideo`**
		resp, err := streamingHandler.GetVideo(ctx, req)

		// **確認回應**
		assert.Error(t, err, "❌ 應該回傳錯誤")
		assert.NotNil(t, resp, "❌ GetVideo 回應不應為 nil")
		assert.False(t, resp.Success, "❌ GetVideo 回應應該為失敗")
		assert.Equal(t, err.Error(), fmt.Sprintf("videoID[%s] 影片尚未處理完成", videoID))
		fmt.Println("✅ 查詢未準備好的影片測試通過")
	})
}

// ✅3️⃣TestIntegrationSearch
func TestIntegrationSearch(t *testing.T) {
	ctx := context.Background()

	// 確保 `streamingHandler` 已經初始化
	assert.NotNil(t, streamingHandler, "❌ streamingHandler 未初始化")

	t.Run("成功搜尋關鍵字", func(t *testing.T) {
		// **準備請求**
		req := &streaming_pb.SearchReq{KeyWord: "Sample"}

		// **執行 `Search`**
		resp, err := streamingHandler.Search(ctx, req)

		// **確認回應**
		assert.NoError(t, err, "❌ Search 應該成功但發生錯誤")
		assert.NotNil(t, resp, "❌ Search 回應不應為 nil")
		assert.True(t, resp.Success, "❌ Search 回應應該為成功")
		assert.Greater(t, len(resp.Video), 0, "❌ 搜尋結果應該至少有 1 部影片")
		// **檢查搜尋結果是否包含 "Sample"**
		for _, video := range resp.Video {
			assert.Contains(t, video.Title, "Sample", "❌ 搜尋結果標題應包含關鍵字")
		}

		fmt.Println("✅ 搜尋關鍵字 'Sample' 測試通過，共找到", len(resp.Video), "部影片")
	})

	t.Run("搜尋不存在的關鍵字", func(t *testing.T) {
		// **準備請求**
		req := &streaming_pb.SearchReq{KeyWord: "NonExistentKeyword"}

		// **執行 `Search`**
		resp, err := streamingHandler.Search(ctx, req)

		// **確認回應**
		assert.NoError(t, err, "❌ Search 應該成功但發生錯誤")
		assert.NotNil(t, resp, "❌ Search 回應不應為 nil")
		assert.True(t, resp.Success, "❌ Search 回應應該為成功")
		assert.Equal(t, 0, len(resp.Video), "❌ 搜尋結果應該為 0 筆")

		fmt.Println("✅ 搜尋關鍵字 'NonExistentKeyword' 測試通過，無結果")
	})

	t.Run("空白搜尋關鍵字", func(t *testing.T) {
		// **準備請求**
		req := &streaming_pb.SearchReq{KeyWord: ""}

		// **執行 `Search`**
		resp, err := streamingHandler.Search(ctx, req)

		// **確認回應**
		assert.NoError(t, err, "❌ Search 應該成功但發生錯誤")
		assert.NotNil(t, resp, "❌ Search 回應不應為 nil")
		assert.True(t, resp.Success, "❌ Search 回應應該為成功")
		assert.Greater(t, len(resp.Video), 0, "❌ 空白搜尋應該回傳所有影片")

		fmt.Println("✅ 空白搜尋測試通過，共找到", len(resp.Video), "部影片")
	})
}

// TODO ❌TestIntegrationGetRecommendations
func TestIntegrationGetRecommendations(t *testing.T) {
	ctx := context.Background()

	// 確保 `streamingHandler` 已經初始化
	assert.NotNil(t, streamingHandler, "❌ streamingHandler 未初始化")

	t.Run("獲取熱門影片推薦（限制 5 部）", func(t *testing.T) {
		// **準備請求**
		req := &streaming_pb.GetRecommendationsReq{Limit: 5}

		// **執行 `GetRecommendations`**
		resp, err := streamingHandler.GetRecommendations(ctx, req)

		// **確認回應**
		assert.NoError(t, err, "❌ GetRecommendations 應該成功但發生錯誤")
		assert.NotNil(t, resp, "❌ GetRecommendations 回應不應為 nil")
		assert.True(t, resp.Success, "❌ GetRecommendations 回應應該為成功")
		assert.GreaterOrEqual(t, len(resp.Video), 1, "❌ 應至少有 1 部推薦影片")
		assert.LessOrEqual(t, len(resp.Video), 5, "❌ 回應數量應該不超過 5 部")

		// **檢查影片觀看次數排序**
		for i := 1; i < len(resp.Video); i++ {
			assert.GreaterOrEqual(t, resp.Video[i-1].ViewCCount, resp.Video[i].ViewCCount, "❌ 推薦影片應依照觀看次數排序")
		}

		fmt.Println("✅ 熱門影片推薦測試通過（限制 5 部），共找到", len(resp.Video), "部影片")
	})

	//! ❌暫時未施作
	// t.Run("獲取熱門影片推薦（無限制）", func(t *testing.T) {
	// 	// **準備請求**
	// 	req := &streaming_pb.GetRecommendationsReq{Limit: 0} // 0 代表不限制數量

	// 	// **執行 `GetRecommendations`**
	// 	resp, err := streamingHandler.GetRecommendations(ctx, req)

	// 	// **確認回應**
	// 	assert.NoError(t, err, "❌ GetRecommendations 應該成功但發生錯誤")
	// 	assert.NotNil(t, resp, "❌ GetRecommendations 回應不應為 nil")
	// 	assert.True(t, resp.Success, "❌ GetRecommendations 回應應該為成功")
	// 	assert.GreaterOrEqual(t, len(resp.Video), 1, "❌ 應至少有 1 部推薦影片")

	// 	fmt.Println("✅ 熱門影片推薦測試通過（無限制），共找到", len(resp.Video), "部影片")
	// })

	// t.Run("熱門影片推薦但資料庫沒有影片", func(t *testing.T) {
	// 	// **清空影片表**
	// 	err := database.ClearTable("videos") // 假設有這個工具函式
	// 	assert.NoError(t, err, "❌ 清空影片表失敗")

	// 	// **準備請求**
	// 	req := &streaming_pb.GetRecommendationsReq{Limit: 5}

	// 	// **執行 `GetRecommendations`**
	// 	resp, err := streamingHandler.GetRecommendations(ctx, req)

	// 	// **確認回應**
	// 	assert.NoError(t, err, "❌ GetRecommendations 應該成功但發生錯誤")
	// 	assert.NotNil(t, resp, "❌ GetRecommendations 回應不應為 nil")
	// 	assert.True(t, resp.Success, "❌ GetRecommendations 回應應該為成功")
	// 	assert.Equal(t, 0, len(resp.Video), "❌ 應該沒有推薦影片")

	// 	fmt.Println("✅ 當資料庫沒有影片時，熱門影片推薦測試通過")
	// })
}

func uploadTestVideo(ctx context.Context, title, description, fileName string) (string, error) {
	mockStream := NewMockUploadVideoServer()

	// **傳送 Metadata**
	mockStream.RecvData(&streaming_pb.UploadVideoReq{
		Data: &streaming_pb.UploadVideoReq_Metadata{
			Metadata: &streaming_pb.VideoMetadata{
				Title:       title,
				Description: description,
				Type:        "mp4",
				FileName:    fileName,
			},
		},
	})

	// **傳送影片 Chunk**
	mockStream.RecvData(&streaming_pb.UploadVideoReq{
		Data: &streaming_pb.UploadVideoReq_Chunk{
			Chunk: &streaming_pb.VideoChunk{
				Content: []byte("dummy_video_data"),
			},
		},
	})

	// **執行 UploadVideo**
	err := streamingHandler.UploadVideo(mockStream)
	if err != nil {
		return "", fmt.Errorf("❌ UploadVideo 失敗: %v", err)
	}

	// **確保成功回應**
	resp := mockStream.GetResponse()
	if !resp.Success {
		return "", fmt.Errorf("❌ UploadVideo 回應失敗: %s", resp.Message)
	}

	fmt.Println("✅ 測試影片上傳成功，VideoID:", resp.VideoId)

	// **模擬 MinIO 內 m3u8 和 TS 段文件**
	videoID := strconv.Itoa(int(resp.VideoId))

	// **創建 m3u8 檔案**
	objectKeyM3U8 := fmt.Sprintf("processed/%s/index.m3u8", videoID)
	m3u8FilePath := fmt.Sprintf("%s/%s.m3u8", integrationFilePath, videoID)
	mockM3U8Content := "#EXTM3U8\n#EXT-X-STREAM-INF:BANDWIDTH=1280000\nvideo_720p.m3u8"

	err = os.WriteFile(m3u8FilePath, []byte(mockM3U8Content), 0644)
	if err != nil {
		return "", fmt.Errorf("❌ 無法寫入 m3u8 測試檔案: %v", err)
	}

	// **上傳 m3u8**
	err = minioClient.UploadFile(ctx, objectKeyM3U8, m3u8FilePath, "application/vnd.apple.mpegurl")
	if err != nil {
		return "", fmt.Errorf("❌ 無法上傳 m3u8 到 MinIO: %v", err)
	}

	// **創建 TS 段影片檔案**
	objectKeyTS := fmt.Sprintf("processed/%s/segment_00001.ts", videoID)
	tsFilePath := fmt.Sprintf("%s/%s.ts", integrationFilePath, videoID)
	mockTSContent := []byte("MOCK_TS_DATA")

	err = os.WriteFile(tsFilePath, mockTSContent, 0644)
	if err != nil {
		return "", fmt.Errorf("❌ 無法寫入 TS 測試檔案: %v", err)
	}

	// **上傳 TS 段影片**
	err = minioClient.UploadFile(ctx, objectKeyTS, tsFilePath, "video/MP2T")
	if err != nil {
		return "", fmt.Errorf("❌ 無法上傳 TS 段影片到 MinIO: %v", err)
	}

	fmt.Println("✅ 測試 m3u8 & TS 文件上傳成功")
	return videoID, nil
}
func cleanupTempDir(path string) error {
	err := os.RemoveAll(path) // 刪除整個資料夾及其內容
	if err != nil {
		return fmt.Errorf("刪除暫存目錄失敗: %v", err)
	}
	return nil
}
func TestIntegrationGetIndexM3U8(t *testing.T) {
	ctx := context.Background()

	// **確保 `streamingHandler` 已初始化**
	assert.NotNil(t, streamingHandler, "❌ streamingHandler 未初始化")

	t.Run("成功獲取 m3u8 播放清單", func(t *testing.T) {
		// **透過 UploadVideo 產生影片**
		videoID, err := uploadTestVideo(ctx, "Test Video", "Integration Test", "test_video.mp4")
		assert.NoError(t, err, "❌ 上傳測試影片失敗")

		// **執行 `GetIndexM3U8`**
		resp, err := streamingHandler.Usecase.GetIndexM3U8(ctx, videoID)

		// **確認回應**
		assert.NoError(t, err, "❌ GetIndexM3U8 應該成功但發生錯誤")
		assert.NotNil(t, resp, "❌ GetIndexM3U8 回應不應為 nil")
		assert.Contains(t, string(resp), "#EXTM3U8", "❌ m3u8 應包含 EXTINF")

		fmt.Println("✅ 成功獲取 m3u8 播放清單")
	})

	t.Run("m3u8 不存在", func(t *testing.T) {
		videoID := "999"

		// **執行 `GetIndexM3U8`**
		resp, err := streamingHandler.Usecase.GetIndexM3U8(ctx, videoID)

		// **確認錯誤**
		assert.Error(t, err, "❌ m3u8 不存在時應該回傳錯誤")
		assert.Nil(t, resp, "❌ m3u8 不存在時回應應為 nil")

		fmt.Println("✅ m3u8 不存在時，錯誤處理測試通過")
	})

	if err := cleanupTempDir("./tmp"); err != nil {
		fmt.Println("❌ 清理 `tmp` 目錄失敗:", err)
	} else {
		fmt.Println("✅ 成功清理 `tmp` 目錄")
	}
}

func TestIntegrationGetHlsSegment(t *testing.T) {
	ctx := context.Background()

	// **確保 `streamingHandler` 已初始化**
	assert.NotNil(t, streamingHandler, "❌ streamingHandler 未初始化")

	t.Run("成功獲取 TS 段影片", func(t *testing.T) {
		// **透過 UploadVideo 產生影片**
		videoID, err := uploadTestVideo(ctx, "Test Video", "Integration Test", "test_video.mp4")
		assert.NoError(t, err, "❌ 上傳測試影片失敗")

		segment := "segment_00001.ts"

		// **執行 `GetHlsSegment`**
		resp, err := streamingHandler.Usecase.GetHlsSegment(ctx, videoID, segment)

		// **確認回應**
		assert.NoError(t, err, "❌ GetHlsSegment 應該成功但發生錯誤")
		assert.NotNil(t, resp, "❌ GetHlsSegment 回應不應為 nil")
		assert.Equal(t, []byte("MOCK_TS_DATA"), resp, "❌ TS 段影片內容不符合預期")

		fmt.Println("✅ 成功獲取 TS 段影片")
	})

	t.Run("TS 段影片不存在", func(t *testing.T) {
		videoID := "999"
		segment := "missing_segment.ts"

		// **執行 `GetHlsSegment`**
		resp, err := streamingHandler.Usecase.GetHlsSegment(ctx, videoID, segment)

		// **確認錯誤**
		assert.Error(t, err, "❌ TS 段影片不存在時應該回傳錯誤")
		assert.Nil(t, resp, "❌ TS 段影片不存在時回應應為 nil")

		fmt.Println("✅ TS 段影片不存在時，錯誤處理測試通過")
	})

	if err := cleanupTempDir("./tmp"); err != nil {
		fmt.Println("❌ 清理 `tmp` 目錄失敗:", err)
	} else {
		fmt.Println("✅ 成功清理 `tmp` 目錄")
	}
}
