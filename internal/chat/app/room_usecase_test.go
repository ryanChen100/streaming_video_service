package app

import (
	"context"
	"testing"

	"streaming_video_service/internal/chat/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 測試 ExecuteRoom
func TestRoomUseCase_ExecuteRoom(t *testing.T) {
	ctx := context.Background()
	members := []string{"user-1", "user-2"}
	roomType := domain.ChatRoomTypePrivate

	mockRoomRepo := new(MockRoomRepository)
	mockInvRepo := new(MockInvitationRepository)

	mockRoomRepo.On("CreateRoom", ctx, mock.Anything).Return(nil)

	uc := NewRoomUseCase(mockInvRepo, mockRoomRepo)
	createdRoomID, err := uc.ExecuteRoom(ctx, roomType, "Test Room", members, domain.JoinModeOpen, "", true, false)

	assert.NoError(t, err)
	assert.NotEmpty(t, createdRoomID)

	mockRoomRepo.AssertExpectations(t)
}

// 測試 JoinRoom
func TestRoomUseCase_JoinRoom(t *testing.T) {
	ctx := context.Background()
	roomID := uuid.New().String()
	userID := uuid.New().String()

	mockRoomRepo := new(MockRoomRepository)
	mockInvRepo := new(MockInvitationRepository)

	room := &domain.ChatRoom{
		ID:       roomID,
		RoomType: domain.ChatRoomTypeGroup,
		JoinMode: domain.JoinModeOpen,
		Members:  []string{"user-1"},
	}

	mockRoomRepo.On("FindByID", ctx, roomID).Return(room, nil)
	mockRoomRepo.On("UpdateRoom", ctx, mock.Anything).Return(nil)

	uc := NewRoomUseCase(mockInvRepo, mockRoomRepo)
	err := uc.JoinRoom(ctx, roomID, userID, "")

	assert.NoError(t, err)
	mockRoomRepo.AssertExpectations(t)
}

// 測試 ExecuteInvite
func TestRoomUseCase_ExecuteInvite(t *testing.T) {
	ctx := context.Background()
	inviterID := uuid.New().String()
	inviteeID := uuid.New().String()

	mockRoomRepo := new(MockRoomRepository)
	mockInvRepo := new(MockInvitationRepository)

	mockInvRepo.On("FindInvitationStatus", ctx, inviterID, inviteeID).Return(nil, nil)
	mockInvRepo.On("CreateInvitation", ctx, mock.Anything).Return(nil)

	uc := NewRoomUseCase(mockInvRepo, mockRoomRepo)
	invID, err := uc.ExecuteInvite(ctx, inviterID, inviteeID)

	assert.NoError(t, err)
	assert.NotEmpty(t, invID)

	mockInvRepo.AssertExpectations(t)
}

// 測試 ExecuteAccept
func TestRoomUseCase_ExecuteAccept(t *testing.T) {
	ctx := context.Background()
	inviterID := uuid.New().String()
	inviteeID := uuid.New().String()

	mockRoomRepo := new(MockRoomRepository)
	mockInvRepo := new(MockInvitationRepository)

	inv := &domain.PrivateChatInvitation{
		ID:        uuid.New().String(),
		InviterID: inviterID,
		InviteeID: inviteeID,
		Status:    domain.InvitationPending,
	}

	mockInvRepo.On("FindInvitationByID", ctx, inviterID, inviteeID).Return(inv, nil)
	mockInvRepo.On("UpdateInvitationStatus", ctx, inviterID, inviteeID, domain.InvitationAccepted).Return(nil)
	mockRoomRepo.On("FindOnePrivateRoom", ctx, inviterID, inviteeID).Return(nil, nil)
	mockRoomRepo.On("CreateRoom", ctx, mock.Anything).Return(nil)

	uc := NewRoomUseCase(mockInvRepo, mockRoomRepo)
	roomID, err := uc.ExecuteAccept(ctx, inviterID, inviteeID)

	assert.NoError(t, err)
	assert.NotEmpty(t, roomID)

	mockInvRepo.AssertExpectations(t)
	mockRoomRepo.AssertExpectations(t)
}

// 測試 ExitRoom
func TestRoomUseCase_ExitRoom(t *testing.T) {
	ctx := context.Background()
	roomID := uuid.New().String()
	userID := uuid.New().String()

	mockRoomRepo := new(MockRoomRepository)
	mockInvRepo := new(MockInvitationRepository)

	room := &domain.ChatRoom{
		ID:      roomID,
		Members: []string{"user-1", userID},
	}

	mockRoomRepo.On("FindByID", ctx, roomID).Return(room, nil)
	mockRoomRepo.On("UpdateRoom", ctx, mock.Anything).Return(nil)

	uc := NewRoomUseCase(mockInvRepo, mockRoomRepo)
	err := uc.ExitRoom(ctx, roomID, userID)

	assert.NoError(t, err)
	mockRoomRepo.AssertExpectations(t)
}
