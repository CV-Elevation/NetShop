package handler

import (
	"context"

	"google.golang.org/grpc"

	userpb "kuoz/netshop/platform/shared/proto/user"
	"netshop/services/user/internal/service"
)

type grpcServer struct {
	userpb.UnimplementedUserServiceServer
	svc *service.UserService
}

func Register(server *grpc.Server, svc *service.UserService) {
	userpb.RegisterUserServiceServer(server, &grpcServer{svc: svc})
}

func (s *grpcServer) LoginOrRegister(ctx context.Context, req *userpb.LoginOrRegisterRequest) (*userpb.LoginOrRegisterResponse, error) {
	userID, isNew, err := s.svc.LoginOrRegister(ctx, req.GetProvider(), req.GetOpenid(), req.GetNickname(), req.GetAvatar(), req.GetEmail())
	if err != nil {
		return nil, err
	}

	return &userpb.LoginOrRegisterResponse{
		UserId: userID,
		IsNew:  isNew,
	}, nil
}

func (s *grpcServer) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	u, err := s.svc.GetUser(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}

	return &userpb.GetUserResponse{
		UserId:   u.ID,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
		Email:    u.Email,
	}, nil
}

func (s *grpcServer) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	err := s.svc.UpdateUser(ctx, req.GetUserId(), req.GetNickname(), req.GetAvatar(), req.GetEmail())
	if err != nil {
		return nil, err
	}

	return &userpb.UpdateUserResponse{Ok: true}, nil
}
