package database

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Connection definition slq setting
type Connection struct {
	ConnectStr string

	RetryCount    int
	RetryInterval time.Duration
}

// MongoDB definition mongo db
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// MinIOConnection definition minio
type MinIOConnection struct {
	Endpoint string
	User string
	Password string
	BucketName string
	UseSSL bool

	RetryCount    int
	RetryInterval time.Duration
}

// KafkaConnection definition kafka
type KafkaConnection struct {
	Brokers []string
	Topic string
	RetryCount    int
	RetryInterval time.Duration
}
