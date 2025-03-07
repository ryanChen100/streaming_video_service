package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"streaming_video_service/internal/streaming/domain"
	"streaming_video_service/pkg/logger"

	"github.com/minio/minio-go/v7"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMinIOClient 是 MinIOClient 的 Mock
type MockMinIOClient struct {
	mock.Mock
}

// UploadFile 模擬 MinIO 上傳行為
func (m *MockMinIOClient) UploadFile(ctx context.Context, objectName, filePath, contentType string) error {
	args := m.Called(ctx, objectName, filePath, contentType)
	return args.Error(0)
}

// DownloadFile 模擬 MinIO 下載行為
func (m *MockMinIOClient) DownloadFile(ctx context.Context, objectName, destPath string) error {
	args := m.Called(ctx, objectName, destPath)
	return args.Error(0)
}

// PresignGetURL 模擬 MinIO presign url
func (m *MockMinIOClient) PresignGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, objectName, expiry)
	return args.Get(0).(string), args.Error(1)
}

// GetObject 模擬 MinIO 取得object
func (m *MockMinIOClient) GetObject(ctx context.Context, objectName string, opts minio.GetObjectOptions) (io.Reader, error) {
	args := m.Called(ctx, objectName, opts)
	return args.Get(0).(io.Reader), args.Error(1)
}

// MockVideoRepo 是 VideoRepo 的 Mock
type MockVideoRepo struct {
	mock.Mock
}

func (m *MockVideoRepo) AutoMigrate() error {
	args := m.Called()
	return args.Error(0)
}

// Create 模擬創建影片記錄
func (m *MockVideoRepo) Create(video *domain.Video) error {
	args := m.Called(video)
	return args.Error(0)
}

func (m *MockVideoRepo) GetByID(id uint) (*domain.Video, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Video), args.Error(1)
}

// Update 模擬更新影片記錄
func (m *MockVideoRepo) Update(video *domain.Video) error {
	args := m.Called(video)
	return args.Error(0)
}

// Update 模擬更新影片記錄
func (m *MockVideoRepo) FindByStatus(status string) ([]domain.Video, error) {
	args := m.Called(status)
	return args.Get(0).([]domain.Video), args.Error(1)
}

// Update 模擬更新影片記錄
func (m *MockVideoRepo) SearchVideos(keyword string) ([]domain.Video, error) {
	args := m.Called(keyword)
	return args.Get(0).([]domain.Video), args.Error(1)
}

// Update 模擬更新影片記錄
func (m *MockVideoRepo) RecommendVideos(limit int) ([]domain.Video, error) {
	args := m.Called(limit)
	return args.Get(0).([]domain.Video), args.Error(1)
}

// MockRabbitChannel 是 RabbitMQ 的 Mock
type MockRabbitChannel struct {
	mock.Mock
}

// GetRabbit 模擬獲取 RabbitMQ Channel
func (m *MockRabbitChannel) GetRabbit() *amqp.Channel {
	args := m.Called()
	return args.Get(0).(*amqp.Channel)
}

func (m *MockRabbitChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	args := m.Called(exchange, key, mandatory, immediate, msg)
	return args.Error(0)
}

type mockFileSystemHelper struct {
	mock.Mock
}

func (m *mockFileSystemHelper) MkdirAll(path string, perm os.FileMode) error {
	args := m.Called(path, perm)
	return args.Error(0)
}

func (m *mockFileSystemHelper) Create(name string) (*os.File, error) {
	args := m.Called(name)
	return args.Get(0).(*os.File), args.Error(1)
}

