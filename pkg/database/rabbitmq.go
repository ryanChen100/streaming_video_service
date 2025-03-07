package database

import (
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// RabbitRepo definition rabbit repo
type RabbitRepo interface {
	GetRabbit() *amqp.Channel
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
}

type rabbitRepo struct {
	channel *amqp.Channel
}

// NewRabbitRepository create a RabbitRepository
func NewRabbitRepository(db *amqp.Channel) RabbitRepo {
	return &rabbitRepo{channel: db}
}

// ConnectRabbitMQWithRetry 嘗試連線到 RabbitMQ，並使用指數回退進行重試。
func ConnectRabbitMQWithRetry(d Connection) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	for attempt := 1; attempt <= d.RetryCount; attempt++ {
		conn, err = amqp.Dial(d.ConnectStr)
		if err == nil {
			log.Printf("RabbitMQ[%s]Q 連線成功 (嘗試 %d 次)", d.ConnectStr, attempt)
			return conn, nil
		}

		log.Printf("RabbitMQ[%s] 連線失敗 (嘗試 %d/%d): %v", d.ConnectStr, attempt, d.RetryCount, err)
		time.Sleep(d.RetryInterval * time.Second)
	}

	return nil, fmt.Errorf("無法連線 RabbitMQ[%s]，經過 %d 次嘗試: %v", d.ConnectStr, d.RetryCount, err)
}

// GetRabbitMQChannelWithRetry 使用已有的 RabbitMQ 連線嘗試取得 Channel
func GetRabbitMQChannelWithRetry(conn *amqp.Connection, maxRetries int, baseDelay time.Duration) (*amqp.Channel, error) {
	var ch *amqp.Channel
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ch, err = conn.Channel()
		if err == nil {
			log.Printf("RabbitMQ Channel 建立成功 (嘗試 %d 次)", attempt)
			return ch, nil
		}

		log.Printf("建立 RabbitMQ Channel 失敗 (嘗試 %d/%d): %v", attempt, maxRetries, err)
		time.Sleep(baseDelay * time.Second)
	}

	return nil, fmt.Errorf("無法取得 RabbitMQ Channel，經過 %d 次嘗試: %v", maxRetries, err)
}

func (r *rabbitRepo) GetRabbit() *amqp.Channel {
	return r.channel
}

func (r *rabbitRepo) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return r.channel.Publish(exchange, key, mandatory, immediate, msg)
}
