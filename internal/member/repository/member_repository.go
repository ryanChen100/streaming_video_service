package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"streaming_video_service/internal/member/domain"
)

// MemberRepository definition get Member info
type MemberRepository interface {
	CreateUser(ctx context.Context, user *domain.Member) error
	UpdateMemberStatus(ctx context.Context, user *domain.Member) error
	FindByMember(ctx context.Context, memberQuery *domain.MemberQuery) (*domain.Member, error)
	// 其他 CRUD ...
}

type memberRepository struct {
	db *pgxpool.Pool
}

// NewMemberRepository create a MemberRepository
func NewMemberRepository(db *pgxpool.Pool) MemberRepository {
	return &memberRepository{db: db}
}

func (r *memberRepository) CreateUser(ctx context.Context, member *domain.Member) error {
	_, err := r.db.Exec(ctx, "INSERT INTO member(member_id,email, password) VALUES ($1, $2, $3)", member.MemberID, member.Email, member.Password)
	return err
}

func (r *memberRepository) UpdateMemberStatus(ctx context.Context, member *domain.Member) error {
	_, err := r.db.Exec(ctx, "UPDATE member SET status = $1 WHERE member_id = $2", member.Status, member.MemberID)
	return err
}

func (r *memberRepository) FindByMember(ctx context.Context, memberQuery *domain.MemberQuery) (*domain.Member, error) {
	queryStr := "SELECT id, member_id, email, password FROM member WHERE 1=1"
	params := []interface{}{}
	paramCount := 1

	if memberQuery.Email != nil {
		queryStr += fmt.Sprintf(" AND email = $%d", paramCount)
		params = append(params, *memberQuery.Email)
		paramCount++
	}
	if memberQuery.MemberID != nil {
		queryStr += fmt.Sprintf(" AND member_id = $%d", paramCount)
		params = append(params, *memberQuery.MemberID)
		paramCount++
	}
	if memberQuery.ID != nil {
		queryStr += fmt.Sprintf(" AND id = $%d", paramCount)
		params = append(params, *memberQuery.ID)
		paramCount++
	}

	row := r.db.QueryRow(ctx, queryStr, params...)
	var member domain.Member
	err := row.Scan(&member.ID, &member.MemberID, &member.Email, &member.Password)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("no member found with given criteria")
		}
		return nil, err
	}

	return &member, nil
}
