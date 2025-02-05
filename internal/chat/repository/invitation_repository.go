package repository

import (
	"context"
	"streaming_video_service/internal/chat/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// InvitationRepository definition invitation room (1 on 1)
type InvitationRepository interface {
	CreateInvitation(ctx context.Context, inv *domain.PrivateChatInvitation) error
	FindInvitationByID(ctx context.Context, inviterID, userID string) (*domain.PrivateChatInvitation, error)
	FindInvitationByPending(ctx context.Context, userID string) ([]*domain.PrivateChatInvitation, error)
	UpdateInvitationStatus(ctx context.Context, inviterID, userID string, newStatus domain.InvitationStatus) error
	FindInvitationStatus(ctx context.Context, inviterID, inviteeID string) (*domain.PrivateChatInvitation, error)
}

type mongoInvitationRepository struct {
	// roomsColl *mongo.Collection
	invitationsColl *mongo.Collection
}

// NewMongoInvitationRepository create new mongo invitation
func NewMongoInvitationRepository(db *mongo.Database) InvitationRepository {
	return &mongoInvitationRepository{
		// roomsColl: db.Collection("rooms"),
		invitationsColl: db.Collection(string(domain.Invite)),
	}
}

// CreateInvitation create invitation
func (r *mongoInvitationRepository) CreateInvitation(ctx context.Context, inv *domain.PrivateChatInvitation) error {
	_, err := r.invitationsColl.InsertOne(ctx, inv)
	return err
}

// FindInvitationByID find invitation by id
func (r *mongoInvitationRepository) FindInvitationByID(ctx context.Context, inviterID, userID string) (*domain.PrivateChatInvitation, error) {
	filter := bson.M{
		"inviter_id": inviterID,
		"invitee_id": userID,
	}

	var invitation domain.PrivateChatInvitation
	err := r.invitationsColl.FindOne(ctx, filter).Decode(&invitation)
	if err != nil {
		return nil, err
	}
	return &invitation, nil
}

// FindInvitationByID find invitation by id
func (r *mongoInvitationRepository) FindInvitationByPending(ctx context.Context, userID string) ([]*domain.PrivateChatInvitation, error) {
	filter := bson.M{
		"invitee_id": userID,
		"status":     "pending",
	}

	// 使用 Find 方法取得 cursor
	cur, err := r.invitationsColl.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var invitations []*domain.PrivateChatInvitation
	// 遍歷 cursor，將每筆資料解碼後加入 slice
	for cur.Next(ctx) {
		var inv domain.PrivateChatInvitation
		if err := cur.Decode(&inv); err != nil {
			return nil, err
		}
		invitations = append(invitations, &inv)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return invitations, nil
}

// UpdateInvitationStatus update invitation status
func (r *mongoInvitationRepository) UpdateInvitationStatus(ctx context.Context, inviterID, userID string, newStatus domain.InvitationStatus) error {
	filter := bson.M{
		"inviter_id": inviterID,
		"invitee_id": userID,
	}
	update := bson.M{"$set": bson.M{"status": newStatus}}
	_, err := r.invitationsColl.UpdateOne(ctx, filter, update)
	return err
}

// FindInvitationStatus find invitation status
func (r *mongoInvitationRepository) FindInvitationStatus(ctx context.Context, inviterID, inviteeID string) (*domain.PrivateChatInvitation, error) {
	filter := bson.M{
		"inviter_id": inviterID,
		"invitee_id": inviteeID,
	}
	var inv domain.PrivateChatInvitation
	err := r.invitationsColl.FindOne(ctx, filter).Decode(&inv)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}
