package service

import (
	"context"
	"log"

	"netshop/services/user/internal/repository"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	repo repository.Repository // 换成接口，不再依赖具体实现
}

func NewUserService(repo repository.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) LoginOrRegister(ctx context.Context, provider, openID, nickname, avatar, email string) (userID string, isNew bool, err error) {
	if provider == "" {
		return "", false, status.Error(codes.InvalidArgument, "provider is required")
	}
	if openID == "" {
		return "", false, status.Error(codes.InvalidArgument, "openid is required")
	}

	existing, ok, err := s.repo.FindByExternal(ctx, provider, openID)
	if err != nil {
		return "", false, status.Error(codes.Internal, "db error")
	}
	if ok {
		// 每次登录刷新用户信息
		updated := existing
		if nickname != "" {
			updated.Nickname = nickname
		}
		if avatar != "" {
			updated.Avatar = avatar
		}
		if email != "" {
			updated.Email = email
		}
		if err := s.repo.Save(ctx, updated); err != nil {
			return "", false, status.Error(codes.Internal, "db error")
		}
		return existing.ID, false, nil
	}

	// 新用户，后端生成 UUID
	newUser := repository.User{
		ID:       uuid.New().String(),
		Provider: provider,
		OpenID:   openID,
		Nickname: nickname,
		Avatar:   avatar,
		Email:    email,
	}
	if err := s.repo.Save(ctx, newUser); err != nil {
		return "", false, status.Error(codes.Internal, "db error")
	}
	log.Printf("[user] created user: %+v", newUser)
	return newUser.ID, true, nil
}

func (s *UserService) GetUser(ctx context.Context, userID string) (repository.User, error) {
	if userID == "" {
		return repository.User{}, status.Error(codes.InvalidArgument, "user_id is required")
	}
	u, ok, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return repository.User{}, status.Error(codes.Internal, "db error")
	}
	if !ok {
		return repository.User{}, status.Error(codes.NotFound, "user not found")
	}
	return u, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID, nickname, avatar, email string) error {
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	ok, err := s.repo.UpdateProfile(ctx, userID, nickname, avatar, email)
	if err != nil {
		return status.Error(codes.Internal, "db error")
	}
	if !ok {
		return status.Error(codes.NotFound, "user not found")
	}
	return nil
}
