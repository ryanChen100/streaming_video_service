package app

import (
	"context"
	"errors"
	"log"
	"time"

	"streaming_video_service/internal/chat/domain"
	"streaming_video_service/internal/chat/repository"
	"streaming_video_service/pkg"

	"github.com/google/uuid"
)

// SendMessageUseCase 負責處理聊天訊息
type SendMessageUseCase struct {
	roomRepo     repository.RoomRepository
	msgRepo      repository.MessageRepository
	memberPubSub *repository.RedisPubSub
	roomPubSub   *repository.RedisPubSub
}

// NewSendMessageUseCase init create message use case
func NewSendMessageUseCase(
	roomRepo repository.RoomRepository,
	msgRepo repository.MessageRepository,
	pub *repository.RedisPubSub,
) *SendMessageUseCase {
	return &SendMessageUseCase{
		roomRepo:     roomRepo,
		msgRepo:      msgRepo,
		memberPubSub: pub,
		roomPubSub:   pub,
	}
}

// Execute send message
func (uc *SendMessageUseCase) Execute(ctx context.Context, roomID, senderID, content string) (string, error) {
	// 1. 檢查房間是否存在(可選)
	room, err := uc.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return "", err
	}
	if room == nil {
		return "", errors.New("room not found")
	}

	// 2. 建立訊息
	today := time.Now().Format("2006-01-02") // 例如 "2025-01-23"
	msgID := uuid.New().String()

	newMsg := domain.ChatMessage{
		ID:        msgID,
		SenderID:  senderID,
		Content:   content,
		Timestamp: time.Now().Unix(),
		ReadBy:    []string{senderID},
	}

	// 嘗試查詢今天的 bucket
	bucket, err := uc.msgRepo.FindBucket(ctx, roomID, today)
	if err != nil {
		// 如果找不到，表示今天還沒有 bucket，則創建新的 bucket
		bucket = &domain.MessageBucket{
			RoomID:   roomID,
			Date:     today,
			Messages: []domain.ChatMessage{newMsg},
		}
		if err := uc.msgRepo.InsertMessages(ctx, bucket); err != nil {
			return "", err
		}
	} else {
		// 如果 bucket 存在，追加訊息
		bucket.Messages = append(bucket.Messages, newMsg)
		// 當訊息數量過大時，你可以檢查 len(bucket.Messages) 是否超過閾值，
		// 並依照容量或筆數進行切割，這裡簡化為直接更新 bucket
		if err := uc.msgRepo.UpdateBucket(ctx, bucket); err != nil {
			return "", err
		}
	}

	// 4. pubSub 同步給房間內除自己的member
	if uc.memberPubSub != nil {
		for _, memberID := range room.Members {
			if memberID != senderID {
				if err := uc.memberPubSub.Publish(memberID, newMsg); err != nil {
					log.Printf("Publish error: %v", err)
				}
			}
		}
	}

	// 5) ephemeral broadcast(同節點):
	// h.hub.Broadcast(roomID, rawMsg)

	// 5. ephemeral broadcast - 同一節點可call hub.Broadcast(roomID, msg)
	// ...

	return msgID, nil
}

// MarkRead - 已讀
func (uc *SendMessageUseCase) MarkRead(ctx context.Context, roomID, messageID, userID string) error {
	// 1. 取得 bucket（假設是當天的，或你可以根據其他條件查詢）
	today := time.Now().Format("2006-01-02")
	bucket, err := uc.msgRepo.FindBucket(ctx, roomID, today)
	if err != nil || bucket == nil {
		return errors.New("bucket not found")
	}
	// 2. 遍歷 bucket.Messages，找到對應 messageID 並更新 read_by
	updated := false
	for i, msg := range bucket.Messages {
		if msg.ID == messageID {
			// 避免重複加入
			if !pkg.Contains(msg.ReadBy, userID) {
				bucket.Messages[i].ReadBy = append(bucket.Messages[i].ReadBy, userID)
				updated = true
			}
			break
		}
	}
	if !updated {
		return errors.New("message not found or already marked")
	}
	return uc.msgRepo.UpdateBucket(ctx, bucket)
}

// GetCountUnreadMessages - get member all room un read message
func (uc *SendMessageUseCase) GetCountUnreadMessages(ctx context.Context, userID string) ([]domain.RoomUnreadInfo, error) {
	return uc.msgRepo.CountUnreadMessagesByRoom(ctx, userID)
}
