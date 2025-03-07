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

// MinIOClientRepo 定義 MinIO 需要實作的介面
type MinIOClientRepo interface {
	UploadFile(ctx context.Context, objectName, filePath, contentType string) error
	DownloadFile(ctx context.Context, objectName, destPath string) error
	PresignGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error)
	GetObject(ctx context.Context, objectName string, opts minio.GetObjectOptions) (io.Reader, error)
}

// MinIOClient definition minio client
// MinIOClient 結構體，負責與 MinIO 互動
type minIOClient struct {
	client     *minio.Client
	bucketName string
}

// NewMinIOConnection create a new minio connection have retry
func NewMinIOConnection(d MinIOConnection) (MinIOClientRepo, error) {
	minioClient, err := minio.New(d.Endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(d.User, d.Password, ""),
			Secure: d.UseSSL,
		})
	if err != nil {
		return nil, fmt.Errorf("初始化 MinIO 失敗: %v", err)
	}

	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, d.BucketName)
	if err != nil {
		return nil, fmt.Errorf("檢查 bucket [%s] 失敗: %v", d.BucketName, err)
	}

	if !exists {
		if err = minioClient.MakeBucket(ctx, d.BucketName, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("建立 bucket [%s] 失敗: %v", d.BucketName, err)
		}
		log.Printf("Bucket [%s] 建立成功", d.BucketName)
	} else {
		log.Printf("Bucket [%s] 已存在", d.BucketName)
	}

	return &minIOClient{
		client:     minioClient,
		bucketName: d.BucketName,
	}, nil
}

// func (m *minIOClient) GetClient() *minio.Client {
// 	return m.client
// }

// func (m *minIOClient) GetBucketName() string {
// 	return m.bucketName
// }

// UploadFile 上傳檔案到 MinIO
func (m *minIOClient) UploadFile(ctx context.Context, objectName, filePath, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("開啟檔案失敗: %v", err)
	}
	defer file.Close()

	_, err = m.client.PutObject(ctx, m.bucketName, objectName, file, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// DownloadFile 下載檔案
func (m *minIOClient) DownloadFile(ctx context.Context, objectName, destPath string) error {
	obj, err := m.client.GetObject(ctx, m.bucketName, objectName, minio.GetObjectOptions{})
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

// PresignGetURL 生成一個 Presigned URL
func (m *minIOClient) PresignGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := m.client.PresignedGetObject(ctx, m.bucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("生成 Presigned URL 失敗: %w", err)
	}
	return presignedURL.String(), nil
}

// GetObject 取得 minio object
func (m *minIOClient) GetObject(ctx context.Context, objectName string, opts minio.GetObjectOptions) (io.Reader, error) {
	return m.client.GetObject(ctx, m.bucketName, objectName, opts)
}
