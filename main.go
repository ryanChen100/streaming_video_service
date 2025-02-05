package main

import (
	"streaming_video_service/internal/api/router"

	"github.com/gofiber/fiber/v2"
)

// 因拆分微服務。此程式用於init swagger
// swag init output ./docs
func main() {
	// 创建 Fiber 应用
	app := fiber.New()

	// 注册路由
	router.RegisterRoutes(app, nil)

}
