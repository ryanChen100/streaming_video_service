package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"streaming_video_service/internal/streaming/domain"
	"streaming_video_service/internal/streaming/repository"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"github.com/streadway/amqp"
)

// VideoHandler definition video handler
// VideoHandler 定義影片上傳處理器
type VideoHandler struct {
	MinioClient   *database.MinIOClient
	VideoRepo     *repository.VideoRepo
	RabbitChannel *amqp.Channel // 用於發布轉碼工作訊息的 RabbitMQ Channel
}

// UploadVideo 接收上傳請求，完成上傳、資料庫寫入與發布轉碼工作訊息
func (h *VideoHandler) UploadVideo(c *fiber.Ctx) error {
	// 1. 取得表單欄位
	title := c.FormValue("title")
	desc := c.FormValue("description")
	videoType := c.FormValue("type") // "short" 或 "long"

	// 2. 取得上傳檔案
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "未檢測到檔案"})
	}

	// 3. 暫存檔案到 ./tmp/ 目錄（先建立該目錄）
	tmpDir := "./tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "建立暫存目錄失敗"})
	}
	tempPath := filepath.Join(tmpDir, fileHeader.Filename)
	if err := c.SaveFile(fileHeader, tempPath); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "儲存檔案失敗"})
	}

	// 4. 建立影片記錄（狀態預設為 "uploaded"）
	video := repository.Video{
		Title:       title,
		Description: desc,
		FileName:    fileHeader.Filename, // 先暫存用，後續更新為 MinIO 的 object key
		Type:        videoType,
		Status:      "uploaded",
	}
	if err := h.VideoRepo.Create(&video); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "資料庫建立影片失敗"})
	}

	// 5. 定義 MinIO 儲存路徑，例如 "original/{videoID}/{filename}"
	objectName := fmt.Sprintf("original/%d/%s", video.ID, fileHeader.Filename)
	ctx := context.Background()
	if err := h.MinioClient.UploadFile(ctx, objectName, tempPath, "video/mp4"); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "上傳 MinIO 失敗"})
	}

	// 6. 更新影片記錄，將 FileName 更新為 MinIO 上的 objectName
	video.FileName = objectName
	if err := h.VideoRepo.Update(&video); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "更新影片記錄失敗"})
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
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "訊息序列化失敗"})
	}
	err = h.RabbitChannel.Publish(
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
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "發布轉碼工作訊息失敗"})
	}

	// 8. 可選：清理本地暫存檔案
	if err := os.Remove(tempPath); err != nil {
		logger.Log.Errorf("清理暫存檔案失敗:", err)
	}

	return c.JSON(fiber.Map{
		"msg":      "上傳成功，等待轉碼",
		"video_id": video.ID,
	})
}

// GetVideo get video
func (h *VideoHandler) GetVideo(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.Atoi(idStr)
	video, err := h.VideoRepo.GetByID(uint(id))
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "找不到影片"})
	}
	if video.Status != "ready" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "影片尚未處理完成"})
	}

	hlsURL := fmt.Sprintf("http://%s/video/hls/%d/index.m3u8", "127.0.0.1:8083", video.ID)

	// 生成播放 URL，此處假設 HLS 主檔上傳於 "processed/{videoID}/index.m3u8"
	// 若 CDN 整合，正式環境可將 host 換成 CDN 域名，例如 "https://cdn.example.com"
	// hlsURL := fmt.Sprintf("http://%s/%s/processed/%d/index.m3u8",
	// 	database.MinIOEndpoint, // MinIO 服務地址
	// 	h.MinioClient.BucketName,
	// 	video.ID,
	// )

	// 定義轉碼後檔案在 MinIO 的 object key (假設是 "processed/{videoID}/index.m3u8")
	// objectKey := fmt.Sprintf("processed/%d/index.m3u8", video.ID)

	// // 生成一個 Presigned URL，有效期例如設定為 15 分鐘
	// ctx := context.Background()
	// logger.Log.Infof("objectKey : ", objectKey)
	// presignedURL, err := h.MinioClient.PresignGetURL(ctx, objectKey, 60*time.Minute)
	// if err != nil {
	// 	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "生成 Presigned URL 失敗"})
	// }

	return c.JSON(fiber.Map{
		"video_id": video.ID,
		"title":    video.Title,
		"hls_url":  hlsURL,
	})
}

// Search Search video
func (h *VideoHandler) Search(c *fiber.Ctx) error {
	keyword := c.Query("q")
	videos, err := h.VideoRepo.SearchVideos(keyword)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "搜尋失敗"})
	}
	return c.JSON(videos)
}

// GetRecommendations get recommendations
func (h *VideoHandler) GetRecommendations(c *fiber.Ctx) error {
	var limit int

	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil {
		logger.Log.Errorf("GetRecommendations limit transfer err :", err)
		limit = 10
	}

	videos, err := h.VideoRepo.RecommendVideos(limit)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "推薦失敗"})
	}
	return c.JSON(videos)
}

// GetIndexM3U8 代理返回 m3u8 播放清單
func (h *VideoHandler) GetIndexM3U8(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.Atoi(idStr)
	// 在 MinIO 中對應的 object key，例如 "processed/{id}/index.m3u8"
	objectKey := fmt.Sprintf("processed/%d/index.m3u8", id)
	ctx := context.Background()

	obj, err := h.MinioClient.Client.GetObject(ctx, h.MinioClient.BucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("無法取得 m3u8 檔案: " + err.Error())
	}
	defer obj.Close()

	// 設置正確的 Content-Type
	c.Set("Content-Type", "application/vnd.apple.mpegurl")
	if _, err := io.Copy(c.Response().BodyWriter(), obj); err != nil {
		return c.Status(http.StatusInternalServerError).SendString("讀取 m3u8 檔案失敗: " + err.Error())
	}
	return nil
}

// GetHlsSegment 代理返回 TS 段檔案
func (h *VideoHandler) GetHlsSegment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	segment := c.Params("segment") // 例如 "index0.ts"
	id, _ := strconv.Atoi(idStr)
	objectKey := fmt.Sprintf("processed/%d/%s", id, segment)
	ctx := context.Background()
	obj, err := h.MinioClient.Client.GetObject(ctx, h.MinioClient.BucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString("無法取得 TS 檔案: " + err.Error())
	}
	defer obj.Close()

	c.Set("Content-Type", "video/mp2t")
	if _, err := io.Copy(c.Response().BodyWriter(), obj); err != nil {
		return c.Status(http.StatusInternalServerError).SendString("讀取 TS 檔案失敗: " + err.Error())
	}
	return nil
}