// 測試 UploadVideo
func TestUploadVideo(t *testing.T) {
	mockMinIO := new(MockMinIOClient)
	mockRepo := new(MockVideoRepo)
	mockRabbit := new(MockRabbitChannel)
	logger.SetNewNop()
	usecase := NewStreamingUseCase(mockMinIO, mockRepo, mockRabbit)

	req := domain.UploadVideoReq{
		Title:       "Test Video",
		Description: "A test video",
		FileName:    "test.mp4",
		Type:        "mp4",
		File:        ioutil.NopCloser(bytes.NewReader([]byte("dummy video content"))),
	}

	// **情境 1: 成功上傳影片**
	t.Run("成功上傳影片", func(t *testing.T) {
		// Mock 影片創建，並手動設定 Video ID
		mockRepo.On("Create", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			video := args.Get(0).(*domain.Video)
			video.ID = 1 // 設定影片 ID
		}).Once()

		// Mock MinIO 上傳
		mockMinIO.On("UploadFile", mock.Anything, "original/1/test.mp4", mock.Anything, "video/mp4").
			Return(nil).Once()

		// Mock 影片記錄更新
		mockRepo.On("Update", mock.Anything).Return(nil).Once()

		// Mock RabbitMQ 發布轉碼工作
		mockRabbit.On("Publish",
			"",               // exchange
			domain.QueueName, // queue
			false,            // mandatory
			false,            // immediate
			mock.MatchedBy(func(p amqp.Publishing) bool {
				return p.ContentType == "application/json" && len(p.Body) > 0
			}),
		).Return(nil).Once()

		// 執行測試
		resp, err := usecase.UploadVideo(req)

		// 確保沒有錯誤
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "上傳成功，等待轉碼", resp.Message)
		assert.Equal(t, 1, resp.VideoID) // 影片 ID 必須正確

		// 驗證 Mock 方法是否被正確呼叫
		mockRepo.AssertExpectations(t)
		mockMinIO.AssertExpectations(t)
		mockRabbit.AssertExpectations(t)
	})

	//**情境 2: 建立暫存目錄失敗**
	t.Run("建立暫存目錄失敗", func(t *testing.T) {
		originalCreateDir := createDir
		defer func() { createDir = originalCreateDir }()

		createDir = func(path string) error {
			return errors.New("mkdir error")
		}

		resp, err := usecase.UploadVideo(req)
		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("fileName[%s] 建立暫存目錄失敗 : mkdir error", req.FileName), err.Error())
		assert.Nil(t, resp)
	})

	//**情境 3: 建立暫存檔案失敗**
	t.Run("建立暫存檔案失敗", func(t *testing.T) {
		originalCreateFile := createFile
		defer func() { createFile = originalCreateFile }()

		createFile = func(name string) (*os.File, error) {
			return nil, errors.New("create file error")
		}

		resp, err := usecase.UploadVideo(req)
		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("fileName[%s] 建立暫存檔案失敗 : create file error", req.FileName), err.Error())
		assert.Nil(t, resp)
	})

	//**情境 4: 儲存檔案失敗**
	t.Run("儲存檔案失敗", func(t *testing.T) {
		originalCopyFile := copyFile
		defer func() { copyFile = originalCopyFile }()

		copyFile = func(dst *os.File, src io.Reader) (written int64, err error) {
			return 0, errors.New("copy file error")
		}

		resp, err := usecase.UploadVideo(req)
		assert.Error(t, err)
		assert.Contains(t, fmt.Sprintf("fileName[%s] 儲存檔案失敗 : copy file error", req.FileName), err.Error())
		assert.Nil(t, resp)
	})

	//**情境 5: 資料庫建立影片失敗**
	t.Run("資料庫建立影片失敗", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything).Return(errors.New("db error")).Once()

		resp, err := usecase.UploadVideo(req)
		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("fileName[%s] 資料庫建立影片失敗 : db error", req.FileName), err.Error())
		assert.Nil(t, resp)
	})

	//**情境 6: 上傳 MinIO 失敗**
	t.Run("上傳 MinIO 失敗", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			video := args.Get(0).(*domain.Video)
			video.ID = 1
		}).Once()
		mockMinIO.On("UploadFile", mock.Anything, "original/1/test.mp4", mock.Anything, "video/mp4").Return(errors.New("minio error")).Once()

		resp, err := usecase.UploadVideo(req)
		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("fileName[%s] 上傳 MinIO 失敗 : minio error", req.FileName), err.Error())
		assert.Nil(t, resp)
	})

	//**情境 7: 更新影片記錄失敗**
	t.Run("更新影片記錄失敗", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			video := args.Get(0).(*domain.Video)
			video.ID = 1
		}).Once()

		mockMinIO.On("UploadFile", mock.Anything, "original/1/test.mp4", mock.Anything, "video/mp4").Return(nil).Once()
		mockRepo.On("Update", mock.Anything).Return(errors.New("update error")).Once()

		resp, err := usecase.UploadVideo(req)
		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("fileName[%s] 更新影片記錄失敗 : update error", req.FileName), err.Error())
		assert.Nil(t, resp)
	})

	//**情境 8: 發布轉碼訊息失敗**
	t.Run("發布轉碼工作訊息失敗", func(t *testing.T) {
		mockRepo.On("Create", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			video := args.Get(0).(*domain.Video)
			video.ID = 1
		})
		mockMinIO.On("UploadFile", mock.Anything, "original/1/test.mp4", mock.Anything, "video/mp4").Return(nil)
		mockRepo.On("Update", mock.Anything).Return(nil)
		mockRabbit.On("Publish",
			"",               // exchange
			domain.QueueName, // key (queue 名稱)
			false,            // mandatory
			false,            // immediate
			mock.MatchedBy(func(p amqp.Publishing) bool {
				return p.ContentType == "application/json" && len(p.Body) > 0
			}),
		).Return(errors.New("rabbit error")).Once()

		resp, err := usecase.UploadVideo(req)
		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("fileName[%s] 發送 RabbitMQ 訊息失敗 : rabbit error", req.FileName), err.Error())
		assert.Nil(t, resp)
	})
}

