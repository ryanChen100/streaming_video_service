package app

import (
	"context"
	"errors"
	"fmt"
	"streaming_video_service/internal/member/domain"
	"streaming_video_service/pkg/encrypt"
	"streaming_video_service/pkg/logger"
	token "streaming_video_service/pkg/token"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMember struct {
	mock.Mock
	domain.Member
}

func (m *MockMember) IsPasswordMatch(input string) error {
	args := m.Called(input)
	return args.Error(0)
}

// MockMemberRepo Mock MemberRepo
type MockMemberRepo struct {
	mock.Mock
}

func (m *MockMemberRepo) CreateUser(ctx context.Context, user *domain.Member) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *MockMemberRepo) UpdateMemberStatus(ctx context.Context, user *domain.Member) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}
func (m *MockMemberRepo) FindByMember(ctx context.Context, memberQuery *domain.MemberQuery) (*domain.Member, error) {
	args := m.Called(ctx, memberQuery)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Member), args.Error(1)
	}
	return nil, args.Error(1)
}

// MockRedisRepo 針對 MemberSession 的 Mock
type MockRedisRepo struct {
	mock.Mock
}

// Set 模擬 Redis Set 操作
func (m *MockRedisRepo) Set(ctx context.Context, key string, value domain.MemberSession, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

// Get 模擬 Redis Get 操作
func (m *MockRedisRepo) Get(ctx context.Context, key string) (domain.MemberSession, error) {
	args := m.Called(ctx, key)
	if args.Get(0) != nil {
		return args.Get(0).(domain.MemberSession), args.Error(1)
	}
	return domain.MemberSession{}, args.Error(1)
}

// Del 模擬 Redis Del 操作
func (m *MockRedisRepo) Del(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

// ExtendTTL 模擬 Redis ExtendTTL 操作
func (m *MockRedisRepo) ExtendTTL(ctx context.Context, key string, ttl time.Duration) error {
	args := m.Called(ctx, key, ttl)
	return args.Error(0)
}

// GetTTL 模擬 Redis GetTTL 操作
func (m *MockRedisRepo) GetTTL(ctx context.Context, key string) (int, error) {
	args := m.Called(ctx, key)
	return args.Int(0), args.Error(1)
}

var hashPasswordFunc = encrypt.HashPassword

func TestMemberUseCase_Register(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	password := "!!Securepassword111"

	mockRepo := new(MockMemberRepo)
	mockRedis := new(MockRedisRepo)

	logger.SetNewNop()

	// **情境 1: 註冊成功**
	t.Run("成功註冊", func(t *testing.T) {
		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).Return(nil, errors.New("not found")).Once()
		mockRepo.On("CreateUser", ctx, mock.Anything).Return(nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.Register(ctx, email, password)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	// **情境 2: Email 已存在**
	t.Run("Email 已存在", func(t *testing.T) {
		existingUser := &domain.Member{
			ID:       1,
			MemberID: "AAA",
			Email:    email,
			Password: password,
			Status:   domain.MemberStatusOffLine,
		}

		// 讓 `FindByMember` 回傳一個「已存在的使用者」，且 `err == nil`
		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).
			Return(existingUser, nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)

		// 3️⃣ 執行 Register
		err := uc.Register(ctx, email, password)

		// 4️⃣ 確保 Register 回傳 "email already exists" 的錯誤
		assert.Error(t, err)
		assert.Equal(t, "email already exists", err.Error())
		mockRepo.AssertExpectations(t)
	})

	// **情境 3: 密碼加密失敗**
	t.Run("密碼加密失敗", func(t *testing.T) {
		mockHashPassword := func(password string) (string, error) {
			return "", errors.New("hash password error")
		}

		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).Return(nil, errors.New("not found")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, mockHashPassword)
		err := uc.Register(ctx, email, password)

		assert.Error(t, err)
		assert.Equal(t, "hash password error", err.Error())
		mockRepo.AssertExpectations(t)
	})

	// **情境 4: 建立用戶失敗**
	t.Run("建立用戶失敗", func(t *testing.T) {
		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).Return(nil, errors.New("not found")).Once()
		mockRepo.On("CreateUser", ctx, mock.Anything).Return(errors.New("db error")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.Register(ctx, email, password)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
		mockRepo.AssertExpectations(t)
	})
}
func TestMemberUseCase_FindMember(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	password := "!!Securepassword111"

	mockRepo := new(MockMemberRepo)
	mockRedis := new(MockRedisRepo)

	logger.SetNewNop()

	// **情境 1: 找到會員**
	t.Run("找到會員", func(t *testing.T) {
		existingUser := &domain.Member{
			ID:       1,
			MemberID: "AAA",
			Email:    email,
			Password: password,
			Status:   domain.MemberStatusOffLine,
		}

		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).Return(existingUser, nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		member, err := uc.FindMember(ctx, &domain.MemberQuery{Email: &email})

		assert.NoError(t, err)
		assert.Equal(t, member, existingUser)
		mockRepo.AssertExpectations(t)
	})

	// **情境 2: 找不到會員**
	t.Run("找不到會員", func(t *testing.T) {
		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).Return(nil, errors.New("no member found with given criteria")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		_, err := uc.FindMember(ctx, &domain.MemberQuery{Email: &email})

		assert.Error(t, err)
		assert.Equal(t, "no member found with given criteria", err.Error())

		mockRepo.AssertExpectations(t)
	})
}
func TestMemberUseCase_Login(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	password := "!!Securepassword111"
	hashedPassword, _ := encrypt.HashPassword(password) // 先模擬加密密碼

	mockRepo := new(MockMemberRepo)
	mockRedis := new(MockRedisRepo)

	logger.SetNewNop() // 禁用測試時的 log 輸出

	// **情境 1: 成功登入**
	t.Run("成功登入", func(t *testing.T) {
		existingUser := &domain.Member{
			MemberID: "AAA",
			Email:    email,
			Password: hashedPassword,
			Status:   domain.MemberStatusOffLine,
		}

		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).
			Return(existingUser, nil).Once()

		mockRepo.On("UpdateMemberStatus", ctx, existingUser).
			Return(nil).Once()

		mockRedis.On("Set", ctx, existingUser.MemberID, mock.Anything, mock.Anything).
			Return(nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		token, err := uc.Login(ctx, email, password, time.Now())

		assert.NoError(t, err)
		assert.NotEmpty(t, token) // 確保 token 不為空
		mockRepo.AssertExpectations(t)
		mockRedis.AssertExpectations(t)
	})

	// **情境 2: 使用者不存在**
	t.Run("使用者不存在", func(t *testing.T) {
		errMsg := fmt.Sprintf("email[%s] can't find!!!", email)
		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).
			Return(nil, errors.New(errMsg)).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		token, err := uc.Login(ctx, email, password, time.Now())

		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("email[%s] can't find!!!", email), err.Error())
		assert.Empty(t, token) // 確保 token 為空
		mockRepo.AssertExpectations(t)
	})

	// **情境 3: 密碼錯誤**
	t.Run("密碼錯誤", func(t *testing.T) {
		mockRepo := new(MockMemberRepo)
		mockRedis := new(MockRedisRepo)

		// **使用 MockMember**
		mockUser := new(MockMember)
		mockUser.Member = domain.Member{
			MemberID: "AAA",
			Email:    email,
			Password: hashedPassword,
			Status:   domain.MemberStatusOffLine,
		}

		// **讓 `FindByMember` 回傳 `*MockMember`**
		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).
			Return(&mockUser.Member, nil).Once() //  確保返回 `*domain.Member`

		// **Mock `IsPasswordMatch` 方法**
		mockUser.On("IsPasswordMatch", "wrong_password").
			Return(errors.New("password does not match")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		token, err := uc.Login(ctx, email, "wrong_password", time.Now())

		assert.Error(t, err)
		assert.Equal(t, "password does not match", err.Error())
		assert.Empty(t, token)

		mockRepo.AssertExpectations(t)
		// mockUser.AssertExpectations(t) // **檢查 Mock 方法是否被正確調用**
	})

	// **情境 4: 更新使用者狀態失敗**
	t.Run("更新使用者狀態失敗", func(t *testing.T) {
		existingUser := &domain.Member{
			MemberID: "AAA",
			Email:    email,
			Password: hashedPassword,
			Status:   domain.MemberStatusOffLine,
		}

		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).
			Return(existingUser, nil).Once()

		mockRepo.On("UpdateMemberStatus", ctx, existingUser).
			Return(errors.New("failed to update status")).Once()

		mockRedis.On("Set", ctx, existingUser.MemberID, mock.Anything, mock.Anything).
			Return(nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		token, err := uc.Login(ctx, email, password, time.Now())

		assert.Error(t, err)
		assert.Equal(t, "failed to update status", err.Error())
		assert.Empty(t, token)
		mockRepo.AssertExpectations(t)
		mockRedis.AssertExpectations(t)
	})

	// **情境 5: JWT 生成失敗**
	t.Run("JWT 生成失敗", func(t *testing.T) {
		mockRepo := new(MockMemberRepo)
		mockRedis := new(MockRedisRepo)

		existingUser := &domain.Member{
			MemberID: "AAA",
			Email:    email,
			Password: hashedPassword,
			Status:   domain.MemberStatusOffLine,
		}

		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).
			Return(existingUser, nil).Once()

		// **1️.先備份原始的 `GenerateJWTFunc`**
		originalGenerateJWT := token.GenerateJWTFunc
		defer func() { token.GenerateJWTFunc = originalGenerateJWT }() // **確保測試結束後恢復**

		// **2️.Mock `GenerateJWTFunc` 並補上 `issuer` 參數**
		errMsg := fmt.Sprintf("email[%s] can't GenerateJWT!!!", email)
		token.GenerateJWTFunc = func(existingUser, role, issuer string) (string, error) {
			return "", errors.New(errMsg)
		}

		existingUser.Status = domain.MemberStatusOnLine
		mockRepo.On("UpdateMemberStatus", ctx, existingUser).
			Return(nil).Once()

		mockRedis.On("Set", ctx, existingUser.MemberID, mock.Anything, mock.Anything).
			Return(nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		tok, err := uc.Login(ctx, email, password, time.Now())

		// **3️.確保 `Login` 返回 JWT 生成錯誤** error無法預期
		assert.Error(t, err)
		assert.Equal(t, errMsg, err.Error())
		assert.Empty(t, tok)

		// mockRepo.AssertExpectations(t)
		// mockRedis.AssertExpectations(t)
	})

	// **情境 6: Redis 存 session 失敗**
	t.Run("Redis 存 session 失敗", func(t *testing.T) {
		mockRepo := new(MockMemberRepo)
		mockRedis := new(MockRedisRepo)

		// **使用 MockMember**
		mockUser := new(MockMember)
		mockUser.Member = domain.Member{
			MemberID: "AAA",
			Email:    email,
			Password: hashedPassword,
			Status:   domain.MemberStatusOffLine,
		}

		//  Mock `FindByMember`
		// **讓 `FindByMember` 回傳 `*MockMember`**
		mockRepo.On("FindByMember", ctx, &domain.MemberQuery{Email: &email}).
			Return(&mockUser.Member, nil).Once() // ✅ 確保返回 `*domain.Member`

		// **Mock `IsPasswordMatch` 方法**
		mockUser.On("IsPasswordMatch", "wrong_password").
			Return(errors.New("password does not match")).Once()

		mockUser.Member.Status = domain.MemberStatusOnLine
		//  Mock `UpdateMemberStatus`

		// **1️.先備份原始的 `GenerateJWTFunc`**
		originalGenerateJWT := token.GenerateJWTFunc
		defer func() { token.GenerateJWTFunc = originalGenerateJWT }() // **確保測試結束後恢復**

		// **2️.Mock `GenerateJWTFunc` 並補上 `issuer` 參數**
		token.GenerateJWTFunc = func(existingUser, role, issuer string) (string, error) {
			return "token", nil
		}

		mockRepo.On("UpdateMemberStatus", ctx, mockUser.Member).
			Return(nil).Once()
		now := time.Now()
		session := domain.MemberSession{
			Token:        "token",
			MemberID:     mockUser.Member.MemberID,
			CreatedAt:    now,
			LastActivity: now,
			ExpiredAt:    now.Add(time.Hour),
		}

		//  Mock `Redis.Set` 讓它返回 `redis error`
		errMsg := fmt.Sprintf("email[%s] can't save to redis !!!", email)
		mockRedis.On("Set", ctx, mockUser.Member.MemberID, session, time.Hour).
			Return(errors.New(errMsg)).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		token, err := uc.Login(ctx, email, password, now)

		assert.Error(t, err)
		assert.Equal(t, errMsg, err.Error())
		assert.Empty(t, token)

		// mockRepo.AssertExpectations(t)
		mockRedis.AssertExpectations(t)
	})

}

