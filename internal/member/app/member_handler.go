package app

import (
	"context"
	"strconv"
	"streaming_video_service/internal/member/domain"
	"streaming_video_service/pkg/logger"
	memberpb "streaming_video_service/pkg/proto/member"

	"go.uber.org/zap"
)

// MemberGRPCServer 用來實作 MemberGRPCServer
type MemberGRPCServer struct {
	memberpb.UnimplementedMemberServiceServer
	Usecase MemberUseCase
}

// Register 實作 Register
func (s *MemberGRPCServer) Register(ctx context.Context, req *memberpb.RegisterReq) (*memberpb.RegisterRes, error) {
	logger.Log.Debug("Register Req", zap.String("email", req.GetEmail()), zap.String("password", req.GetPassword()))
	err := s.Usecase.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		logger.Log.Error("Register Err", zap.String("email", req.GetEmail()), zap.String("password", req.GetPassword()), zap.String("Err :", err.Error()))

		return &memberpb.RegisterRes{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &memberpb.RegisterRes{
		Success: true,
		Message: "create success",
	}, nil
}

// FindMember 實作 尋找member
func (s *MemberGRPCServer) FindMember(ctx context.Context, req *memberpb.FindByMemberReq) (*memberpb.FindByMemberRes, error) {
	logger.Log.Debug("FindByEmail :", zap.String("id", strconv.Itoa(int(req.Param.GetId()))), zap.String("member_id", req.Param.GetMemberId()), zap.String("email", req.Param.GetEmail()))
	id := req.Param.GetId()
	memberID := req.Param.GetMemberId()
	email := req.Param.GetEmail()
	member, err := s.Usecase.FindMember(ctx, &domain.MemberQuery{
		ID:       &id,
		MemberID: &memberID,
		Email:    &email,
	})
	if err != nil {
		logger.Log.Error("FindByEmail Err", zap.String("id", strconv.Itoa(int(req.Param.GetId()))), zap.String("member_id", req.Param.GetMemberId()), zap.String("email", req.Param.GetEmail()), zap.String("Err :", err.Error()))

		return &memberpb.FindByMemberRes{
			Success: false,
			Info: &memberpb.MemberInfo{
				Id:       "",
				Email:    "",
				Password: "",
			},
			Message: err.Error(),
		}, nil
	}
	return &memberpb.FindByMemberRes{
		Success: true,
		Info: &memberpb.MemberInfo{
			Id:       member.MemberID,
			Email:    member.Email,
			Password: member.Password,
		},
		Message: "create success",
	}, nil
}

// Login 實作 Login
func (s *MemberGRPCServer) Login(ctx context.Context, req *memberpb.LoginReq) (*memberpb.LoginRes, error) {
	logger.Log.Debug("Login :", zap.String("email", req.GetEmail()), zap.String("password", req.GetPassword()))
	token, err := s.Usecase.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		logger.Log.Error("Login Err", zap.String("email", req.GetPassword()), zap.String("password", req.GetPassword()), zap.String("Err :", err.Error()))
		return &memberpb.LoginRes{
			Success: false,
			Token:   "",
			Message: err.Error(),
		}, nil
	}
	return &memberpb.LoginRes{
		Success: true,
		Token:   token,
		Message: "login success",
	}, nil
}

// Logout 實作 Logout
func (s *MemberGRPCServer) Logout(ctx context.Context, req *memberpb.LogoutReq) (*memberpb.LogoutRes, error) {
	logger.Log.Info("logout", zap.String("token", req.GetToken()))
	err := s.Usecase.Logout(ctx, req.GetToken())
	if err != nil {
		return &memberpb.LogoutRes{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &memberpb.LogoutRes{
		Success: true,
		Message: "logout success",
	}, nil
}

// ForceLogout 實作 ForceLogout
func (s *MemberGRPCServer) ForceLogout(ctx context.Context, req *memberpb.ForceLogoutReq) (*memberpb.ForceLogoutRes, error) {
	logger.Log.Info("ForceLogout", zap.String("token", req.GetMemberId()))
	err := s.Usecase.ForceLogout(ctx, req.GetMemberId())
	if err != nil {
		return &memberpb.ForceLogoutRes{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &memberpb.ForceLogoutRes{
		Success: true,
		Message: "logout success",
	}, nil
}

// CheckSessionTimeout 實作 CheckSessionTimeout
func (s *MemberGRPCServer) CheckSessionTimeout(ctx context.Context, req *memberpb.CheckSessionTimeoutReq) (*memberpb.CheckSessionTimeoutRes, error) {
	logger.Log.Info("CheckSessionTimeout", zap.String("token", req.GetToken()))
	expire, err := s.Usecase.CheckSessionTimeout(ctx, req.GetToken())
	if err != nil {
		return &memberpb.CheckSessionTimeoutRes{
			Success: false,
			Expire:  expire,
			Message: err.Error(),
		}, nil
	}
	return &memberpb.CheckSessionTimeoutRes{
		Success: true,
		Expire:  expire,
		Message: "logout success",
	}, nil
}

// ReconnectSession 實作 ReconnectSession
func (s *MemberGRPCServer) ReconnectSession(ctx context.Context, req *memberpb.ReconnectSessionReq) (*memberpb.ReconnectSessionRes, error) {
	logger.Log.Info("ReconnectSession", zap.String("token", req.GetToken()))
	err := s.Usecase.ReconnectSession(ctx, req.GetToken())
	if err != nil {
		return &memberpb.ReconnectSessionRes{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	return &memberpb.ReconnectSessionRes{
		Success: true,
		Message: "logout success",
	}, nil
}