func TestGetVideo(t *testing.T) {
	mockRepo := new(MockVideoRepo) // Mock 影片儲存庫
	mockMinIO := new(MockMinIOClient)
	mockRabbit := new(MockRabbitChannel)

	logger.SetNewNop()
	usecase := NewStreamingUseCase(mockMinIO, mockRepo, mockRabbit)

	videoID := "1"
	parsedID, _ := strconv.Atoi(videoID) // 轉換為 int
	hlsURL := fmt.Sprintf("http://%s/video/hls/%d/index.m3u8", "127.0.0.1:8083", parsedID)

	// **情境 1: 成功取得影片**
	t.Run("成功取得影片", func(t *testing.T) {
		mockRepo.On("GetByID", uint(parsedID)).Return(&domain.Video{
			ID:     uint(parsedID),
			Title:  "Test Video",
			Status: string(domain.VideoReady),
		}, nil).Once()

		resp, err := usecase.GetVideo(videoID)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, parsedID, resp.VideoID)
		assert.Equal(t, "Test Video", resp.Title)
		assert.Equal(t, hlsURL, resp.HlsURL)

		mockRepo.AssertExpectations(t)
	})

	// **情境 2: 影片不存在**
	t.Run("影片不存在", func(t *testing.T) {
		mockRepo.On("GetByID", uint(parsedID)).Return(&domain.Video{ID: uint(parsedID)}, errors.New("影片不存在")).Once()

		resp, err := usecase.GetVideo(videoID)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("videoID[%s] 找不到影片: 影片不存在", videoID), err.Error())

		mockRepo.AssertExpectations(t)
	})

	// **情境 3: 影片未處理完成**
	t.Run("影片未處理完成", func(t *testing.T) {
		mockRepo.On("GetByID", uint(parsedID)).Return(&domain.Video{
			ID:     uint(parsedID),
			Title:  "Test Video",
			Status: string(domain.VideoProcessing),
		}, nil).Once()

		resp, err := usecase.GetVideo(videoID)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("videoID[%s] 影片尚未處理完成", videoID), err.Error())

		mockRepo.AssertExpectations(t)
	})
}

