package domain

const (
	//QueueName definition queue name
	QueueName = "transcode"
)

// TranscodingJob 定義轉碼工作訊息
type TranscodingJob struct {
	VideoID  uint   `json:"video_id"`
	FileName string `json:"file_name"` // 原始檔在 MinIO 上的 object key
	Type     string `json:"type"`      // "short" 或 "long"
}
