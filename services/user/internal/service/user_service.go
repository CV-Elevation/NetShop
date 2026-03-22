package service

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"netshop/services/user/internal/repository"
)

type UserService struct {
	repo      *repository.MemoryRepository
	idCounter uint64
}

func NewUserService(repo *repository.MemoryRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) LoginOrRegister(_ context.Context, provider, openID, nickname, avatar, email string) (userID string, isNew bool, err error) {
	if provider == "" {
		return "", false, status.Error(codes.InvalidArgument, "provider is required")
	}
	if openID == "" {
		return "", false, status.Error(codes.InvalidArgument, "openid is required")
	}

	existing, ok := s.repo.FindByExternal(provider, openID)
	if ok {
		// Keep profile fresh on every third-party login.
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
		s.repo.Save(updated)
		return existing.ID, false, nil
	}

	id := fmt.Sprintf("user_%d", atomic.AddUint64(&s.idCounter, 1))
	newUser := repository.User{
		ID:       id,
		Provider: provider,
		OpenID:   openID,
		Nickname: nickname,
		Avatar:   avatar,
		Email:    email,
	}
	s.repo.Save(newUser)
	log.Printf("[user] created user: %+v", newUser)
	return id, true, nil
}

func (s *UserService) GetUser(_ context.Context, userID string) (repository.User, error) {
	if userID == "" {
		return repository.User{}, status.Error(codes.InvalidArgument, "user_id is required")
	}

	u, ok := s.repo.FindByID(userID)
	if !ok {
		return repository.User{}, status.Error(codes.NotFound, "user not found")
	}
	return u, nil
}

func (s *UserService) UpdateUser(_ context.Context, userID, nickname, avatar, email string) error {
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	ok := s.repo.UpdateProfile(userID, nickname, avatar, email)
	if !ok {
		return status.Error(codes.NotFound, "user not found")
	}
	return nil
}