func TestSearch(t *testing.T) {
	mockRepo := new(MockVideoRepo) // Mock 影片儲存庫
	mockMinIO := new(MockMinIOClient)
	mockRabbit := new(MockRabbitChannel)

	logger.SetNewNop()
	usecase := NewStreamingUseCase(mockMinIO, mockRepo, mockRabbit)

	keyWord := "test"
	// **情境 1: 成功取得影片**
	t.Run("成功取得影片", func(t *testing.T) {
		mockRepo.On("SearchVideos", keyWord).Return([]domain.Video{
			{ID: 1,
				Title:       "title1",
				Description: "desc1",
				FileName:    "filename1", // 存於 MinIO 上的 object key
				Type:        "short",     // "short" 或 "long"
				Status:      string(domain.VideoReady),     // "uploaded", "processing", "ready"
				ViewCount:   100},
			{ID: 2,
				Title:       "title2",
				Description: "desc2",
				FileName:    "filename2", // 存於 MinIO 上的 object key
				Type:        "long",      // "short" 或 "long"
				Status:      string(domain.VideoReady),     // "uploaded", "processing", "ready"
				ViewCount:   200},
		}, nil).Once()

		resp, err := usecase.Search(keyWord)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockRepo.AssertExpectations(t)
	})

	// **情境 2: 找不到影片**
	t.Run("找不到影片", func(t *testing.T) {
		mockRepo.On("SearchVideos", keyWord).Return([]domain.Video{}, errors.New("找不到影片")).Once()
		resp, err := usecase.Search(keyWord)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("keyword[%s] search err : 找不到影片", keyWord), err.Error())

		mockRepo.AssertExpectations(t)
	})
}
func TestGetRecommendations(t *testing.T) {
	mockRepo := new(MockVideoRepo) // Mock 影片儲存庫
	mockMinIO := new(MockMinIOClient)
	mockRabbit := new(MockRabbitChannel)

	logger.SetNewNop()
	usecase := NewStreamingUseCase(mockMinIO, mockRepo, mockRabbit)

	limit := 10
	// **情境 1: 成功取得影片**
	t.Run("成功取得影片", func(t *testing.T) {
		mockRepo.On("RecommendVideos", limit).Return([]domain.Video{
			{ID: 1,
				Title:       "title1",
				Description: "desc1",
				FileName:    "filename1", // 存於 MinIO 上的 object key
				Type:        "short",     // "short" 或 "long"
				Status:      string(domain.VideoReady),     // "uploaded", "processing", "ready"
				ViewCount:   100},
			{ID: 2,
				Title:       "title2",
				Description: "desc2",
				FileName:    "filename2", // 存於 MinIO 上的 object key
				Type:        "long",      // "short" 或 "long"
				Status:      string(domain.VideoReady),     // "uploaded", "processing", "ready"
				ViewCount:   200},
		}, nil).Once()

		resp, err := usecase.GetRecommendations(limit)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockRepo.AssertExpectations(t)
	})

	// **情境 2: 找不到影片**
	t.Run("找不到影片", func(t *testing.T) {
		mockRepo.On("RecommendVideos", limit).Return([]domain.Video{}, errors.New("找不到影片")).Once()
		resp, err := usecase.GetRecommendations(limit)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("limit[%d] get recommendations err : 找不到影片", limit), err.Error())

		mockRepo.AssertExpectations(t)
	})
}

// 模擬 MinIO 的 GetObject 回傳值
func mockMinioObject() *minio.Object {
	// 建立假的 m3u8 檔案內容
	mockContent := []byte("#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1280000\nindex.m3u8")

	// 使用 io.Pipe 來模擬 MinIO 物件的讀取行為
	reader := io.NopCloser(bytes.NewReader(mockContent))

	// 創建 `minio.Object`，這裡使用 `reader` 來模擬它的行為
	obj := &minio.Object{}
	reflect.ValueOf(obj).Elem().FieldByName("Reader").Set(reflect.ValueOf(reader))

	return obj
}

