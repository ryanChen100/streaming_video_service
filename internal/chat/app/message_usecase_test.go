package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"streaming_video_service/internal/chat/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 測試 SendMessageUseCase.Execute
func TestSendMessageUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	roomID := uuid.New().String()
	senderID := uuid.New().String()
	content := "Hello, world!"
	today := time.Now().Format("2006-01-02")

	mockRoomRepo := new(MockRoomRepository)
	mockMsgRepo := new(MockMessageRepository)
	mockPubSub := new(MockRedisPubSub)

	// 模擬房間存在
	mockRoom := &domain.ChatRoom{
		ID:      roomID,
		Members: []string{senderID, "member-2"},
	}
	mockRoomRepo.On("FindByID", ctx, roomID).Return(mockRoom, nil)

	// 模擬找不到當天的 bucket，需要新建
	mockMsgRepo.On("FindBucket", ctx, roomID, today).Return(nil, errors.New("not found"))
	mockMsgRepo.On("InsertMessages", ctx, mock.Anything).Return(nil)

	// 模擬 PubSub 發送
	mockPubSub.On("Publish", "member-2", mock.Anything).Return(nil)

	uc := NewSendMessageUseCase(mockRoomRepo, mockMsgRepo, mockPubSub)
	msgID, err := uc.Execute(ctx, roomID, senderID, content)

	assert.NoError(t, err)
	assert.NotEmpty(t, msgID)

	mockRoomRepo.AssertExpectations(t)
	mockMsgRepo.AssertExpectations(t)
	mockPubSub.AssertExpectations(t)
}

// 測試 MarkRead
func TestSendMessageUseCase_MarkRead(t *testing.T) {
	ctx := context.Background()
	roomID := uuid.New().String()
	messageID := uuid.New().String()
	userID := uuid.New().String()
	today := time.Now().Format("2006-01-02")

	mockMsgRepo := new(MockMessageRepository)

	// 模擬 bucket 內有該訊息
	mockBucket := &domain.MessageBucket{
		RoomID: roomID,
		Date:   today,
		Messages: []domain.ChatMessage{
			{ID: messageID, SenderID: "someone", Content: "Test message", ReadBy: []string{}},
		},
	}

	mockMsgRepo.On("FindBucket", ctx, roomID, today).Return(mockBucket, nil)
	mockMsgRepo.On("UpdateBucket", ctx, mock.Anything).Return(nil)

	uc := &SendMessageUseCase{msgRepo: mockMsgRepo}
	err := uc.MarkRead(ctx, roomID, messageID, userID)

	assert.NoError(t, err)
	mockMsgRepo.AssertExpectations(t)
}

// 測試 GetCountUnreadMessages
func TestSendMessageUseCase_GetCountUnreadMessages(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New().String()

	mockMsgRepo := new(MockMessageRepository)

	mockUnreadInfo := []domain.RoomUnreadInfo{
		{RoomID: "room-1", UnreadCount: 5},
		{RoomID: "room-2", UnreadCount: 2},
	}

	mockMsgRepo.On("CountUnreadMessagesByRoom", ctx, userID).Return(mockUnreadInfo, nil)

	uc := &SendMessageUseCase{msgRepo: mockMsgRepo}
	result, err := uc.GetCountUnreadMessages(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, mockUnreadInfo, result)

	mockMsgRepo.AssertExpectations(t)
}
