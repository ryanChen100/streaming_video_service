package database

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOEndpoint save minio endpoint
var MinIOEndpoint string

// MinIOClient definition minio client
type MinIOClient struct {
	Client     *minio.Client
	BucketName string
}

// NewMinIOConnection create a new minio connection have retry
func NewMinIOConnection(d MinIOConnection) (*MinIOClient, error) {
	var mc *MinIOClient
	var err error

	for i := 1; i <= d.RetryCount; i++ {
		mc, err = NewMinioClient(d.Endpoint, d.User, d.Password, d.BucketName, d.UseSSL)
		if err == nil {
			MinIOEndpoint = d.Endpoint
			log.Printf("minIO[%s] 連線成功 (嘗試 %d 次)", d.Endpoint, i)
			return mc, nil
		}

		log.Printf("minIO[%s] 連線失敗 (嘗試 %d/%d): %v", d.Endpoint, i, d.RetryCount, err)
		time.Sleep(d.RetryInterval * time.Second)
	}

	return mc, err
}

// NewMinioClient create a new minio
func NewMinioClient(endpoint, accessKey, secretKey, bucketName string, useSSL bool) (*MinIOClient, error) {
	minioClient, err := minio.New(endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
		})
	if err != nil {
		return nil, fmt.Errorf("初始化 MinIO 失敗: %v", err)
	}

	ctx := context.Background()
	// 檢查 bucket 是否存在
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		// 如果檢查出錯，返回錯誤
		return nil, fmt.Errorf("檢查 bucket [%s] 失敗: %v", bucketName, err)
	}

	// 如果 bucket 不存在，嘗試建立
	if !exists {
		if err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			// 如果建立過程出錯，返回錯誤
			return nil, fmt.Errorf("建立 bucket [%s] 失敗: %v", bucketName, err)
		}
		// 記錄成功建立的日志
		log.Printf("Bucket [%s] 建立成功", bucketName)
	} else {
		// 如果 bucket 已經存在，記錄日志
		log.Printf("Bucket [%s] 已存在", bucketName)
	}

	return &MinIOClient{
		Client:     minioClient,
		BucketName: bucketName,
	}, nil
}

// UploadFile minio upload file func
func (m *MinIOClient) UploadFile(ctx context.Context, objectName, filePath, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("開啟檔案失敗: %v", err)
	}
	defer file.Close()

	_, err = m.Client.PutObject(ctx, m.BucketName, objectName, file, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// DownloadFile minio download file func
func (m *MinIOClient) DownloadFile(ctx context.Context, objectName, destPath string) error {
	obj, err := m.Client.GetObject(ctx, m.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("取得物件失敗: %v", err)
	}
	defer obj.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("建立檔案失敗: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, obj)
	return err
}

// PresignGetURL 生成一個 Presigned URL 用來獲取指定的 object
func (m *MinIOClient) PresignGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	// 可以傳入額外參數，這裡傳 nil
	reqParams := make(url.Values)
	presignedURL, err := m.Client.PresignedGetObject(ctx, m.BucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("生成 Presigned URL 失敗: %w", err)
	}
	return presignedURL.String(), nil
}