func TestMemberUseCase_Logout(t *testing.T) {
	ctx := context.Background()
	tokenStr := "mockToken"
	memberID := "AAA"

	mockRepo := new(MockMemberRepo)
	mockRedis := new(MockRedisRepo)

	logger.SetNewNop() // 停用 Logger 避免測試時輸出

	// ** 1.解析 Token 失敗**
	t.Run("解析 Token 失敗", func(t *testing.T) {
		// 備份原始 `ParseJWTFunc`
		originalParseJWTFunc := token.ParseJWTFunc
		defer func() { token.ParseJWTFunc = originalParseJWTFunc }() // 確保測試結束後恢復

		// Mock `ParseJWTFunc` 讓它回傳錯誤
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return nil, errors.New("invalid token")
		}

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.Logout(ctx, tokenStr)

		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})

	// **2️.Redis 刪除 session 失敗**
	t.Run("Redis 刪除 session 失敗", func(t *testing.T) {
		// Mock `ParseJWTFunc` 返回正常 Token
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return &token.Claims{MemberID: memberID}, nil
		}

		// Mock `Del` 讓 Redis 回傳錯誤
		mockRedis.On("Del", ctx, memberID).Return(errors.New("redis error")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.Logout(ctx, tokenStr)

		assert.Error(t, err)
		assert.Equal(t, "redis error", err.Error())

		mockRedis.AssertExpectations(t)
	})

	// **3️.更新使用者狀態失敗**
	t.Run("更新使用者狀態失敗", func(t *testing.T) {
		//  Mock `ParseJWTFunc`
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return &token.Claims{MemberID: memberID}, nil
		}

		//  Mock `Del` 成功
		mockRedis.On("Del", ctx, memberID).Return(nil).Once()

		//  Mock `UpdateMemberStatus` 失敗
		mockRepo.On("UpdateMemberStatus", ctx, &domain.Member{
			MemberID: memberID,
			Status:   domain.MemberStatusOffLine,
		}).Return(errors.New("db error")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.Logout(ctx, tokenStr)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())

		mockRedis.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	// **4️.成功登出**
	t.Run("成功登出", func(t *testing.T) {
		//  Mock `ParseJWTFunc`
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return &token.Claims{MemberID: memberID}, nil
		}

		//  Mock `Del` 成功
		mockRedis.On("Del", ctx, memberID).Return(nil).Once()

		//  Mock `UpdateMemberStatus` 成功
		mockRepo.On("UpdateMemberStatus", ctx, &domain.Member{
			MemberID: memberID,
			Status:   domain.MemberStatusOffLine,
		}).Return(nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.Logout(ctx, tokenStr)

		assert.NoError(t, err)

		mockRedis.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}

func TestMemberUseCase_ForceLogout(t *testing.T) {
	ctx := context.Background()
	memberID := "AAA"

	mockRepo := new(MockMemberRepo)
	mockRedis := new(MockRedisRepo)

	logger.SetNewNop() // 停用 Logger 避免測試時輸出

	// **1️.Redis 刪除 session 失敗**
	t.Run("Redis 刪除 session 失敗", func(t *testing.T) {
		mockRedis.On("Del", ctx, memberID).
			Return(errors.New("redis error")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.ForceLogout(ctx, memberID)

		assert.Error(t, err)
		assert.Equal(t, "redis error", err.Error())

		mockRedis.AssertExpectations(t)
	})

	// **2️.更新使用者狀態失敗**
	t.Run("更新使用者狀態失敗", func(t *testing.T) {
		//  Mock `Del` 成功
		mockRedis.On("Del", ctx, memberID).
			Return(nil).Once()

		//  Mock `UpdateMemberStatus` 失敗
		mockRepo.On("UpdateMemberStatus", ctx, &domain.Member{
			MemberID: memberID,
			Status:   domain.MemberStatusOffLine,
		}).Return(errors.New("db error")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.ForceLogout(ctx, memberID)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())

		mockRedis.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})

	// **3️.成功登出**
	t.Run("成功登出", func(t *testing.T) {
		//  Mock `Del` 成功
		mockRedis.On("Del", ctx, memberID).
			Return(nil).Once()

		//  Mock `UpdateMemberStatus` 成功
		mockRepo.On("UpdateMemberStatus", ctx, &domain.Member{
			MemberID: memberID,
			Status:   domain.MemberStatusOffLine,
		}).Return(nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.ForceLogout(ctx, memberID)

		assert.NoError(t, err)

		mockRedis.AssertExpectations(t)
		mockRepo.AssertExpectations(t)
	})
}
func TestMemberUseCase_CheckSessionTimeout(t *testing.T) {
	ctx := context.Background()
	tokenStr := "mockToken"
	memberID := "AAA"

	mockRepo := new(MockMemberRepo)
	mockRedis := new(MockRedisRepo)

	logger.SetNewNop() // 停用 Logger 避免測試時輸出

	// **1️.Mock `ParseJWTFunc` 失敗**
	t.Run("解析 Token 失敗", func(t *testing.T) {
		//  備份原始 `ParseJWTFunc`
		originalParseJWTFunc := token.ParseJWTFunc
		defer func() { token.ParseJWTFunc = originalParseJWTFunc }() // 確保測試結束後恢復

		//  Mock `ParseJWTFunc` 回傳錯誤
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return nil, errors.New("invalid token")
		}

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		timedOut, err := uc.CheckSessionTimeout(ctx, tokenStr)

		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
		assert.True(t, timedOut) // 預期 Session 逾時
	})

	// **2️.Mock `Redis.GetTTL` 失敗**
	t.Run("Redis 查詢 TTL 失敗", func(t *testing.T) {
		//  Mock `ParseJWTFunc` 返回正常 Token
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return &token.Claims{MemberID: memberID}, nil
		}

		//  Mock `GetTTL` 讓 Redis 回傳錯誤
		mockRedis.On("GetTTL", ctx, memberID).Return(0, errors.New("redis error")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		timedOut, err := uc.CheckSessionTimeout(ctx, tokenStr)

		assert.Error(t, err)
		assert.Equal(t, "redis error", err.Error())
		assert.True(t, timedOut) // 預期 Session 逾時

		mockRedis.AssertExpectations(t)
	})

	// **3️.Mock `Redis.GetTTL` 返回 > 0（Session 未過期）**
	t.Run("Session 尚未過期", func(t *testing.T) {
		//  Mock `ParseJWTFunc`
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return &token.Claims{MemberID: memberID}, nil
		}

		//  Mock `GetTTL` 讓 Redis 回傳 `60` 秒（代表 Session 未過期）
		mockRedis.On("GetTTL", ctx, memberID).Return(60, nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		timedOut, err := uc.CheckSessionTimeout(ctx, tokenStr)

		assert.NoError(t, err)
		assert.False(t, timedOut) // 預期 Session 未過期

		mockRedis.AssertExpectations(t)
	})

	// **4️.Mock `Redis.GetTTL` 返回 0（Session 已過期）**
	t.Run("Session 已過期", func(t *testing.T) {
		//  Mock `ParseJWTFunc`
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return &token.Claims{MemberID: memberID}, nil
		}

		//  Mock `GetTTL` 讓 Redis 回傳 `0`（代表 Session 已過期）
		mockRedis.On("GetTTL", ctx, memberID).Return(0, nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		timedOut, err := uc.CheckSessionTimeout(ctx, tokenStr)

		assert.NoError(t, err)
		assert.True(t, timedOut) // 預期 Session 過期

		mockRedis.AssertExpectations(t)
	})
}

func TestMemberUseCase_ReconnectSession(t *testing.T) {
	ctx := context.Background()
	tokenStr := "mockToken"
	memberID := "AAA"

	mockRepo := new(MockMemberRepo)
	mockRedis := new(MockRedisRepo)

	logger.SetNewNop() // 停用 Logger 避免測試時輸出

	// **1️.Mock `ParseJWTFunc` 失敗**
	t.Run("解析 Token 失敗", func(t *testing.T) {
		//  備份原始 `ParseJWTFunc`
		originalParseJWTFunc := token.ParseJWTFunc
		defer func() { token.ParseJWTFunc = originalParseJWTFunc }() // 確保測試結束後恢復

		//  Mock `ParseJWTFunc` 讓它回傳錯誤
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return nil, errors.New("invalid token")
		}

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.ReconnectSession(ctx, tokenStr)

		assert.Error(t, err)
		assert.Equal(t, "invalid token", err.Error())
	})

	// **2️.Mock `Redis.ExtendTTL` 失敗**
	t.Run("Redis 延長 TTL 失敗", func(t *testing.T) {
		//  Mock `ParseJWTFunc` 返回正常 Token
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return &token.Claims{MemberID: memberID}, nil
		}

		//  Mock `ExtendTTL` 讓 Redis 回傳錯誤
		mockRedis.On("ExtendTTL", ctx, memberID, time.Hour).Return(errors.New("redis error")).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.ReconnectSession(ctx, tokenStr)

		assert.Error(t, err)
		assert.Equal(t, "redis error", err.Error())

		mockRedis.AssertExpectations(t)
	})

	// **3️.Mock `Redis.ExtendTTL` 成功**
	t.Run("成功延長 Session", func(t *testing.T) {
		//  Mock `ParseJWTFunc`
		token.ParseJWTFunc = func(t string) (*token.Claims, error) {
			return &token.Claims{MemberID: memberID}, nil
		}

		//  Mock `ExtendTTL` 成功
		mockRedis.On("ExtendTTL", ctx, memberID, time.Hour).Return(nil).Once()

		uc := NewMemberUseCase(mockRepo, time.Hour, mockRedis, encrypt.HashPassword)
		err := uc.ReconnectSession(ctx, tokenStr)

		assert.NoError(t, err)

		mockRedis.AssertExpectations(t)
	})
}
