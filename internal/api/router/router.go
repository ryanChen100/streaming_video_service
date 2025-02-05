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
func RegisterRoutes(app *fiber.App, memberHandler *handlers.MemberHandler) {
	app.Get("/swagger/*", swagger.HandlerDefault)
	app.Get("/", comm.ConnectCheck)
	app.Post("/debug", comm.DebugLogFlag)

	memberRoutes := app.Group("/member")
	memberRoutes.Post("/register", memberHandler.Register)
	memberRoutes.Post("/login", memberHandler.Login)
	memberRoutes.Get("/find", memberHandler.FindByEmail)

	memberRoutes.Use(middlewares.JWTMiddleware())
	memberRoutes.Post("/logout", memberHandler.Logout)

}
