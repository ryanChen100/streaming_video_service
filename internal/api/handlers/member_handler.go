package handlers

import (
	"context"
	"fmt"
	"streaming_video_service/pkg/logger"
	"streaming_video_service/pkg/middlewares"
	memberpb "streaming_video_service/pkg/proto/member"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// MemberHandler 处理用户相关的 HTTP 请求
type MemberHandler struct {
	MemberClient memberpb.MemberServiceClient
}

// NewMemberHandler 创建新的 UserHandler
func NewMemberHandler(memberClient memberpb.MemberServiceClient) *MemberHandler {
	return &MemberHandler{
		MemberClient: memberClient,
	}
}

// Register 注册新用户
// @Summary 注册新用户
// @Description 处理用户注册请求
// @Tags Members
// @Accept json
// @Produce json
// @Param request body memberpb.RegisterReq true "注册请求"
// @Success 200 {object} memberpb.RegisterRes "注册成功"
// @Failure 400 {object} string "请求错误"
// @Failure 500 {object} string "服务器错误"
// @Router /member/register [post]
func (h *MemberHandler) Register(c *fiber.Ctx) error {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	logger.Log.Debug("Register request", zap.String("email", req.Email), zap.String("password", req.Password))
	logger.Log.Info("Register request", zap.String("email", req.Email), zap.String("password", req.Password))

	resp, err := h.MemberClient.Register(context.Background(), &memberpb.RegisterReq{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil || !resp.Success {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": resp.GetMessage()})
	}

	logger.Log.Info(fmt.Sprintf("MemberClient.Register %v", resp))
	return c.JSON(fiber.Map{"message": "register success"})
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户通过邮箱和密码登录
// @Tags Members
// @Accept json
// @Produce json
// @Param request body memberpb.LoginReq true "用户登录信息"
// @Success 200 {object} string "登录成功"
// @Failure 400 {object} string "请求错误"
// @Failure 401 {object} string "登录失败"
// @Router /member/login [post]
func (h *MemberHandler) Login(c *fiber.Ctx) error {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	logger.Log.Debug("Login", zap.String("Email", req.Email), zap.String("Password", req.Password))

	resp, err := h.MemberClient.Login(context.Background(), &memberpb.LoginReq{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil || !resp.Success {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": resp.GetMessage()})
	}

	logger.Log.Info(fmt.Sprintf("MemberClient.Login %v", resp))
	return c.JSON(fiber.Map{"token": resp.GetToken(), "message": "login success"})
}

// Logout 用户登出
// @Summary 用户登出
// @Description 注销用户会话
// @Tags Members
// @Accept json
// @Produce json
// @Param auth query string false "用户登出信息"
// @Success 200 {object} string "注销成功"
// @Failure 400 {object} string "请求错误"
// @Failure 500 {object} string "服务器错误"
// @Router /member/logout [post]
func (h *MemberHandler) Logout(c *fiber.Ctx) error {

	token, ok := c.Locals(middlewares.TokenMemberID).(string)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fmt.Sprintf("c.Locals(%s) is nill", middlewares.TokenMemberID)})
	}

	resp, err := h.MemberClient.Logout(context.Background(), &memberpb.LogoutReq{
		Token: token,
	})

	if err != nil || !resp.Success {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": resp.GetMessage()})
	}

	logger.Log.Info(fmt.Sprintf("MemberClient.Logout %v", resp))
	return c.JSON(fiber.Map{"message": "logout success"})
}

// FindByEmail 查找用户信息
// @Summary 查找用户信息
// @Description 根据邮箱查找用户信息
// @Tags Members
// @Accept json
// @Produce json
// @Param email query string true "用户邮箱"
// @Success 200 {object} string "用户信息"
// @Failure 400 {object} string "请求错误"
// @Failure 404 {object} string "未找到用户"
// @Router /member/find [get]
func (h *MemberHandler) FindByEmail(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
	}

	resp, err := h.MemberClient.FindMember(context.Background(), &memberpb.FindByMemberReq{
		Param: &memberpb.FindMemberParam{
			Email: email,
		}})

	if err != nil || !resp.Success {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": resp.GetMessage()})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":       resp.Info.Id,
			"email":    resp.Info.Email,
			"password": resp.Info.Password, // 避免泄漏密码，建议移除
		},
	})
}
