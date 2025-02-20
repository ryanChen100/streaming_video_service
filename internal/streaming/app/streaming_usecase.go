package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"streaming_video_service/internal/streaming/domain"
	"streaming_video_service/internal/streaming/repository"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/logger"

	"github.com/minio/minio-go/v7"
	"github.com/streadway/amqp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamingUseCase 這裡封裝了對外提供的應用服務
type StreamingUseCase interface {
	UploadVideo(up domain.UploadVideoReq) (*domain.UploadVideoRes, error)
	GetVideo(videoID string) (*domain.GetVideoRes, error)
	Search(keyWord string) ([]domain.Video, error)
	GetRecommendations(limit int) ([]domain.Video, error)
	GetIndexM3U8(ctx context.Context, videoID string) ([]byte, error)
	GetHlsSegment(ctx context.Context, videoID, segment string) ([]byte, error)
}

type streamingUseCase struct {
	MinioClient   *database.MinIOClient
	VideoRepo     *repository.VideoRepo
	RabbitChannel *amqp.Channel // 用於發布轉碼工作訊息的 RabbitMQ Channel
}

// NewStreamingUseCase 建立一個新的 UserUseCase
func NewStreamingUseCase(minIO *database.MinIOClient,
	repo *repository.VideoRepo,
	rabbitChannel *amqp.Channel,
) StreamingUseCase {
	return &streamingUseCase{
		MinioClient:   minIO,
		VideoRepo:     repo,
		RabbitChannel: rabbitChannel,
	}
}

// UploadVideo 接收上傳請求，完成上傳、資料庫寫入與發布轉碼工作訊息
func (s *streamingUseCase) UploadVideo(up domain.UploadVideoReq) (*domain.UploadVideoRes, error) {

	tmpDir := "./tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, errors.New("建立暫存目錄失敗")
	}
	tempPath := filepath.Join(tmpDir, up.FileName)
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("建立暫存檔案失敗: %w", err)
	}

	// 寫入檔案資料
	if _, err := io.Copy(tempFile, up.File); err != nil {
		tempFile.Close()
		return nil, fmt.Errorf("儲存檔案失敗: %w", err)
	}
	tempFile.Close()

	// 4. 建立影片記錄（狀態預設為 "uploaded"）
	video := repository.Video{
		Title:       up.Title,
		Description: up.Description,
		FileName:    up.FileName, // 先暫存用，後續更新為 MinIO 的 object key
		Type:        up.Type,
		Status:      "uploaded",
	}
	if err := s.VideoRepo.Create(&video); err != nil {
		return nil, errors.New("資料庫建立影片失敗")
	}

	// 5. 定義 MinIO 儲存路徑，例如 "original/{videoID}/{filename}"
	objectName := fmt.Sprintf("original/%d/%s", video.ID, up.FileName)
	ctx := context.Background()
	if err := s.MinioClient.UploadFile(ctx, objectName, tempPath, "video/mp4"); err != nil {
		return nil, errors.New("上傳 MinIO 失敗")
	}

	// 6. 更新影片記錄，將 FileName 更新為 MinIO 上的 objectName
	video.FileName = objectName
	if err := s.VideoRepo.Update(&video); err != nil {
		return nil, errors.New("更新影片記錄失敗")
	}

	// 7. 發布轉碼工作訊息到消息佇列 (Producer 動作)
	job := domain.TranscodingJob{
		VideoID:  video.ID,
		FileName: video.FileName,
		Type:     video.Type,
	}
	data, err := json.Marshal(job)
	if err != nil {
		logger.Log.Errorf("Job JSON 轉換失敗: %v", err)
		return nil, errors.New("訊息序列化失敗")
	}
	err = s.RabbitChannel.Publish(
		"",               // 預設 exchange
		domain.QueueName, // queue 名稱
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
	if err != nil {
		logger.Log.Errorf("發送 RabbitMQ 訊息失敗: %v", err)
		return nil, errors.New("發布轉碼工作訊息失敗")
	}

	// 8. 可選：清理本地暫存檔案
	if err := os.Remove(tempPath); err != nil {
		logger.Log.Errorf("清理暫存檔案失敗:", err)
	}

	return &domain.UploadVideoRes{
		Message: "上傳成功，等待轉碼",
		VideoID: int(video.ID),
	}, nil

}

