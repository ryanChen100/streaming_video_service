package handlers

import (
	"fmt"
	"net/url"
	"strconv"
	"streaming_video_service/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// ConnectCheck check api connect start
// @Summary Check API Gateway status
// @Description Returns a simple confirmation message
// @Tags Shared
// @Success 200 {string} string "api gateway start!"
// @Router / [get]
func ConnectCheck(c *fiber.Ctx) error {
	return c.SendString("api gateway start!")
}

// DebugLogFlag toggle debug log flag
// @Summary Toggle Debug Log Flag
// @Description Enable or disable debug logging for a service
// @Tags Shared
// @Param service query string true "Service name"
// @Param status query bool true "Debug status"
// @Success 200 {string} string "Service debug mode updated"
// @Failure 400 {string} string "Invalid status value"
// @Router /debug [post]
func DebugLogFlag(c *fiber.Ctx) error {
	// prase payload
	query, err := url.ParseQuery(string(c.Context().QueryArgs().QueryString()))
	service := query.Get("service")
	statusStr := query.Get("status")
	logger.Log.Info("debug", zap.String("status", statusStr))
	status, err := strconv.ParseBool(statusStr)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	switch service {
	default:
		logger.Log.SetDebugMode(status)
	}
	return c.SendString(fmt.Sprintf("service[%s]: debug mode is : %t", service, status))
}
