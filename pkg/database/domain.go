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
