package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// NewKafkaWriterWithRetry 嘗試建立 Kafka Writer 並發送測試訊息以確認連線
func NewKafkaWriterWithRetry(k KafkaConnection) (*kafka.Writer, error) {
	var writer *kafka.Writer
	var err error

	for attempt := 1; attempt <= k.RetryCount; attempt++ {
		writer = kafka.NewWriter(kafka.WriterConfig{
			Brokers:  k.Brokers,
			Topic:    k.Topic,
			Balancer: &kafka.LeastBytes{},
		})

		// 發送一個測試訊息（例如 "ping"），確認連線是否成功
		err = writer.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte("ping"),
			Value: []byte("ping"),
		})
		if err == nil {
			log.Printf("Kafka Writer 建立成功 (嘗試 %d 次)", attempt)
			// 若不想保留測試訊息，可考慮後續處理；這裡僅用作確認連線。
			return writer, nil
		}

		log.Printf("Kafka Writer 建立失敗 (嘗試 %d/%d): %v", attempt, k.RetryCount, err)
		writer.Close()
		time.Sleep(k.RetryInterval * time.Second)
	}

	return nil, fmt.Errorf("無法建立 Kafka Writer，經過 %d 次嘗試: %v", k.RetryCount, err)
}
