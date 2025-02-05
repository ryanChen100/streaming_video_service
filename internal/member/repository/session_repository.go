package repository

// import (
// 	"context"

// 	"streaming_video_service/internal/member/domain"
// 	"time"

// 	"github.com/jackc/pgx/v4/pgxpool"
// )

// type SessionRepository interface {
// 	CreateSession(ctx context.Context, session *domain.MemberSession) error
// 	FindSession(ctx context.Context, token string) (*domain.MemberSession, error)
// 	UpdateSessionLastActivity(ctx context.Context, token string, lastActivity time.Time) error
// 	ExpireSession(ctx context.Context, token string) error
// }

// type sessionRepository struct {
// 	db *pgxpool.Pool
// }

// func NewSessionRepository(db *pgxpool.Pool) SessionRepository {
// 	return &sessionRepository{db: db}
// }

// func (r *sessionRepository) CreateSession(ctx context.Context, s *domain.MemberSession) error {
// 	_, err := r.db.Exec(ctx, `
//       INSERT INTO user_sessions(token, user_id, created_at, last_activity, expired_at)
//       VALUES ($1, $2, $3, $4, $5)
//     `,
// 		s.Token, s.MemberID, s.CreatedAt, s.LastActivity, s.ExpiredAt,
// 	)
// 	return err
// }

// func (r *sessionRepository) FindSession(ctx context.Context, token string) (*domain.MemberSession, error) {
// 	row := r.db.QueryRow(ctx, `
//       SELECT token, user_id, created_at, last_activity, expired_at
//       FROM user_sessions
//       WHERE token = $1
//     `, token)
// 	var s domain.MemberSession
// 	if err := row.Scan(
// 		&s.Token, &s.MemberID, &s.CreatedAt, &s.LastActivity, &s.ExpiredAt,
// 	); err != nil {
// 		return nil, err
// 	}
// 	return &s, nil
// }

// func (r *sessionRepository) UpdateSessionLastActivity(ctx context.Context, token string, lastActivity time.Time) error {
// 	_, err := r.db.Exec(ctx, `
//       UPDATE user_sessions
//       SET last_activity = $1
//       WHERE token = $2
//     `, lastActivity, token)
// 	return err
// }

// func (r *sessionRepository) ExpireSession(ctx context.Context, token string) error {
// 	_, err := r.db.Exec(ctx, `
//       DELETE FROM user_sessions
//       WHERE token = $1
//     `, token)
// 	return err
// }
