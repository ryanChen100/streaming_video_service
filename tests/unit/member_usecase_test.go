package unit

import (
	"context"
	"testing"
	"time"

	"streaming_video_service/internal/member/app"
	"streaming_video_service/internal/member/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// === 以下為假的 mock repository，用來做 TDD ===
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) CreateUser(ctx context.Context, user *domain.Member) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *mockUserRepo) FindByMember(ctx context.Context, memberQuery *domain.MemberQuery) (*domain.Member, error) {
	args := m.Called(ctx, memberQuery)
	return args.Get(0).(*domain.Member), args.Error(1)
}

func (m *mockUserRepo) UpdateMemberStatus(ctx context.Context, user *domain.Member) error {
	args := m.Called(ctx, user)
	return args.Error(1)
}

type mockSessionRepo struct {
	mock.Mock
}

func (m *mockSessionRepo) CreateSession(ctx context.Context, s *domain.MemberSession) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}
func (m *mockSessionRepo) FindSession(ctx context.Context, token string) (*domain.MemberSession, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(*domain.MemberSession), args.Error(1)
}
func (m *mockSessionRepo) UpdateSessionLastActivity(ctx context.Context, token string, lastActivity time.Time) error {
	args := m.Called(ctx, token, lastActivity)
	return args.Error(0)
}
func (m *mockSessionRepo) ExpireSession(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

type mockRedisRepo struct {
	mock.Mock
}

func (m *mockRedisRepo) Set(ctx context.Context, key string, ms domain.MemberSession, ttl time.Duration) error {
	args := m.Called(ctx, key, ms, ttl)
	return args.Error(0)
}

func (m *mockRedisRepo) Get(ctx context.Context, key string) (domain.MemberSession, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(domain.MemberSession), args.Error(1)
}

func (m *mockRedisRepo) Del(ctx context.Context, memberID string) error {
	args := m.Called(ctx, memberID)
	return args.Error(0)
}

func (m *mockRedisRepo) ExtendTTL(ctx context.Context, memberID string, ttl time.Duration) error {
	args := m.Called(ctx, memberID, ttl)
	return args.Error(0)
}

func (m *mockRedisRepo) GetTTL(ctx context.Context, memberID string) (int, error) {
	args := m.Called(ctx, memberID)
	return 0, args.Error(0)
}

type mockRedisUnmarshal struct {
	mock.Mock
}

// === 測試 Login ===
func TestUserUseCase_Login(t *testing.T) {
	ctx := context.Background()

	userRepo := new(mockUserRepo)
	// sessionRepo := new(mockSessionRepo)
	redisRepo := new(mockRedisRepo)
	usecase := app.NewMemberUseCase(userRepo, 30*time.Minute, redisRepo)

	// 模擬 userRepo.FindByEmail
	userRepo.On("FindByEmail", ctx, "user@example.com").
		Return(&domain.Member{ID: 1, Email: "user@example.com", Password: "pass1234"}, nil)

	// 模擬 sessionRepo.CreateSession
	// sessionRepo.On("CreateSession", mock.Anything, mock.Anything).
	// 	Return(nil)

	// 測試正確密碼
	token, err := usecase.Login(ctx, "user@example.com", "pass1234")
	assert.NotEmpty(t, token)
	assert.NoError(t, err)

	// 測試錯誤密碼
	_, err = usecase.Login(ctx, "user@example.com", "wrongpass")
	assert.Error(t, err)
	assert.Equal(t, "invalid credentials", err.Error())
}

// === 測試 CheckSessionTimeout ===
func TestUserUseCase_CheckSessionTimeout(t *testing.T) {
	ctx := context.Background()

	userRepo := new(mockUserRepo)
	sessionRepo := new(mockSessionRepo)
	redisRepo := new(mockRedisRepo)
	usecase := app.NewMemberUseCase(userRepo, 30*time.Minute, redisRepo)

	// 模擬已過期 session
	expiredSession := &domain.MemberSession{
		Token:     "token123",
		MemberID:  "1",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiredAt: time.Now().Add(-1 * time.Hour),
	}

	sessionRepo.On("FindSession", ctx, "token123").Return(expiredSession, nil)

	expired, err := usecase.CheckSessionTimeout(ctx, "token123")
	assert.NoError(t, err)
	assert.True(t, expired)
}
