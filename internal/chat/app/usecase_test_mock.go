package app

import (
	"context"
	"streaming_video_service/internal/chat/domain"

	"github.com/stretchr/testify/mock"
)

// MockRoomRepository Mock RoomRepository
type MockRoomRepository struct {
	mock.Mock
}

// CreateRoom moke create room
func (m *MockRoomRepository) CreateRoom(ctx context.Context, room *domain.ChatRoom) error {
	args := m.Called(ctx, room)
	return args.Error(0)
}

// FindByID moke find room by room id
func (m *MockRoomRepository) FindByID(ctx context.Context, roomID string) (*domain.ChatRoom, error) {
	args := m.Called(ctx, roomID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.ChatRoom), args.Error(1)
	}
	return nil, args.Error(1)
}

// UpdateRoom moke update room
func (m *MockRoomRepository) UpdateRoom(ctx context.Context, room *domain.ChatRoom) error {
	args := m.Called(ctx, room)
	return args.Error(0)
}

// FindOnePrivateRoom moke find one private room
func (m *MockRoomRepository) FindOnePrivateRoom(ctx context.Context, userA, userB string) (*domain.ChatRoom, error) {
	args := m.Called(ctx, userA, userB)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.ChatRoom), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockMessageRepository Mock MessageRepository
type MockMessageRepository struct {
	mock.Mock
}

// FindBucket moke find room message date by bucket
func (m *MockMessageRepository) FindBucket(ctx context.Context, roomID, date string) (*domain.MessageBucket, error) {
	args := m.Called(ctx, roomID, date)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.MessageBucket), args.Error(1)
	}
	return nil, args.Error(1)
}

// InsertMessages moke insert msg
func (m *MockMessageRepository) InsertMessages(ctx context.Context, bucket *domain.MessageBucket) error {
	args := m.Called(ctx, bucket)
	return args.Error(0)
}

// UpdateBucket moke update msg bucket
func (m *MockMessageRepository) UpdateBucket(ctx context.Context, bucket *domain.MessageBucket) error {
	args := m.Called(ctx, bucket)
	return args.Error(0)
}

// CountUnreadMessagesByRoom moke get count unread by user id
func (m *MockMessageRepository) CountUnreadMessagesByRoom(ctx context.Context, userID string) ([]domain.RoomUnreadInfo, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]domain.RoomUnreadInfo), args.Error(1)
}

// FindEarliestUnread moke find earliest unread msg
func (m *MockMessageRepository) FindEarliestUnread(ctx context.Context, roomID, userID string) (*domain.MessageBucket, error) {
	args := m.Called(ctx, roomID, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.MessageBucket), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindMessagesBefore moke find before meg
func (m *MockMessageRepository) FindMessagesBefore(ctx context.Context, roomID string, limit int64) ([]domain.ChatMessage, error) {
	args := m.Called(ctx, roomID, limit)
	if args.Get(0) != nil {
		return args.Get(0).([]domain.ChatMessage), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockRedisPubSub Mock RedisPubSub
type MockRedisPubSub struct {
	mock.Mock
}

// Publish moke publisher
func (m *MockRedisPubSub) Publish(channel string, message interface{}) error {
	args := m.Called(channel, message)
	return args.Error(0)
}

// Subscribe moke subscriber
func (m *MockRedisPubSub) Subscribe(ctx context.Context, channel string, handler func(resp domain.WSResponse)) error {
	args := m.Called(channel, handler)
	return args.Error(0)
}

// MockInvitationRepository Mock InvitationRepository
type MockInvitationRepository struct {
	mock.Mock
}

// CreateInvitation moke create invitation
func (m *MockInvitationRepository) CreateInvitation(ctx context.Context, inv *domain.PrivateChatInvitation) error {
	args := m.Called(ctx, inv)
	return args.Error(0)
}

// FindInvitationByID moke find invitation by user & inviter ID
func (m *MockInvitationRepository) FindInvitationByID(ctx context.Context, inviterID, userID string) (*domain.PrivateChatInvitation, error) {
	args := m.Called(ctx, inviterID, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.PrivateChatInvitation), args.Error(1)
	}
	return nil, args.Error(1)
}

// FindInvitationByPending moke find invitation status is pending
func (m *MockInvitationRepository) FindInvitationByPending(ctx context.Context, userID string) ([]*domain.PrivateChatInvitation, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).([]*domain.PrivateChatInvitation), args.Error(1)
	}
	return nil, args.Error(1)
}

// UpdateInvitationStatus moke update invitation status
func (m *MockInvitationRepository) UpdateInvitationStatus(ctx context.Context, inviterID, userID string, newStatus domain.InvitationStatus) error {
	args := m.Called(ctx, inviterID, userID, newStatus)
	return args.Error(0)
}

// FindInvitationStatus moke find invitation by status
func (m *MockInvitationRepository) FindInvitationStatus(ctx context.Context, inviterID, inviteeID string) (*domain.PrivateChatInvitation, error) {
	args := m.Called(ctx, inviterID, inviteeID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.PrivateChatInvitation), args.Error(1)
	}
	return nil, args.Error(1)
}
