package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"streaming_video_service/internal/streaming/domain"
	"streaming_video_service/internal/streaming/repository"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/logger"

	// Kafka 客戶端
	"github.com/streadway/amqp" // RabbitMQ 客戶端
)

// Consumer 定義一個消息消費者，將所有必要的依賴注入進來
type Consumer struct {
	rabbitChannel *amqp.Channel
	minioClient   *database.MinIOClient
	videoRepo     *repository.VideoRepo
	queueName     string
}

// NewConsumer 建構 Consumer 實例
func NewConsumer(rabbitChannel *amqp.Channel, minioClient *database.MinIOClient, videoRepo *repository.VideoRepo, queueName string) *Consumer {
	return &Consumer{
		rabbitChannel: rabbitChannel,
		minioClient:   minioClient,
		videoRepo:     videoRepo,
		queueName:     queueName,
	}
}

// StartConsumer 開始消費訊息，並處理轉碼工作
func (c *Consumer) StartConsumer(ctx context.Context) {
	// 設定消費該 queue
	msgs, err := c.rabbitChannel.Consume(
		c.queueName, // 使用依賴注入進來的 queue name
		"",          // consumer tag，留空由系統分配
		false,       // autoAck 為 false，使用手動確認
		false,       // exclusive
		false,       // noLocal
		false,       // noWait
		nil,         // arguments
	)
	if err != nil {
		log.Fatalf("無法開始消費 RabbitMQ 訊息: %v", err)
	}

	log.Println("Consumer 已啟動，等待轉碼工作訊息...")

	// 持續監聽訊息
	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				log.Println("RabbitMQ 消費 channel 已關閉")
				return
			}

			// 處理收到的訊息
			var job domain.TranscodingJob
			if err := json.Unmarshal(d.Body, &job); err != nil {
				log.Printf("解析轉碼工作訊息失敗: %v", err)
				// 若解析失敗，拒絕並重新排入佇列
				if err := d.Nack(false, true); err != nil {
					log.Printf("Nack 訊息失敗: %v", err)
				}
				continue
			}

			log.Printf("收到轉碼工作訊息: VideoID=%d, FileName=%s, Type=%s", job.VideoID, job.FileName, job.Type)

			// 呼叫 processTranscodingJob 執行轉碼工作
			if err := processTranscodingJob(ctx, job, c.minioClient, c.videoRepo); err != nil {
				log.Printf("處理轉碼工作失敗: %v", err)
				// 處理失敗時，拒絕訊息並重新排入佇列

				logger.Log.Errorf("處理轉碼工作失敗:", err)
				time.Sleep(10 * time.Second)
				if err := d.Nack(false, true); err != nil {
					log.Printf("Nack 訊息失敗: %v", err)
				}
				continue
			}

			// 處理成功後，確認訊息
			if err := d.Ack(false); err != nil {
				log.Printf("確認訊息失敗: %v", err)
			} else {
				log.Printf("成功處理並確認訊息，VideoID: %d", job.VideoID)
			}
		case <-ctx.Done():
			log.Println("Consumer 收到停止訊號")
			return
		}
	}
}

// processTranscodingJob 負責執行轉碼工作：
// 1. 從 MinIO 下載原始影片檔
// 2. 使用 FFmpeg 轉碼成 HLS (可依需求擴展到 DASH 或多碼率轉碼)
// 3. 將轉碼結果上傳到 MinIO 的 processed/{videoID}/ 目錄
// 4. 更新資料庫中該影片的狀態為 "ready"
// 5. 清理本地暫存檔案
func processTranscodingJob(ctx context.Context, job domain.TranscodingJob, mClient *database.MinIOClient, videoRepo *repository.VideoRepo) error {
	// 1. 定義本地檔案的暫存路徑
	localInputPath := fmt.Sprintf("./tmp/%d_original.mp4", job.VideoID)
	localOutputDir := fmt.Sprintf("./tmp/%d_processed", job.VideoID)

	// 2. 從 MinIO 下載原始影片檔
	log.Printf("下載原始影片，VideoID: %d, ObjectKey: %s", job.VideoID, job.FileName)
	if err := mClient.DownloadFile(ctx, job.FileName, localInputPath); err != nil {
		return fmt.Errorf("下載原始影片失敗: %w", err)
	}

	// 3. 建立本地轉碼輸出目錄
	if err := os.MkdirAll(localOutputDir, 0755); err != nil {
		return fmt.Errorf("建立轉碼輸出目錄失敗: %w", err)
	}

	// 4. 呼叫 FFmpeg 進行轉碼
	// 根據影片類型，你可以選擇不同的轉碼策略，這裡以 HLS 為例
	log.Printf("開始轉碼影片 VideoID: %d 為 HLS 格式", job.VideoID)
	if err := TranscodeToHLS(localInputPath, localOutputDir); err != nil {
		return fmt.Errorf("FFmpeg HLS 轉碼失敗: %w", err)
	}

	// 5. 將轉碼結果上傳回 MinIO
	// 假設轉碼後在 localOutputDir 會產生 index.m3u8 與 TS 段檔
	files, err := ioutil.ReadDir(localOutputDir)
	if err != nil {
		return fmt.Errorf("讀取轉碼輸出目錄失敗: %w", err)
	}
	for _, file := range files {
		localFilePath := filepath.Join(localOutputDir, file.Name())
		// 定義上傳到 MinIO 的 object key，例如：processed/{videoID}/{檔名}
		objectName := fmt.Sprintf("processed/%d/%s", job.VideoID, file.Name())
		log.Printf("上傳轉碼結果檔案 %s 至 MinIO ObjectKey: %s", localFilePath, objectName)
		if err := mClient.UploadFile(ctx, objectName, localFilePath, getContentType(objectName)); err != nil { //TODO
			return fmt.Errorf("上傳轉碼結果失敗: %w", err)
		}
	}

	// 6. 更新資料庫中該影片的狀態為 "ready"
	video, err := videoRepo.GetByID(job.VideoID)
	if err != nil {
		return fmt.Errorf("從資料庫取得影片失敗: %w", err)
	}
	video.Status = "ready"
	if err := videoRepo.Update(video); err != nil {
		return fmt.Errorf("更新影片狀態失敗: %w", err)
	}
	log.Printf("影片 VideoID: %d 狀態更新為 ready", job.VideoID)

	// 7. 清理本地暫存檔案
	if err := os.Remove(localInputPath); err != nil {
		log.Printf("警告：清理本地原始檔失敗: %v", err)
	}
	if err := os.RemoveAll(localOutputDir); err != nil {
		log.Printf("警告：清理本地轉碼輸出目錄失敗: %v", err)
	}

	return nil
}

func getContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".m3u8":
		return "application/vnd.apple.mpegurl"
	case ".ts":
		return "video/MP2T"
	default:
		return "application/octet-stream"
	}
}

// 在消息消費端（例如 RabbitMQ 消費端）的某個函式中：
func consumeTranscodingMessage(ctx context.Context, message []byte, mClient *database.MinIOClient, videoRepo *repository.VideoRepo) {
	var job domain.TranscodingJob
	if err := json.Unmarshal(message, &job); err != nil {
		log.Printf("解析轉碼工作訊息失敗: %v", err)
		return
	}

	if err := processTranscodingJob(ctx, job, mClient, videoRepo); err != nil {
		log.Printf("處理轉碼工作失敗: %v", err)
		// 根據需求，你可以選擇重試此消息或記錄錯誤
	} else {
		log.Printf("成功處理轉碼工作，VideoID: %d", job.VideoID)
	}
}
