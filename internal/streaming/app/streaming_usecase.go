package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"streaming_video_service/internal/streaming/domain"
	"streaming_video_service/internal/streaming/repository"
	"streaming_video_service/pkg/database"
	errprocess "streaming_video_service/pkg/err"

	"github.com/minio/minio-go/v7"
	"github.com/streadway/amqp"
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
	MinioClient   database.MinIOClientRepo
	VideoRepo     repository.VideoRepo
	RabbitChannel database.RabbitRepo // 用於發布轉碼工作訊息的 RabbitMQ Channel
}

// NewStreamingUseCase 建立一個新的 UserUseCase
func NewStreamingUseCase(minIO database.MinIOClientRepo,
	repo repository.VideoRepo,
	rabbitChannel database.RabbitRepo,
) StreamingUseCase {
	return &streamingUseCase{
		MinioClient:   minIO,
		VideoRepo:     repo,
		RabbitChannel: rabbitChannel,
	}
}

// 讓 `streaming_usecase` test mock使用包裝函數 詳情轉跳至 jwt_wrapper.go
var (
	createDir = func(path string) error {
		return os.MkdirAll(path, 0755)
	}

	createFile = func(name string) (*os.File, error) {
		return os.Create(name)
	}

	copyFile = func(dst *os.File, src io.Reader) (written int64, err error) {
		return io.Copy(dst, src)
	}

	readFile = func(r io.Reader) ([]byte, error) {
		return io.ReadAll(r)
	}
)

