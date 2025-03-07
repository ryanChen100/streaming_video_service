package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"streaming_video_service/internal/member/domain"
	"streaming_video_service/internal/member/repository"
	"streaming_video_service/pkg/database"
	errprocess "streaming_video_service/pkg/err"
	"streaming_video_service/pkg/logger"
	token "streaming_video_service/pkg/token"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MemberUseCase 這裡封裝了對外提供的應用服務
type MemberUseCase interface {
	Register(ctx context.Context, email, password string) error
	FindMember(ctx context.Context, param *domain.MemberQuery) (*domain.Member, error)
	Login(ctx context.Context, email, password string, now time.Time) (string, error)
	Logout(ctx context.Context, token string) error
	ForceLogout(ctx context.Context, memberID string) error
	CheckSessionTimeout(ctx context.Context, token string) (bool, error)
	ReconnectSession(ctx context.Context, token string) error
}

type memberUseCase struct {
	memberRepo       repository.MemberRepository
	sessionTTL       time.Duration
	redisRepo        database.RedisRepository[domain.MemberSession]
	hashPasswordFunc func(string) (string, error)
}

// NewMemberUseCase 建立一個新的 UserUseCase
func NewMemberUseCase(MemberRepo repository.MemberRepository,
	sessionTTL time.Duration,
	redisRepo database.RedisRepository[domain.MemberSession],
	hashFunc func(string) (string, error), // 接受 hash 函數
) MemberUseCase {
	return &memberUseCase{
		memberRepo:       MemberRepo,
		sessionTTL:       sessionTTL,
		redisRepo:        redisRepo,
		hashPasswordFunc: hashFunc,
	}
}

// Register
func (m *memberUseCase) Register(ctx context.Context, email, password string) error {
	// 檢查 email 是否已存在
	_, err := m.memberRepo.FindByMember(ctx, &domain.MemberQuery{Email: &email})
	if err == nil {
		return errors.New("email already exists")
	}

	pw, err := m.hashPasswordFunc(password)
	if err != nil {
		logger.Log.Errorf("password err :", err)
		return errors.New("hash password error")
	}

	// 建立新使用者
	user := domain.Member{
		MemberID: uuid.New().String(),
		Email:    email,
		Password: pw,
	}

	logger.Log.Info(fmt.Sprintf("usecase Register : %v", user))

	if err := m.memberRepo.CreateUser(ctx, &user); err != nil {
		return err
	}

	return nil
}

// FindByEmail 用mail來尋找使用者
func (m *memberUseCase) FindMember(ctx context.Context, param *domain.MemberQuery) (*domain.Member, error) {
	return m.memberRepo.FindByMember(ctx, param)
}

// Login
func (m *memberUseCase) Login(ctx context.Context, email, password string, now time.Time) (string, error) {
	// 取得使用者
	member, err := m.memberRepo.FindByMember(ctx, &domain.MemberQuery{Email: &email})
	if err != nil {
		errMsg := fmt.Sprintf("email[%s] can't find!!!", email)
		return "", errprocess.Set(errMsg)
	}

	if err = member.IsPasswordMatch(password); err != nil {
		errMsg := fmt.Sprintf("email[%s] password can't match!!!", email)
		return "", errprocess.Set(errMsg)
	}

	member.Status = domain.MemberStatusOnLine

	token, err := token.GenerateJWTWrapper(member.MemberID, string(token.RoleMember))
	if err != nil {
		errMsg := fmt.Sprintf("email[%s] can't GenerateJWT!!!", email)
		return "", errprocess.Set(errMsg)
	}

	session := domain.MemberSession{
		Token:        token,
		MemberID:     member.MemberID,
		CreatedAt:    now,
		LastActivity: now,
		ExpiredAt:    now.Add(m.sessionTTL),
	}

	err = m.redisRepo.Set(ctx, member.MemberID, session, m.sessionTTL)
	if err != nil {
		errMsg := fmt.Sprintf("email[%s] can't save to redis !!!", email)
		return "", errprocess.Set(errMsg)
	}

	if err := m.memberRepo.UpdateMemberStatus(ctx, member); err != nil {
		errMsg := fmt.Sprintf("email[%s] can't UpdateMemberStatus error :%v !!!", email, err)
		return "", errprocess.Set(errMsg)
	}

	return token, nil
}

