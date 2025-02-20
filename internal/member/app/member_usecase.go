	package app

	import (
		"context"
		"errors"
		"fmt"
		"time"

		"streaming_video_service/internal/member/domain"
		"streaming_video_service/internal/member/repository"
		"streaming_video_service/pkg/config"
		"streaming_video_service/pkg/database"
		"streaming_video_service/pkg/encrypt"
		"streaming_video_service/pkg/logger"
		token "streaming_video_service/pkg/token"

		"github.com/google/uuid"
		"go.uber.org/zap"
	)

	// MemberUseCase 這裡封裝了對外提供的應用服務
	type MemberUseCase interface {
		Register(ctx context.Context, email, password string) error
		FindMember(ctx context.Context, param *domain.MemberQuery) (*domain.Member, error)
		Login(ctx context.Context, email, password string) (string, error)
		Logout(ctx context.Context, token string) error
		ForceLogout(ctx context.Context, memberID string) error
		CheckSessionTimeout(ctx context.Context, token string) (bool, error)
		ReconnectSession(ctx context.Context, token string) error
	}

	type memberUseCase struct {
		memberRepo repository.MemberRepository
		sessionTTL time.Duration
		redisRepo  database.RedisRepository[domain.MemberSession]
	}

	// NewMemberUseCase 建立一個新的 UserUseCase
	func NewMemberUseCase(MemberRepo repository.MemberRepository,
		sessionTTL time.Duration,
		redisRepo database.RedisRepository[domain.MemberSession],
	) MemberUseCase {
		return &memberUseCase{
			memberRepo: MemberRepo,
			sessionTTL: sessionTTL,
			redisRepo:  redisRepo,
		}
	}

	// Register
	func (m *memberUseCase) Register(ctx context.Context, email, password string) error {
		// 檢查 email 是否已存在
		if _, err := m.memberRepo.FindByMember(ctx, &domain.MemberQuery{Email: &email}); err == nil {
			return errors.New("email already exists")
		}

		pw, err := encrypt.HashPassword(password)
		if err != nil {
			logger.Log.Errorf("password err :", err)
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
	func (m *memberUseCase) Login(ctx context.Context, email, password string) (string, error) {
		// 取得使用者
		member, err := m.memberRepo.FindByMember(ctx, &domain.MemberQuery{Email: &email})
		if err != nil {
			logger.Log.Error("email can't find!!!")
			return "", errors.New("user not found")
		}

		if err = member.IsPasswordMatch(password); err != nil {
			logger.Log.Error("password can't match!!!")
			return "", err
		}

		member.Status = domain.MemberStatusOnLine

		token, err := token.GenerateJWT(member.MemberID, string(token.RoleMember), config.EnvConfig.MemberService)
		now := time.Now()
		session := domain.MemberSession{
			Token:        token,
			MemberID:     member.MemberID,
			CreatedAt:    now,
			LastActivity: now,
			ExpiredAt:    now.Add(m.sessionTTL),
		}

		m.redisRepo.Set(context.Background(), member.MemberID, session, m.sessionTTL)

		if err := m.memberRepo.UpdateMemberStatus(ctx, member); err != nil {
			return "", err
		}

		return token, nil
	}

	// Logout
	func (m *memberUseCase) Logout(ctx context.Context, t string) error {
		// 取得使用者
		tokenInfo, err := token.ParseJWT(t)
		if err != nil {
			logger.Log.Error("Logout err :", zap.String("err", err.Error()))
			return err
		}
		logger.Log.Debug("logout", zap.String("member token info", fmt.Sprintf("%v", tokenInfo)))

		m.redisRepo.Del(context.Background(), tokenInfo.MemberID)

		if err := m.memberRepo.UpdateMemberStatus(ctx, &domain.Member{
			MemberID: tokenInfo.MemberID,
			Status:   domain.MemberStatusOffLine,
		}); err != nil {
			return err
		}
		return nil
	}

	// Force Logout
	// 假設我們直接把該 userID 下所有 session 都清除
	func (m *memberUseCase) ForceLogout(ctx context.Context, memberID string) error {
		m.redisRepo.Del(context.Background(), memberID)

		if err := m.memberRepo.UpdateMemberStatus(ctx, &domain.Member{
			MemberID: memberID,
			Status:   domain.MemberStatusOffLine,
		}); err != nil {
			return err
		}
		return nil
	}

	// Check Session Timeout
	func (m *memberUseCase) CheckSessionTimeout(ctx context.Context, t string) (bool, error) {
		// 取得使用者
		tokenInfo, err := token.ParseJWT(t)
		if err != nil {
			logger.Log.Error("Logout err :", zap.String("err", err.Error()))
			return true, err
		}
		logger.Log.Debug("CheckSessionTimeout", zap.String("member token info", fmt.Sprintf("%v", tokenInfo)))

		ttl, err := m.redisRepo.GetTTL(context.Background(), tokenInfo.MemberID)
		if err != nil {
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
		tokenInfo, err := token.ParseJWT(t)
		if err != nil {
			logger.Log.Error("Logout err :", zap.String("err", err.Error()))
			return err
		}
		logger.Log.Debug("ReconnectSession", zap.String("member token info", fmt.Sprintf("%v", tokenInfo)))

		m.redisRepo.ExtendTTL(context.Background(), tokenInfo.MemberID, m.sessionTTL)

		return nil
	}
