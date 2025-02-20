package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"streaming_video_service/internal/chat/domain"
	"streaming_video_service/pkg/logger"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedisPubSub definition redis pub/sub
type RedisPubSub struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisPubSub create RedisPubSub
func NewRedisPubSub(client *redis.Client) *RedisPubSub {
	return &RedisPubSub{
		client: client,
		ctx:    context.Background(),
	}
}

// Publish 將 message 序列化後，發布到指定 channel
func (r *RedisPubSub) Publish(channel string, message interface{}) error {
	// channel := fmt.Sprintf("chat:user:%s", userID)
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.client.Publish(r.ctx, channel, data).Err()
}

// Subscribe 訂閱自己member ID，收到訊息後呼叫 handler 處理
func (r *RedisPubSub) Subscribe(ctx context.Context, channel string, handler func(resp domain.WSResponse)) error {
	// channel := fmt.Sprintf("chat:user:%s", userID)
	sub := r.client.Subscribe(r.ctx, channel)
	go func() {
		ch := sub.Channel()

		for {
			select {
			case m, ok := <-ch:
				if !ok {
					return
				}

				var result domain.ChatMessage
				if err := json.Unmarshal([]byte(m.Payload), &result); err != nil {
					logger.Log.Error("Logout err :", zap.String("err", fmt.Sprintf("failed to unmarshal session data: %v", err)))
					continue
				}

				resp := domain.WSResponse{
					Action:  string(domain.NotifyMessage), // 你可以按需要设定 action
					Success: true,
					Payload: map[string]interface{}{
						"message_id": result.ID,
						"sender_id":  result.SenderID,
						"message":    result.Content,
						"timestamp":  result.Timestamp,
					},
				}
				handler(resp)
			case <-ctx.Done():
				logger.Log.Info(fmt.Sprintf("%s , sub close", channel))
				// 當 ctx 被取消時，退出循環並關閉訂閱
				sub.Close()
			}
		}
	}()
	return nil
}