// Logout
func (m *memberUseCase) Logout(ctx context.Context, t string) error {
	// 取得使用者
	tokenInfo, err := token.ParseJWTFunc(t)
	if err != nil {
		logger.Log.Error("Logout token.ParseJWTFunc err :", zap.String("err", err.Error()))
		return err
	}
	logger.Log.Debug("logout", zap.String("member token info", fmt.Sprintf("%v", tokenInfo)))

	if err := m.redisRepo.Del(context.Background(), tokenInfo.MemberID); err != nil {
		logger.Log.Errorf("Logout redisRepo.Del err : ", err)
		return err
	}

	if err := m.memberRepo.UpdateMemberStatus(ctx, &domain.Member{
		MemberID: tokenInfo.MemberID,
		Status:   domain.MemberStatusOffLine,
	}); err != nil {
		logger.Log.Errorf("Logout memberRepo.UpdateMemberStatus err : ", err)
		return err
	}
	return nil
}

// Force Logout
// 假設我們直接把該 userID 下所有 session 都清除
func (m *memberUseCase) ForceLogout(ctx context.Context, memberID string) error {
	if err := m.redisRepo.Del(context.Background(), memberID); err != nil {
		logger.Log.Error("ForceLogout redisRepo.Del err :", zap.String("err", err.Error()))
		return err
	}

	if err := m.memberRepo.UpdateMemberStatus(ctx, &domain.Member{
		MemberID: memberID,
		Status:   domain.MemberStatusOffLine,
	}); err != nil {
		logger.Log.Error("ForceLogout UpdateMemberStatus err :", zap.String("err", err.Error()))
		return err
	}
	return nil
}

// Check Session Timeout
func (m *memberUseCase) CheckSessionTimeout(ctx context.Context, t string) (bool, error) {
	// 取得使用者
	tokenInfo, err := token.ParseJWTFunc(t)
	if err != nil {
		logger.Log.Error("CheckSessionTimeout token.ParseJWTFunc err :", zap.String("err", err.Error()))
		return true, err
	}
	logger.Log.Debug("CheckSessionTimeout", zap.String("member token info", fmt.Sprintf("%v", tokenInfo)))

	ttl, err := m.redisRepo.GetTTL(context.Background(), tokenInfo.MemberID)
	if err != nil {
		if err.Error() == redis.Nil.Error() {
			return true, nil
		}
		logger.Log.Error("CheckSessionTimeout redisRepo.GetTTL err :", zap.String("err", err.Error()))
		return true, err
	}

	if ttl > 0 {
		return false, nil
	}
	return true, nil
}

// 5. Disconnected reconnection
// 當使用者重新連線，更新 last activity 並延長？或維持原 session 到期時間？
// 視需求而定，以下範例只是單純更新 lastActivity
func (m *memberUseCase) ReconnectSession(ctx context.Context, t string) error {
	// 取得使用者
	tokenInfo, err := token.ParseJWTFunc(t)
	if err != nil {
		logger.Log.Error("ReconnectSession token.ParseJWTFunc err :", zap.String("err", err.Error()))
		return err
	}
	logger.Log.Debug("ReconnectSession", zap.String("member token info", fmt.Sprintf("%v", tokenInfo)))

	if err := m.redisRepo.ExtendTTL(context.Background(), tokenInfo.MemberID, m.sessionTTL); err != nil {
		logger.Log.Error("ReconnectSession redisRepo.ExtendTTL err :", zap.String("err", err.Error()))
		return err
	}
	return nil
}
