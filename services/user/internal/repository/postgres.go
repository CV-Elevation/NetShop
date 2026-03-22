package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// FindByExternal 通过第三方 provider + openID 查找用户
func (r *PostgresRepository) FindByExternal(ctx context.Context, provider, openID string) (User, bool, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT a.id, o.provider, o.provider_uid, a.username, a.avatar_url, COALESCE(a.email, '')
		FROM users.oauth o
		JOIN users.accounts a ON a.id = o.user_id
		WHERE o.provider = $1 AND o.provider_uid = $2
	`, provider, openID).Scan(&u.ID, &u.Provider, &u.OpenID, &u.Nickname, &u.Avatar, &u.Email)

	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return u, true, nil
}

// FindByID 通过平台用户 ID 查找用户
func (r *PostgresRepository) FindByID(ctx context.Context, userID string) (User, bool, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, avatar_url, COALESCE(email, '')
		FROM users.accounts
		WHERE id = $1
	`, userID).Scan(&u.ID, &u.Nickname, &u.Avatar, &u.Email)

	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return u, true, nil
}

// Save 新用户写入 accounts + oauth 两张表，已存在则更新 accounts 信息
func (r *PostgresRepository) Save(ctx context.Context, user User) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// upsert 用户主表
	_, err = tx.Exec(ctx, `
		INSERT INTO users.accounts (id, username, avatar_url, email)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		ON CONFLICT (id) DO UPDATE
		    SET username   = EXCLUDED.username,
		        avatar_url = EXCLUDED.avatar_url,
		        email      = EXCLUDED.email,
		        updated_at = NOW()
	`, user.ID, user.Nickname, user.Avatar, user.Email)
	if err != nil {
		return err
	}

	// 插入 oauth 绑定，已存在则忽略
	_, err = tx.Exec(ctx, `
		INSERT INTO users.oauth (id, user_id, provider, provider_uid)
		VALUES (gen_random_uuid(), $1, $2, $3)
		ON CONFLICT (provider, provider_uid) DO NOTHING
	`, user.ID, user.Provider, user.OpenID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateProfile 更新用户昵称、头像、邮箱
func (r *PostgresRepository) UpdateProfile(ctx context.Context, userID, nickname, avatar, email string) (bool, error) {
	result, err := r.pool.Exec(ctx, `
		UPDATE users.accounts
		SET username   = $1,
		    avatar_url = $2,
		    email      = NULLIF($3, ''),
		    updated_at = NOW()
		WHERE id = $4
	`, nickname, avatar, email, userID)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}
