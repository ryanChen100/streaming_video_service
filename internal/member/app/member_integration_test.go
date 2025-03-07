package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"streaming_video_service/internal/member/domain"
	"streaming_video_service/internal/member/repository"
	"streaming_video_service/pkg/config"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/encrypt"
	"streaming_video_service/pkg/logger"
	memberpb "streaming_video_service/pkg/proto/member"
	testtool "streaming_video_service/pkg/test_tool"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// **測試用的容器**
var postgresContainer testcontainers.Container
var redisContainer testcontainers.Container

// **Handler**
var memberHandler *MemberGRPCServer

func TestMain(m *testing.M) {
	logger.SetNewNop()
	ctx := context.Background()
	var err error

	// **啟動 PostgreSQL**
	postgresContainer, postgresHost, postgresPort, err := testtool.SetupContainer(ctx, testcontainers.ContainerRequest{
		Image: "postgres:latest",
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp"),
	})
	if err != nil {
		log.Fatalf("❌ Failed to start PostgreSQL container: %v", err)
	}
	fmt.Printf("✅ PostgreSQL running at %s:%s\n", postgresHost, postgresPort)

	// **啟動 Redis**
	redisContainer, redisHost, redisPort, err := testtool.SetupContainer(ctx, testcontainers.ContainerRequest{
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	})
	if err != nil {
		log.Fatalf("❌ Failed to start Redis container: %v", err)
	}
	fmt.Printf("✅ Redis running at %s:%s\n", redisHost, redisPort)

	// **設定環境變數**
	os.Setenv("DATABASE_URL", fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", postgresHost, postgresPort))
	os.Setenv("REDIS_URL", fmt.Sprintf("%s:%s", redisHost, redisPort)) // **普通 Redis**

	fmt.Printf("🔹 DATABASE_URL=%s\n", os.Getenv("DATABASE_URL"))
	fmt.Printf("🔹 REDIS_URL=%s\n", os.Getenv("REDIS_URL"))

	// **執行 Migrations**
	migrationsPath, err := config.GetPath("Makefile/migrations", 5)
	if err != nil {
		log.Fatalf("get migrations path Error : %v", err)
	}
	fmt.Printf("🔹 migrations path = %s\n", migrationsPath)
	cmd := exec.Command("migrate", "-database", os.Getenv("DATABASE_URL"), "-path", migrationsPath, "up")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// **等待 Redis 確保已經準備好**
	time.Sleep(5 * time.Second)

	// **初始化資料庫**
	db, err := database.NewDatabaseConnection(database.Connection{
		ConnectStr:    os.Getenv("DATABASE_URL"),
		RetryCount:    5,
		RetryInterval: 5,
	})
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}

	// **初始化 Redis**
	redisClient, err := database.NewRedisRepository[domain.MemberSession]("", os.Getenv("REDIS_URL"), []string{}, 0)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	fmt.Println("✅ Connected to Redis successfully!")

	// **初始化 Repository**
	memberRepo := repository.NewMemberRepository(db)

	// **初始化 UseCase**
	memberUsecase := NewMemberUseCase(memberRepo, time.Hour, redisClient, encrypt.HashPassword)

	// **初始化 Handler**
	memberHandler = new(MemberGRPCServer)
	memberHandler.Usecase = memberUsecase

	// **執行測試**
	code := m.Run()

	// **停止測試容器**
	_ = postgresContainer.Terminate(ctx)
	_ = redisContainer.Terminate(ctx)

	os.Exit(code)
}

// 預設member資料
// id = 1
// member_id = 550e8400-e29b-41d4-a716-446655440000
// email = test@example.com
// password = !Password123
// status = offline

var (
	// 預設member資料
	defaultID       = 1
	defaultMemberID = "550e8400-e29b-41d4-a716-446655440000"
	defaultToken    = "test-token-123"
	defaultEmail    = "test@example.com"
	defaultPassword = "!Password123"
	defaultStatus   = 0

	email     = "testIntegration@integration.com"
	pw        = "!Integration123"
	pwInvalid = "pw123"
)

