package domain

import "io"

// UploadVideoReq usecase upload video request
type UploadVideoReq struct {
	Title       string
	Description string
	Type        string
	FileName    string
	File        io.Reader
}

// UploadVideoRes usecase upload video response
type UploadVideoRes struct {
	Message string
	VideoID int
}

// GetVideoRes usecase get video response
type GetVideoRes struct {
	VideoID int
	Title   string
	HlsURL  string
}

// Video 定義影片模型
type Video struct {
	ID          uint `gorm:"primaryKey"`
	Title       string
	Description string
	FileName    string // 存於 MinIO 上的 object key
	Type        string // "short" 或 "long"
	Status      string // "uploaded", "processing", "ready"
	ViewCount   uint   // 瀏覽次數
	// 可加入 UserID、CreatedAt 等欄位
}
