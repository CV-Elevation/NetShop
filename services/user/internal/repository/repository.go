package repository

import "context"

type User struct {
	ID       string
	Provider string
	OpenID   string
	Nickname string
	Avatar   string
	Email    string
}

// repository/repository.go
type Repository interface {
	FindByExternal(ctx context.Context, provider, openID string) (User, bool, error)
	FindByID(ctx context.Context, userID string) (User, bool, error)
	Save(ctx context.Context, user User) error
	UpdateProfile(ctx context.Context, userID, nickname, avatar, email string) (bool, error)
}
