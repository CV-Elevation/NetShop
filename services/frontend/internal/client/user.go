package client

import (
	"context"

	userpb "kuoz/netshop/platform/shared/proto/user"
)

type UserServiceClient struct {
	grpcClient userpb.UserServiceClient
}

type LoginOrRegisterRequest struct {
	Provider string
	OpenID   string
	Nickname string
	Avatar   string
	Email    string
}

type LoginOrRegisterResponse struct {
	UserID string
	IsNew  bool
}

func NewUserServiceClient(grpcClient userpb.UserServiceClient) *UserServiceClient {
	return &UserServiceClient{grpcClient: grpcClient}
}

func (c *UserServiceClient) LoginOrRegister(ctx context.Context, req LoginOrRegisterRequest) (LoginOrRegisterResponse, error) {
	resp, err := c.grpcClient.LoginOrRegister(ctx, &userpb.LoginOrRegisterRequest{
		Provider: req.Provider,
		Openid:   req.OpenID,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Email:    req.Email,
	})
	if err != nil {
		return LoginOrRegisterResponse{}, err
	}

	return LoginOrRegisterResponse{
		UserID: resp.GetUserId(),
		IsNew:  resp.GetIsNew(),
	}, nil
}
