package app

import (
	"context"
	"errors"

	"time"

	"github.com/google/uuid"

	"streaming_video_service/internal/chat/domain"
	"streaming_video_service/internal/chat/repository"
)

// RoomUseCase - 用於直接建立聊天室 (群組或 1對1)
type RoomUseCase struct {
	invRepo  repository.InvitationRepository
	roomRepo repository.RoomRepository
}

// NewRoomUseCase init room ues case
func NewRoomUseCase(i repository.InvitationRepository, r repository.RoomRepository) *RoomUseCase {
	return &RoomUseCase{
		invRepo:  i,
		roomRepo: r,
	}
}

// ExecuteRoom create room
func (uc *RoomUseCase) ExecuteRoom(
	ctx context.Context,
	roomType domain.ChatRoomType,
	name string,
	members []string,
	joinMode domain.JoinMode,
	password string,
	isPrivate bool,
	isInvite bool,
) (string, error) {

	if roomType == domain.ChatRoomTypePrivate && len(members) != 2 {
		return "", errors.New("private room must have exactly 2 members")
	}

	room := &domain.ChatRoom{
		ID:        uuid.New().String(), // or let DB handle _id
		RoomType:  roomType,
		Name:      name,
		Members:   members,
		JoinMode:  joinMode,
		Password:  password,
		IsPrivate: isPrivate,
		IsInvite:  isInvite,
		CreatedAt: time.Now().Unix(),
	}

	// 若是 group，預設第一位成員當 admin
	if roomType == domain.ChatRoomTypeGroup && len(members) > 0 {
		room.Admins = []string{members[0]}
	}

	if err := uc.roomRepo.CreateRoom(ctx, room); err != nil {
		return "", err
	}
	return room.ID, nil
}

// JoinRoom join room
func (uc *RoomUseCase) JoinRoom(ctx context.Context, roomID, userID, password string) error {
	room, err := uc.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return errors.New("room not found")
	}

	if room.RoomType != domain.ChatRoomTypeGroup {
		return errors.New("not a group chat room")
	}

	switch room.JoinMode {
	case domain.JoinModeOpen:
		room.Members = appendIfNotExists(room.Members, userID)

	case domain.JoinModePassword:
		if password == "" || password != room.Password {
			return errors.New("invalid password")
		}
		room.Members = appendIfNotExists(room.Members, userID)

	case domain.JoinModeApprove:
		return errors.New("need admin approval")
	}

	return uc.roomRepo.UpdateRoom(ctx, room)
}

func appendIfNotExists(list []string, val string) []string {
	for _, v := range list {
		if v == val {
			return list
		}
	}
	return append(list, val)
}

// -----------------------------------------------------------
// InvitePrivateChatUseCase - 用於發起 1對1 邀請 (pending 狀態)
// -----------------------------------------------------------

// ExecuteInvite create invite
func (uc *RoomUseCase) ExecuteInvite(ctx context.Context, inviterID, inviteeID string) (string, error) {
	// 檢查是否已存在 pending
	pending, _ := uc.invRepo.FindInvitationStatus(ctx, inviterID, inviteeID)
	if pending != nil {
		return "", errors.New("already invited, wait for acceptance")
	}

	inv := &domain.PrivateChatInvitation{
		ID:        uuid.New().String(),
		InviterID: inviterID,
		InviteeID: inviteeID,
		Status:    domain.InvitationPending,
		CreatedAt: time.Now().Unix(),
	}

	if err := uc.invRepo.CreateInvitation(ctx, inv); err != nil {
		return "", err
	}
	return inv.ID, nil
}

// ExecuteAccept accept  chat
func (uc *RoomUseCase) ExecuteAccept(ctx context.Context, InviterID, userID string) (string, error) {
	inv, err := uc.invRepo.FindInvitationByID(ctx, InviterID, userID)
	if err != nil {
		return "", err
	}
	if inv == nil {
		return "", errors.New("invitation not found")
	}
	if inv.InviteeID != userID {
		return "", errors.New("not the correct invitee")
	}
	if inv.Status != domain.InvitationPending {
		return "", errors.New("invitation not pending")
	}

	// 更新為 accepted
	if err := uc.invRepo.UpdateInvitationStatus(ctx, InviterID, userID, domain.InvitationAccepted); err != nil {
		return "", err
	}

	// 檢查是否已存在(A,B)的 1對1房
	existRoom, _ := uc.roomRepo.FindOnePrivateRoom(ctx, inv.InviterID, inv.InviteeID)
	if existRoom != nil {
		return existRoom.ID, nil
	}

	// 否則新建 1對1房
	room := &domain.ChatRoom{
		ID:        uuid.New().String(), // or new ID
		RoomType:  domain.ChatRoomTypePrivate,
		Members:   []string{inv.InviterID, inv.InviteeID},
		IsInvite:  true,
		CreatedAt: time.Now().Unix(),
	}
	err = uc.roomRepo.CreateRoom(ctx, room)
	if err != nil {
		return "", err
	}
	return room.ID, nil
}

// ExitRoom member exit room
func (uc *RoomUseCase) ExitRoom(ctx context.Context, roomID, userID string) error {
	// 1. 找到 room
	room, err := uc.roomRepo.FindByID(ctx, roomID)
	if err != nil || room == nil {
		return errors.New("room not found")
	}

	// 2. 從 room.Members 移除 userID
	newMembers := make([]string, 0, len(room.Members))
	for _, m := range room.Members {
		if m != userID {
			newMembers = append(newMembers, m)
		}
	}
	room.Members = newMembers

	// 3. 更新 DB
	err = uc.roomRepo.UpdateRoom(ctx, room)
	if err != nil {
		return err
	}
	return nil
}
