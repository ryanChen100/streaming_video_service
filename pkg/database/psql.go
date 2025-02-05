package database

import (
	"context"
	"fmt"
	"streaming_video_service/pkg/logger"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

// NewDatabaseConnection create a new postgresSQL connection
func NewDatabaseConnection(d Connection) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	dbConfig, _ := pgxpool.ParseConfig(d.ConnectStr)
	for i := 0; i < d.RetryCount; i++ {
		pool, err = pgxpool.ConnectConfig(context.Background(), dbConfig)
		if err == nil {
			break
		}
		logger.Log.Warn(
			"Failed to connect to postgreSQL database, retrying...",
			zap.Int("attempt", i+1),
			zap.String("address", fmt.Sprintf("[%s]", d.ConnectStr)),
			zap.Error(err),
		)
		time.Sleep(d.RetryInterval * time.Second)
	}

	return pool, err
}
