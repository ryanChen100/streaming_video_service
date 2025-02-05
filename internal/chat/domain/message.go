package domain

// MessageBucket 表示某個聊天室某天的訊息存儲
type MessageBucket struct {
	RoomID   string        `bson:"room_id" json:"room_id"`
	Date     string        `bson:"date" json:"date"` // 格式："2025-01-23"
	Messages []ChatMessage `bson:"messages" json:"messages"`
}

// ChatMessage 表示一則聊天訊息
type ChatMessage struct {
	ID        string   `bson:"id" json:"id"` // 可以用 UUID
	SenderID  string   `bson:"sender_id" json:"sender_id"`
	Content   string   `bson:"content" json:"content"`
	Timestamp int64    `bson:"timestamp" json:"timestamp"`
	ReadBy    []string `bson:"read_by,omitempty" json:"read_by,omitempty"`
}

// RoomUnreadInfo definition unread by room
type RoomUnreadInfo struct {
	RoomID              string `bson:"_id" json:"room_id"`
	UnreadCount         int    `bson:"unread_count" json:"unread_count"`
	LastUnreadTimeStamp int64  `bson:"last_unread_timestamp" json:"last_unread_timestamp"`
}
