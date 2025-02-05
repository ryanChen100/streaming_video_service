package database

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// NewMongoDB create a new MongoDB connection
func NewMongoDB(ctx context.Context, c Connection, dbName string) (*MongoDB, error) {
	clientOpts := options.Client().ApplyURI(c.ConnectStr)

	var client *mongo.Client
	var err error

	for i := 0; i <= c.RetryCount; i++ {
		client, err = mongo.Connect(ctx, clientOpts)
		if err == nil {
			// Ping the database to verify the connection
			pingErr := client.Ping(ctx, readpref.Primary())
			if pingErr == nil {
				db := client.Database(dbName)
				return &MongoDB{
					Client:   client,
					Database: db,
				}, nil
			}
			err = pingErr

		}

		if i < c.RetryCount {
			time.Sleep(c.RetryInterval)
		}
	}

	return nil, errors.New("failed to connect to MongoDB after retries: " + err.Error())
}

// Close disenable mongoDB connection
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
