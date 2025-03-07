package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"streaming_video_service/internal/chat/repository"
	"streaming_video_service/pkg/database"
	"streaming_video_service/pkg/logger"
	memberpb "streaming_video_service/pkg/proto/member"
	testtool "streaming_video_service/pkg/test_tool"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
)

// **測試用的容器**
var mongoContainer testcontainers.Container
var redisContainer testcontainers.Container
var grpcServer *grpc.Server
var chatApp *fiber.App
var chatHandler *ChatWebsocketHandler

// **TestMain 初始化測試環境**
func TestMain(m *testing.M) {
	ctx := context.Background()
	logger.SetNewNop()
	var err error

	// **啟動 MongoDB**
	mongoContainer, mongoHost, mongoPort, err := testtool.SetupContainer(ctx, testcontainers.ContainerRequest{
		Image:        "mongo:latest",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017/tcp"),
	})
	if err != nil {
		log.Fatalf("❌ Failed to start MongoDB container: %v", err)
	}
	fmt.Printf("✅ MongoDB running at %s:%s\n", mongoHost, mongoPort)

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
	os.Setenv("MONGO_URL", fmt.Sprintf("mongodb://%s:%s", mongoHost, mongoPort))
	os.Setenv("REDIS_URL", fmt.Sprintf("%s:%s", redisHost, redisPort))

	fmt.Printf("🔹 MONGO_URL=%s\n", os.Getenv("MONGO_URL"))
	fmt.Printf("🔹 REDIS_URL=%s\n", os.Getenv("REDIS_URL"))

	// **初始化 MongoDB**
	mongo, err := database.NewMongoDB(ctx, database.Connection{
		ConnectStr:    os.Getenv("MONGO_URL"),
		RetryCount:    5,
		RetryInterval: 5,
	}, "test_chat_db")
	if err != nil {
		log.Fatalf("❌ Failed to connect to MongoDB: %v", err)
	}
	defer mongo.Close(ctx)

	// **初始化 Redis**
	redisClient := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
		DB:   0,
	})

	// **啟動 Mock gRPC `member` 服務**
	grpcServer, grpcAddress := testtool.StartMockMemberGRPCServer()
	os.Setenv("MEMBER_GRPC_ADDR", grpcAddress)
	fmt.Println("✅ Mock gRPC Member Service started at", grpcAddress)

	client, err := database.CreateGRPCClient(grpcAddress)
	if err != nil {
		log.Fatalf("create member GRPC err : %v", err)
	}
	logger.Log.Info(fmt.Sprintf("grpc connect :%v", client.GetState()))
	memberClient := memberpb.NewMemberServiceClient(client)

	// **初始化 Repository**
	roomRepo := repository.NewMongoChatRepository(mongo.Database)
	inviteRepo := repository.NewMongoInvitationRepository(mongo.Database)
	msgRepo := repository.NewMongoChatMessageRepository(mongo.Database)
	pub := repository.NewRedisPubSub(redisClient)

	// 6. 初始化 UseCases
	roomUC := NewRoomUseCase(inviteRepo, roomRepo)
	sendMessageUC := NewSendMessageUseCase(roomRepo, msgRepo, pub)

	// **初始化 Fiber WebSocket Server**
	chatHandler = NewChatWebsocketHandler(roomUC, sendMessageUC, &memberClient)

	chatApp = fiber.New()
	chatApp.Get("/ws", websocket.New(func(c *websocket.Conn) {
		// 這裡可以建立一個「執行個體」，將 UseCase 等注入
		chatHandler.HandleConnection(context.Background(), c)
	}))

	// **啟動 WebSocket Server**
	go func() {
		err := chatApp.Listen(":8081")
		if err != nil {
			log.Fatalf("❌ Failed to start WebSocket server: %v", err)
		}
	}()
	fmt.Println("✅ WebSocket Server started at ws://localhost:8081/ws")

	// **等待 WebSocket Server 啟動**
	time.Sleep(5 * time.Second)

	// **執行測試**
	code := m.Run()

	// **清理測試環境**
	_ = mongoContainer.Terminate(ctx)
	_ = redisContainer.Terminate(ctx)
	grpcServer.Stop()
	chatApp.Shutdown()

	os.Exit(code)
}

// ✅ 1️⃣ WebSocket 連線測試
func TestFiberWebSocketConnection(t *testing.T) {
	wsURL := "ws://127.0.0.1:8081/ws"

	// 使用 Gorilla WebSocket 連線
	dialer := gws.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket 連線失敗")
	defer conn.Close()

	fmt.Println("✅ WebSocket 連線成功!")

	// 測試發送訊息
	message := []byte(`{"type": "message", "content": "Hello, World!"}`)
	err = conn.WriteMessage(websocket.TextMessage, message)
	assert.NoError(t, err, "發送訊息失敗")

	// 測試接收訊息
	_, response, err := conn.ReadMessage()
	assert.NoError(t, err, "接收訊息失敗")
	fmt.Println("✅ 收到訊息:", string(response))
}