// **測試會員註冊**
func TestMemberRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("Email 已存在", func(t *testing.T) {
		req := &memberpb.RegisterReq{
			Email:    defaultEmail,
			Password: defaultPassword,
		}

		resp, err := memberHandler.Register(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "email already exists", err.Error())
		assert.Equal(t, resp, &memberpb.RegisterRes{
			Success: false,
			Message: "email already exists",
		})
		fmt.Println("✅ Register Response: Email 已存在")
	})

	t.Run("密碼加密失敗", func(t *testing.T) {
		req := &memberpb.RegisterReq{
			Email:    email,
			Password: pwInvalid,
		}

		resp, err := memberHandler.Register(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, "hash password error", err.Error())
		assert.Equal(t, resp, &memberpb.RegisterRes{
			Success: false,
			Message: "hash password error",
		})
		fmt.Println("✅ Register Response: 密碼加密失敗")
	})
	t.Run("註冊成功", func(t *testing.T) {
		req := &memberpb.RegisterReq{
			Email:    email,
			Password: pw,
		}

		resp, err := memberHandler.Register(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "create success", resp.Message)
		fmt.Println("✅ Register Response:", resp.Message)
	})
}

// **測試取得會員**
func TestFindMember(t *testing.T) {
	ctx := context.Background()

	t.Run("找不到會員", func(t *testing.T) {
		req := &memberpb.FindByMemberReq{
			Param: &memberpb.FindMemberParam{
				Id:       int64(defaultID),
				MemberId: defaultMemberID,
				Email:    email,
			},
		}

		_, err := memberHandler.FindMember(ctx, req)
		fmt.Println("err", err)
		assert.Error(t, err)
		fmt.Println("✅ findMember Response: 找不到會員")
	})

	t.Run("找到會員", func(t *testing.T) {
		req := &memberpb.FindByMemberReq{
			Param: &memberpb.FindMemberParam{
				Id:       int64(defaultID),
				MemberId: defaultMemberID,
				Email:    defaultEmail,
			},
		}

		resp, err := memberHandler.FindMember(ctx, req)

		assert.NoError(t, err)
		fmt.Println("✅ findMember Response:", resp.Message)
	})

}

// **測試會員登入**
func TestMemberLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("找不到會員", func(t *testing.T) {
		req := &memberpb.LoginReq{
			Email:    email + "unFind",
			Password: pw,
		}

		resp, err := memberHandler.Login(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("email[%s] can't find!!!", email+"unFind"), err.Error())
		assert.Empty(t, resp.Token)
		fmt.Println("✅ Login Response: 找不到會員")
	})

	t.Run("密碼錯誤", func(t *testing.T) {
		req := &memberpb.LoginReq{
			Email:    defaultEmail,
			Password: pwInvalid,
		}

		resp, err := memberHandler.Login(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, fmt.Sprintf("email[%s] password can't match!!!", defaultEmail), err.Error())
		assert.Empty(t, resp.Token)
		fmt.Println("✅ Login Response: 密碼錯誤")
	})

	t.Run("成功登入", func(t *testing.T) {
		req := &memberpb.LoginReq{
			Email:    defaultEmail,
			Password: defaultPassword,
		}

		resp, err := memberHandler.Login(ctx, req)

		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		fmt.Println("✅ Login Response:", resp.Message)
	})

}

// **測試會員登出**
func TestMemberLogout(t *testing.T) {
	ctx := context.Background()

	t.Run("無效 Token", func(t *testing.T) {
		req := &memberpb.LogoutReq{Token: "invalid_token"}
		resp, err := memberHandler.Logout(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, false, resp.Success)
		fmt.Println("✅ Logout Response: 無效 Token")
	})

	t.Run("成功登出", func(t *testing.T) {
		// 1️⃣ 先登入取得有效 Token
		loginReq := &memberpb.LoginReq{Email: defaultEmail, Password: defaultPassword}
		loginResp, err := memberHandler.Login(ctx, loginReq)
		assert.NoError(t, err)

		// 2️⃣ 使用該 Token 來登出
		req := &memberpb.LogoutReq{Token: loginResp.Token}
		resp, err := memberHandler.Logout(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "logout success", resp.Message)
		fmt.Println("✅ Logout Response: 成功登出")
	})
}