// GetVideo get video
func (s *streamingUseCase) GetVideo(videoID string) (*domain.GetVideoRes, error) {
	id, _ := strconv.Atoi(videoID)
	video, err := s.VideoRepo.GetByID(uint(id))
	if err != nil {
		return nil, errors.New("找不到影片")
	}
	if video.Status != "ready" {
		return nil, errors.New("影片尚未處理完成")
	}

	hlsURL := fmt.Sprintf("http://%s/video/hls/%d/index.m3u8", "127.0.0.1:8083", video.ID)

	return &domain.GetVideoRes{
		VideoID: int(video.ID),
		Title:   video.Title,
		HlsURL:  hlsURL,
	}, nil

}

// Search Search video
func (s *streamingUseCase) Search(keyWord string) ([]domain.Video, error) {
	videos, err := s.VideoRepo.SearchVideos(keyWord)
	if err != nil {
		return nil, errors.New("搜尋失敗")
	}

	videosRes := make([]domain.Video, len(videos))
	for i, video := range videos {
		videosRes[i] = domain.Video{
			ID:          video.ID,
			Title:       video.Title,
			Description: video.Description,
			FileName:    video.FileName,
			Type:        video.Type,
			Status:      video.Status,
			ViewCount:   video.ViewCount,
		}
	}

	return videosRes, nil
}

// GetRecommendations get recommendations
func (s *streamingUseCase) GetRecommendations(limit int) ([]domain.Video, error) {
	videos, err := s.VideoRepo.RecommendVideos(limit)
	if err != nil {
		return nil, errors.New("推薦失敗")
	}

	videosRes := make([]domain.Video, len(videos))
	for i, video := range videos {
		videosRes[i] = domain.Video{
			ID:          video.ID,
			Title:       video.Title,
			Description: video.Description,
			FileName:    video.FileName,
			Type:        video.Type,
			Status:      video.Status,
			ViewCount:   video.ViewCount,
		}
	}
	return videosRes, nil
}

// GetIndexM3U8 實現取得 m3u8 播放清單
func (s *streamingUseCase) GetIndexM3U8(ctx context.Context, videoID string) ([]byte, error) {
	// 組合 object key，與原先 HTTP 版本保持一致
	objectKey := "processed/" + videoID + "/index.m3u8"

	// 对象存在后，再获取对象
	obj, err := s.MinioClient.Client.GetObject(ctx, s.MinioClient.BucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "無法取得 m3u8 檔案: %v", err)
	}
	defer obj.Close()

	// 读取全部内容
	content, err := ioutil.ReadAll(obj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "讀取 m3u8 檔案失敗: %v", err)
	}

	return content, nil

	// 從 MinIO 取得 object
	// obj, err := s.MinioClient.Client.GetObject(ctx, s.MinioClient.BucketName, objectKey, minio.GetObjectOptions{})
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "無法取得 m3u8 檔案: %v", err)
	// }
	// defer obj.Close()
	// fmt.Printf("GetHlsSegment obj : %v \n", obj)
	// // 讀取全部內容
	// content, err := ioutil.ReadAll(obj)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "讀取 m3u8 檔案失敗: %v", err)
	// }

	// return content, nil
}

// GetHlsSegment 實現取得 TS 分段檔案
func (s *streamingUseCase) GetHlsSegment(ctx context.Context, videoID, segment string) ([]byte, error) {
	// 組合 object key，例如 "processed/{videoID}/{segment}"
	objectKey := "processed/" + videoID + "/" + segment

	// 对象存在后，再获取对象
	obj, err := s.MinioClient.Client.GetObject(ctx, s.MinioClient.BucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "無法取得 m3u8 檔案: %v", err)
	}
	defer obj.Close()

	// 读取全部内容
	content, err := ioutil.ReadAll(obj)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "讀取 m3u8 檔案失敗: %v", err)
	}

	return content, nil
}