// ✅ 2️⃣ InvitePrivate 測試
func TestInvitePrivate(t *testing.T) {
	wsURL := "ws://127.0.0.1:8081/ws"
	dialer := gws.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket 連線失敗")
	defer conn.Close()

	// 發送邀請
	inviteReq := []byte(`{"action": "invite_private", "invitee_id": "user_456"}`)
	err = conn.WriteMessage(gws.TextMessage, inviteReq)
	assert.NoError(t, err, "邀請好友請求失敗")

	// 接收邀請回應
	_, response, err := conn.ReadMessage()
	assert.NoError(t, err, "邀請好友回應失敗")
	fmt.Println("✅ 邀請好友回應:", string(response))
}

// ✅ 2️⃣ AcceptInvite 測試
func TestAcceptInvite(t *testing.T) {
	wsURL := "ws://127.0.0.1:8081/ws"
	dialer := gws.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket 連線失敗")
	defer conn.Close()

	// 接受邀請
	acceptReq := []byte(`{"action": "accept_invite", "inviter_id": "user_123"}`)
	err = conn.WriteMessage(gws.TextMessage, acceptReq)
	assert.NoError(t, err, "接受邀請請求失敗")

	// 接收回應
	_, response, err := conn.ReadMessage()
	assert.NoError(t, err, "接受邀請回應失敗")
	fmt.Println("✅ 接受邀請回應:", string(response))
}

// ✅ 3️⃣ CreateRoom 測試
func TestCreateRoom(t *testing.T) {
	wsURL := "ws://127.0.0.1:8081/ws"
	dialer := gws.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket 連線失敗")
	defer conn.Close()

	// 創建聊天室
	createRoomReq := []byte(`{"action": "create_room", "room_type": "group", "room_name": "Test Room", "join_mode": "public"}`)
	err = conn.WriteMessage(gws.TextMessage, createRoomReq)
	assert.NoError(t, err, "建立聊天室請求失敗")

	// 接收回應
	_, response, err := conn.ReadMessage()
	assert.NoError(t, err, "建立聊天室回應失敗")
	fmt.Println("✅ 建立聊天室回應:", string(response))
}

// ✅ 4️⃣ JoinRoom 測試測試
func TestJoinRoom(t *testing.T) {
	wsURL := "ws://127.0.0.1:8081/ws"
	dialer := gws.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket 連線失敗")
	defer conn.Close()

	// 加入聊天室
	joinRoomReq := []byte(`{"action": "join_room", "room_id": "room_123", "password": ""}`)
	err = conn.WriteMessage(gws.TextMessage, joinRoomReq)
	assert.NoError(t, err, "加入聊天室請求失敗")

	// 接收回應
	_, response, err := conn.ReadMessage()
	assert.NoError(t, err, "加入聊天室回應失敗")
	fmt.Println("✅ 加入聊天室回應:", string(response))
}

// ✅ 5️⃣ SendMessage 測試
func TestSendMessage(t *testing.T) {
	wsURL := "ws://127.0.0.1:8081/ws"
	dialer := gws.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket 連線失敗")
	defer conn.Close()

	// 發送訊息
	messageReq := []byte(`{"action": "send_message", "room_id": "room_123", "content": "Hello, World!"}`)
	err = conn.WriteMessage(gws.TextMessage, messageReq)
	assert.NoError(t, err, "發送訊息請求失敗")

	// 接收回應
	_, response, err := conn.ReadMessage()
	assert.NoError(t, err, "發送訊息回應失敗")
	fmt.Println("✅ 發送訊息回應:", string(response))
}

// ✅ 6️⃣ ReadMessage 測試
func TestReadMessage(t *testing.T) {
	wsURL := "ws://127.0.0.1:8081/ws"
	dialer := gws.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket 連線失敗")
	defer conn.Close()

	// 讀取訊息
	readReq := []byte(`{"action": "read_message", "room_id": "room_123", "message_id": "msg_789"}`)
	err = conn.WriteMessage(gws.TextMessage, readReq)
	assert.NoError(t, err, "讀取訊息請求失敗")

	// 接收回應
	_, response, err := conn.ReadMessage()
	assert.NoError(t, err, "讀取訊息回應失敗")
	fmt.Println("✅ 讀取訊息回應:", string(response))
}

// ✅ 7️⃣ ReadMessage 測試
func TestGetUnreadMessages(t *testing.T) {
	wsURL := "ws://127.0.0.1:8081/ws"
	dialer := gws.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket 連線失敗")
	defer conn.Close()

	// 查詢未讀訊息
	getUnreadReq := []byte(`{"action": "get_unread"}`)
	err = conn.WriteMessage(gws.TextMessage, getUnreadReq)
	assert.NoError(t, err, "獲取未讀訊息請求失敗")

	// 接收回應
	_, response, err := conn.ReadMessage()
	assert.NoError(t, err, "獲取未讀訊息回應失敗")
	fmt.Println("✅ 未讀訊息回應:", string(response))
}