// 1. 解決大檔案處理的問題
//   - 直接處理流（streaming upload）可能會有記憶體壓力問題
//   - 假如 up.File 是一個 io.Reader（如 HTTP multipart 或 gRPC stream），如果直接讀取所有內容到記憶體，可能會造成 Out of Memory（OOM），特別是當上傳大影片時。
//   - 解法：透過暫存檔案，逐步寫入磁碟，避免一次性占用大量記憶體。
//
// 2. 確保 MinIO 上傳前的完整性
//   - 直接上傳 io.Reader 可能會有問題
//   - MinIO 的 UploadFile 需要明確的檔案路徑，若 up.File 是 stream，直接傳遞可能導致不完整的資料。
//   - 解法：先存成本地檔案，再透過 MinIO SDK 確保完整性後上傳，避免發生中途失敗導致影片損壞。
//
// 3. 支援異步與重試機制
//   - MinIO、RabbitMQ 可能會發生短暫錯誤
//   - 若影片未完整寫入 MinIO 或 RabbitMQ 發送失敗，使用暫存檔案可以提供「重試」機制，而不會因為檔案已經消失而失敗。
//   - 解法：
//   - 成功上傳 MinIO 後才刪除暫存檔案。
//   - 若 RabbitMQ 發送失敗，可考慮「重試」或標記影片為 pending，稍後重新發送。
//
// 4. 支援本地快取（Cache）與日後 Debug
//   - 若影片上傳過程失敗，開發者可以手動檢查 ./tmp 目錄，確保問題是發生在：
//     1.	影片上傳前的讀取問題？
//     2.	MinIO 上傳失敗？
//     3.	RabbitMQ 無法發送？
//   - 這對於除錯（Debug）非常有幫助。
//
// UploadVideo 接收上傳請求，完成上傳、資料庫寫入與發布轉碼工作訊息
func (s *streamingUseCase) UploadVideo(up domain.UploadVideoReq) (*domain.UploadVideoRes, error) {
	tmpDir := "./tmp"
	if err := createDir(tmpDir); err != nil {
		errMsg := fmt.Sprintf("fileName[%s] 建立暫存目錄失敗 : %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
	}

	tempPath := filepath.Join(tmpDir, up.FileName)
	tempFile, err := createFile(tempPath)
	if err != nil {
		errMsg := fmt.Sprintf("fileName[%s] 建立暫存檔案失敗 : %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
	}
	defer tempFile.Close()

	// 寫入檔案資料
	if _, err := copyFile(tempFile, up.File); err != nil {
		tempFile.Close()
		errMsg := fmt.Sprintf("fileName[%s] 儲存檔案失敗 : %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
	}

	// 4. 建立影片記錄（狀態預設為 "uploaded"）
	video := domain.Video{
		Title:       up.Title,
		Description: up.Description,
		FileName:    up.FileName, // 先暫存用，後續更新為 MinIO 的 object key
		Type:        up.Type,
		Status:      string(domain.VideoUpload),
	}

	if err := s.VideoRepo.Create(&video); err != nil {
		errMsg := fmt.Sprintf("fileName[%s] 資料庫建立影片失敗 : %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
	}

	// 5. 定義 MinIO 儲存路徑，例如 "original/{videoID}/{filename}"
	objectName := fmt.Sprintf("original/%d/%s", video.ID, up.FileName)
	ctx := context.Background()
	if err := s.MinioClient.UploadFile(ctx, objectName, tempPath, "video/mp4"); err != nil {
		errMsg := fmt.Sprintf("fileName[%s] 上傳 MinIO 失敗 : %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
	}

	// 6. 更新影片記錄，將 FileName 更新為 MinIO 上的 objectName
	video.FileName = objectName
	if err := s.VideoRepo.Update(&video); err != nil {
		errMsg := fmt.Sprintf("fileName[%s] 更新影片記錄失敗 : %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
	}

	// 7. 發布轉碼工作訊息到消息佇列 (Producer 動作)
	job := domain.TranscodingJob{
		VideoID:  video.ID,
		FileName: video.FileName,
		Type:     video.Type,
	}
	data, err := json.Marshal(job)
	if err != nil {
		errMsg := fmt.Sprintf("fileName[%s] Job JSON 訊息序列化失敗轉換失敗 : %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
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
		errMsg := fmt.Sprintf("fileName[%s] 發送 RabbitMQ 訊息失敗 : %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
	}

	// 8. 可選：清理本地暫存檔案
	if err := os.Remove(tempPath); err != nil {
		errMsg := fmt.Sprintf("fileName[%s] 清理暫存檔案失敗: %v", up.FileName, err)
		return nil, errprocess.Set(errMsg)
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
		errMsg := fmt.Sprintf("videoID[%s] 找不到影片: %v", videoID, err)
		return nil, errprocess.Set(errMsg)
	}
	if video.Status != string(domain.VideoReady) {
		errMsg := fmt.Sprintf("videoID[%s] 影片尚未處理完成", videoID)
		return nil, errprocess.Set(errMsg)
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
		errMsg := fmt.Sprintf("keyword[%s] search err : %v", keyWord, err)
		return nil, errprocess.Set(errMsg)
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
		errMsg := fmt.Sprintf("limit[%d] get recommendations err : %v", limit, err)
		return nil, errprocess.Set(errMsg)
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
	obj, err := s.MinioClient.GetObject(ctx, objectKey, minio.GetObjectOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("videoID[%s] 無法取得 m3u8 檔案 : %v", videoID, err)
		return nil, errprocess.Set(errMsg)
	}
	// defer obj.Close()

	// 读取全部内容
	content, err := readFile(obj)
	if err != nil {
		errMsg := fmt.Sprintf("videoID[%s] 讀取 m3u8 檔案失敗 : %v", videoID, err)
		return nil, errprocess.Set(errMsg)
	}

	return content, nil
}

// GetHlsSegment 實現取得 TS 分段檔案
func (s *streamingUseCase) GetHlsSegment(ctx context.Context, videoID, segment string) ([]byte, error) {
	// 組合 object key，例如 "processed/{videoID}/{segment}"
	objectKey := "processed/" + videoID + "/" + segment

	// 对象存在后，再获取对象
	obj, err := s.MinioClient.GetObject(ctx, objectKey, minio.GetObjectOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("videoID_segment[%s_%s] 無法取得 segment 檔案 : %v", videoID, segment, err)
		return nil, errprocess.Set(errMsg)
	}
	// defer obj.Close()

	// 读取全部内容
	content, err := readFile(obj)
	if err != nil {
		errMsg := fmt.Sprintf("videoID_segment[%s_%s] 讀取 segment 檔案失敗 : %v", videoID, segment, err)
		return nil, errprocess.Set(errMsg)
	}

	return content, nil
}
