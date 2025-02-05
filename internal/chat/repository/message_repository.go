package repository

import (
	"context"
	"fmt"
	"time"

	"streaming_video_service/internal/chat/domain"
	"streaming_video_service/pkg"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MessageRepository definition create room info
type MessageRepository interface {
	// InsertMessages 在指定桶中新增多則訊息。如果桶不存在，可選擇創建新的桶。
	InsertMessages(ctx context.Context, bucket *domain.MessageBucket) error
	// FindBucket 查詢指定聊天室及日期的桶
	FindBucket(ctx context.Context, roomID, date string) (*domain.MessageBucket, error)
	// UpdateBucket 更新桶內的訊息（例如更新已讀狀態）
	UpdateBucket(ctx context.Context, bucket *domain.MessageBucket) error
	// FindUnreadMessages 從多個桶中拉取該用戶未讀的訊息
	FindEarliestUnread(ctx context.Context, userID, roomID string) (*domain.MessageBucket, error)
	FindMessagesBefore(ctx context.Context, roomID string, beforeTimestamp int64) ([]domain.ChatMessage, error)
	CountUnreadMessagesByRoom(ctx context.Context, userID string) ([]domain.RoomUnreadInfo, error)
}

type chatMessageRepository struct {
	coll *mongo.Collection
}

// NewMongoChatMessageRepository create a ChatMessageRepository
func NewMongoChatMessageRepository(db *mongo.Database) MessageRepository {
	return &chatMessageRepository{
		coll: db.Collection("chat_messages"),
	}
}

func (r *chatMessageRepository) FindBucket(ctx context.Context, roomID, date string) (*domain.MessageBucket, error) {
	filter := bson.M{"room_id": roomID, "date": date}
	var bucket domain.MessageBucket
	err := r.coll.FindOne(ctx, filter).Decode(&bucket)
	if err != nil {
		return nil, err
	}
	return &bucket, nil
}

// InsertMessage member insert message
// InsertMessage - 寫入一筆聊天訊息
func (r *chatMessageRepository) InsertMessages(ctx context.Context, bucket *domain.MessageBucket) error {
	// 嘗試插入新的桶，如果已存在可選擇更新
	_, err := r.coll.InsertOne(ctx, bucket)
	return err
}

// MarkAsRead - 將 messageID 的訊息加入 read_by: userID
func (r *chatMessageRepository) UpdateBucket(ctx context.Context, bucket *domain.MessageBucket) error {
	filter := bson.M{"room_id": bucket.RoomID, "date": bucket.Date}
	update := bson.M{"$set": bucket}
	_, err := r.coll.UpdateOne(ctx, filter, update)
	return err
}

// FindUnreadMessages - 尋找 userID 在 roomIDs 裡所有未讀訊息
func (r *chatMessageRepository) FindEarliestUnread(ctx context.Context, userID, roomID string) (*domain.MessageBucket, error) {
	// 建立查詢條件：符合指定房間
	filter := bson.M{"room_id": roomID}
	// 按日期升序排序（最早的桶在前）
	opts := options.Find()
	opts.SetSort(bson.M{"date": 1})
	cur, err := r.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var buckets []domain.MessageBucket
	if err := cur.All(ctx, &buckets); err != nil {
		return nil, err
	}

	// 遍歷每個 bucket，找到第一個包含至少一則未讀訊息的 bucket
	for _, bucket := range buckets {
		for _, msg := range bucket.Messages {
			if !pkg.Contains(msg.ReadBy, userID) {
				// 找到第一個含未讀訊息的 bucket，返回整個 bucket（即當天所有訊息）
				return &bucket, nil
			}
		}
	}
	// 如果都沒有未讀訊息，可以返回空或自定義錯誤
	return nil, nil
}

func (r *chatMessageRepository) FindMessagesBefore(ctx context.Context, roomID string, beforeTimestamp int64) ([]domain.ChatMessage, error) {
	// 假設先找出當天的 bucket，再過濾訊息（實際上可能需要聚合跨桶查詢）
	today := time.Unix(beforeTimestamp, 0).Format("2006-01-02")
	filter := bson.M{
		"room_id": roomID,
		"date":    today,
	}
	var bucket domain.MessageBucket
	err := r.coll.FindOne(ctx, filter).Decode(&bucket)
	if err != nil {
		return nil, err
	}
	var messages []domain.ChatMessage
	// 過濾出 timestamp < beforeTimestamp
	for _, msg := range bucket.Messages {
		if msg.Timestamp < beforeTimestamp {
			messages = append(messages, msg)
		}
	}
	// 如果 messages 數量大於 limit，取最新的 limit 筆（根據時間排序）
	// 實作上可進一步調整排序和分頁邏輯
	// if len(messages) > limit {
	// 	messages = messages[len(messages)-limit:]
	// }
	return messages, nil
}

func (r *chatMessageRepository) CountUnreadMessagesByRoom(ctx context.Context, userID string) ([]domain.RoomUnreadInfo, error) {
	pipeline := mongo.Pipeline{
		// 1. 展開每個 bucket 的 messages 陣列
		bson.D{{Key: "$unwind", Value: "$messages"}},
		// 2. 過濾出未讀訊息（ReadBy 不包含 userID）
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "messages.read_by", Value: bson.D{{Key: "$ne", Value: userID}}},
		}}},
		// 3. 按 room_id 分組，計算未讀數量和該組未讀訊息中的最大時間戳
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$room_id"},
			{Key: "unread_count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "last_unread_timestamp", Value: bson.D{{Key: "$max", Value: "$messages.timestamp"}}},
		}}},
		// 4. 根據 last_unread_timestamp 降序排序
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: "last_unread_timestamp", Value: -1},
		}}},
	}

	cur, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate error: %w", err)
	}

	type result struct {
		RoomID      string `bson:"_id"`
		UnreadCount int    `bson:"unread_count"`
	}

	var results []domain.RoomUnreadInfo
	if err := cur.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("cursor All error: %w", err)
	}

	return results, nil
}
