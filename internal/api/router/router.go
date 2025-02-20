package router

import (
	"streaming_video_service/internal/api/comm"
	"streaming_video_service/internal/api/handlers"
	"streaming_video_service/pkg/middlewares"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// RegisterRoutes 注册用户相关的路由
// @title Streaming Video Service API
// @version 1.0
// @description API documentation for Streaming Video Service
// @host localhost:8080
// @BasePath /
func RegisterRoutes(app *fiber.App, memberHandler *handlers.MemberHandler, streamingHandler *handlers.StreamingHandler) {
	app.Get("/swagger/*", swagger.HandlerDefault)
	app.Get("/", comm.ConnectCheck)
	app.Post("/debug", comm.DebugLogFlag)

	memberRoutes := app.Group("/member")
	memberRoutes.Post("/register", memberHandler.Register)
	memberRoutes.Post("/login", memberHandler.Login)
	memberRoutes.Get("/find", memberHandler.FindByEmail)

	memberRoutes.Use(middlewares.JWTMiddleware())
	memberRoutes.Post("/logout", memberHandler.Logout)

	// Chat 路由：加上 JWT Middleware，並建立 WebSocket 連線
	// chatRoutes := app.Group("/chat")
	// chatRoutes.Use(middlewares.JWTMiddleware())
	// chatRoutes.Get("/ws", websocket.New(func(c *websocket.Conn) {
	// 	// 這裡直接使用已注入的 ChatWebsocketHandler 處理 WebSocket 連線
	// 	chatWebsocket.HandleConnection(context.Background(), c)
	// }))

	streamingRoutes := app.Group("/streaming")
	streamingRoutes.Use(middlewares.JWTMiddleware())
	streamingRoutes.Post("/upload", streamingHandler.UploadVideo)
	streamingRoutes.Get("/video/:video_id", streamingHandler.GetVideo)
	streamingRoutes.Get("/video/hls/:video_id/index", streamingHandler.GetIndexM3U8)
	streamingRoutes.Get("/video/hls/:video_id/:segment", streamingHandler.GetHlsSegment)
	streamingRoutes.Get("/search", streamingHandler.Search)
	streamingRoutes.Get("/recommend", streamingHandler.GetRecommendations)
}