// **測試強制登出**
func TestForceLogout(t *testing.T) {
	ctx := context.Background()

	t.Run("會員不存在", func(t *testing.T) {
		req := &memberpb.ForceLogoutReq{MemberId: "non-existent-id"}
		resp, err := memberHandler.ForceLogout(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, false, resp.Success)
		fmt.Println("✅ ForceLogout Response: 會員不存在")
	})

	t.Run("成功強制登出", func(t *testing.T) {
		// 1️⃣ 先登入取得有效 Token
		loginReq := &memberpb.LoginReq{Email: defaultEmail, Password: defaultPassword}
		_, err := memberHandler.Login(ctx, loginReq)
		assert.NoError(t, err)

		// 2️⃣ 使用會員 ID 來強制登出
		req := &memberpb.ForceLogoutReq{MemberId: defaultMemberID}
		resp, err := memberHandler.ForceLogout(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "logout success", resp.Message)
		fmt.Println("✅ ForceLogout Response: 成功強制登出")
	})
}

// **測試檢查 Session 過期**
func TestCheckSessionTimeout(t *testing.T) {
	ctx := context.Background()

	t.Run("token 錯誤", func(t *testing.T) {
		req := &memberpb.CheckSessionTimeoutReq{Token: "expired_token"}
		resp, err := memberHandler.CheckSessionTimeout(ctx, req)
		assert.Error(t, err)
		assert.True(t, resp.Expire)
		fmt.Println("✅ CheckSessionTimeout Response: token 錯誤")
	})
	t.Run("Session 過期", func(t *testing.T) {
		loginReq := &memberpb.LoginReq{Email: defaultEmail, Password: defaultPassword}
		loginResp, err := memberHandler.Login(ctx, loginReq)
		assert.NoError(t, err)

		req := &memberpb.LogoutReq{Token: loginResp.Token}
		_, err = memberHandler.Logout(ctx, req)
		assert.NoError(t, err)

		checkreq := &memberpb.CheckSessionTimeoutReq{Token: "expired_token"}
		checkresp, err := memberHandler.CheckSessionTimeout(ctx, checkreq)
		assert.Error(t, err)
		assert.True(t, checkresp.Expire)

		fmt.Println("✅ CheckSessionTimeout Response: Session 過期")
	})

	t.Run("Session 有效", func(t *testing.T) {
		// 1️⃣ 先登入取得有效 Token
		loginReq := &memberpb.LoginReq{Email: defaultEmail, Password: defaultPassword}
		loginResp, err := memberHandler.Login(ctx, loginReq)
		assert.NoError(t, err)

		// 2️⃣ 使用該 Token 來檢查 Session
		req := &memberpb.CheckSessionTimeoutReq{Token: loginResp.Token}
		resp, err := memberHandler.CheckSessionTimeout(ctx, req)

		assert.NoError(t, err)
		assert.False(t, resp.Expire)
		fmt.Println("✅ CheckSessionTimeout Response: Session 有效")
	})
}

// **測試重新連線**
func TestReconnectSession(t *testing.T) {
	ctx := context.Background()

	t.Run("Token 無效", func(t *testing.T) {
		req := &memberpb.ReconnectSessionReq{Token: "invalid_token"}
		resp, err := memberHandler.ReconnectSession(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, false, resp.Success)
		fmt.Println("✅ ReconnectSession Response: Token 無效")
	})

	t.Run("成功重連", func(t *testing.T) {
		// 1️⃣ 先登入取得有效 Token
		loginReq := &memberpb.LoginReq{Email: defaultEmail, Password: defaultPassword}
		loginResp, err := memberHandler.Login(ctx, loginReq)
		assert.NoError(t, err)

		// 2️⃣ 使用該 Token 來重新連線
		req := &memberpb.ReconnectSessionReq{Token: loginResp.Token}
		resp, err := memberHandler.ReconnectSession(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, "logout success", resp.Message)
		fmt.Println("✅ ReconnectSession Response: 成功重連")
	})
}