func TestGetIndexM3U8(t *testing.T) {
	mockMinIO := new(MockMinIOClient)
	mockRepo := new(MockVideoRepo)
	mockRabbit := new(MockRabbitChannel)

	logger.SetNewNop()
	usecase := NewStreamingUseCase(mockMinIO, mockRepo, mockRabbit)
	ctx := context.Background()
	videoID := "1"
	objectKey := "processed/" + videoID + "/index.m3u8"

	//  正確的 Mock MinIO 回傳
	mockContent := []byte("#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1280000\nindex.m3u8")
	mockReader := io.NopCloser(bytes.NewReader(mockContent))

	// **情境 1: 成功取得播放清單**
	t.Run("成功取得播放清單", func(t *testing.T) {
		//  建立符合 `*minio.Object` 的 Mock
		mockMinIO.On("GetObject", ctx, objectKey, mock.Anything).
			Return(mockReader, nil).Once()

		//  Mock 讀取內容
		readFile = func(r io.Reader) ([]byte, error) {
			return mockContent, nil
		}

		resp, err := usecase.GetIndexM3U8(ctx, videoID)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mockContent, resp) // 確保內容一致

		mockMinIO.AssertExpectations(t)
	})
	// **情境 2: 無法取得m3u8檔案**
	t.Run("無法取得 m3u8 檔案", func(t *testing.T) {
		mockMinIO.On("GetObject", ctx, objectKey, mock.Anything).
			Return(bytes.NewReader(nil), errors.New("minio error")).Once()

		resp, err := usecase.GetIndexM3U8(ctx, videoID)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("videoID[%s] 無法取得 m3u8 檔案 : minio error", videoID), err.Error())

		mockMinIO.AssertExpectations(t)
	})

	// **情境 3: 讀取m3u8檔案失敗**
	t.Run("讀取m3u檔案失敗", func(t *testing.T) {
		mockMinIO.On("GetObject", ctx, objectKey, mock.Anything).
			Return(bytes.NewReader(nil), nil).Once()

		// Mock `readFile` 讓它回傳錯誤
		readFile = func(_ io.Reader) ([]byte, error) {
			return nil, errors.New("read error")
		}

		resp, err := usecase.GetIndexM3U8(ctx, videoID)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("videoID[%s] 讀取 m3u8 檔案失敗 : read error", videoID), err.Error())

		mockMinIO.AssertExpectations(t)
	})
}

func TestGetHlsSegment(t *testing.T) {
	mockMinIO := new(MockMinIOClient)
	mockRepo := new(MockVideoRepo)
	mockRabbit := new(MockRabbitChannel)

	logger.SetNewNop()
	usecase := NewStreamingUseCase(mockMinIO, mockRepo, mockRabbit)
	ctx := context.Background()
	videoID := "1"
	segment := "segment"
	objectKey := "processed/" + videoID + "/" + segment

	//  正確的 Mock MinIO 回傳
	mockContent := []byte("#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1280000\nindex.m3u8")
	mockReader := io.NopCloser(bytes.NewReader(mockContent))

	// **情境 1: 成功取得 TS 分段檔案**
	t.Run("成功取得 TS 分段檔案", func(t *testing.T) {
		//  建立符合 `*minio.Object` 的 Mock
		mockMinIO.On("GetObject", ctx, objectKey, mock.Anything).
			Return(mockReader, nil).Once()

		//  Mock 讀取內容
		readFile = func(r io.Reader) ([]byte, error) {
			return mockContent, nil
		}

		resp, err := usecase.GetHlsSegment(ctx, videoID, segment)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mockContent, resp) // 確保內容一致

		mockMinIO.AssertExpectations(t)
	})
	// **情境 2: 找不到 segment TS 分段檔案失敗**
	t.Run("找不到 segment TS 分段檔案失敗", func(t *testing.T) {
		mockMinIO.On("GetObject", ctx, objectKey, mock.Anything).
			Return(bytes.NewReader(nil), errors.New("minio error")).Once()

		resp, err := usecase.GetHlsSegment(ctx, videoID, segment)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("videoID_segment[%s_%s] 無法取得 segment 檔案 : minio error", videoID, segment), err.Error())

		mockMinIO.AssertExpectations(t)
	})

	// **情境 3: 讀取 segment TS 分段檔案失敗**
	t.Run("讀取 segment TS 分段檔案失敗", func(t *testing.T) {
		mockMinIO.On("GetObject", ctx, objectKey, mock.Anything).
			Return(bytes.NewReader(nil), nil).Once()

		// Mock `readFile` 讓它回傳錯誤
		readFile = func(_ io.Reader) ([]byte, error) {
			return nil, errors.New("read error")
		}

		resp, err := usecase.GetHlsSegment(ctx, videoID, segment)

		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("videoID_segment[%s_%s] 讀取 segment 檔案失敗 : read error", videoID, segment), err.Error())

		mockMinIO.AssertExpectations(t)
	})
}
