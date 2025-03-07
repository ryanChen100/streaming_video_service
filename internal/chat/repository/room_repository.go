package repository

import (
	"context"
	"streaming_video_service/internal/chat/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// RoomRepository definition chat room
type RoomRepository interface {
	CreateRoom(ctx context.Context, room *domain.ChatRoom) error
	FindByID(ctx context.Context, roomID string) (*domain.ChatRoom, error)
	UpdateRoom(ctx context.Context, room *domain.ChatRoom) error
	FindOnePrivateRoom(ctx context.Context, userA, userB string) (*domain.ChatRoom, error)
}

type chatRepository struct {
	roomsColl *mongo.Collection
}

// NewMongoChatRepository create new mongo chat
func NewMongoChatRepository(db *mongo.Database) RoomRepository {
	return &chatRepository{
		roomsColl: db.Collection(string(domain.Group)),
		// invitationsColl: db.Collection("invitations"),
	}
}

// CreateRoom create room
func (r *chatRepository) CreateRoom(ctx context.Context, room *domain.ChatRoom) error {
	_, err := r.roomsColl.InsertOne(ctx, room)
	return err
}

// FindByID find room by id
func (r *chatRepository) FindByID(ctx context.Context, roomID string) (*domain.ChatRoom, error) {
	var room domain.ChatRoom
	err := r.roomsColl.FindOne(ctx, bson.M{"_id": roomID}).Decode(&room)
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// UpdateRoom update room info
func (r *chatRepository) UpdateRoom(ctx context.Context, room *domain.ChatRoom) error {
	filter := bson.M{"_id": room.ID}
	update := bson.M{"$set": room}
	_, err := r.roomsColl.UpdateOne(ctx, filter, update)
	return err
}

// FindOnePrivateRoom find private room
func (r *chatRepository) FindOnePrivateRoom(ctx context.Context, userA, userB string) (*domain.ChatRoom, error) {
	filter := bson.M{
		"room_type": domain.ChatRoomTypePrivate,
		"members": bson.M{
			"$all": []string{userA, userB},
		},
	}
	var room domain.ChatRoom
	err := r.roomsColl.FindOne(ctx, filter).Decode(&room)
	if err != nil {
		return nil, err
	}
	return &room, nil
}
