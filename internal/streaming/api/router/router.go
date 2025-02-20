package router

import (
	"streaming_video_service/internal/streaming/api/handlers"

	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes 注册用户相关的路由
func RegisterRoutes(app *fiber.App, videoHandler *handlers.VideoHandler) {
	app.Post("/upload", videoHandler.UploadVideo)
	app.Get("/video/:id", videoHandler.GetVideo)
	app.Get("/video/hls/:id/index.m3u8", videoHandler.GetIndexM3U8)
	app.Get("/video/hls/:id/:segment", videoHandler.GetHlsSegment)
	app.Get("/video/:id", videoHandler.GetVideo)
	app.Get("/search", videoHandler.Search)
	app.Get("/recommend", videoHandler.GetRecommendations)
}
