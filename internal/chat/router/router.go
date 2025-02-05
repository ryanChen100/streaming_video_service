package router

import (
	"context"

	"streaming_video_service/internal/chat/app"
	"streaming_video_service/pkg/middlewares"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// RegisterRoutes 注册用户相关的路由
func RegisterRoutes(r *fiber.App, chatWebsocket *app.ChatWebsocketHandler) {
	r.Use(middlewares.JWTMiddleware())

	r.Get("/ws", websocket.New(func(c *websocket.Conn) {
		// 這裡可以建立一個「執行個體」，將 UseCase 等注入
		chatWebsocket.HandleConnection(context.Background(), c)
	}))

	// 假設也提供 REST API
	// appFiber.Post("/rooms", func(ctx *fiber.Ctx) error { ... })

}
