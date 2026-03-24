package repository

import (
	"context"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type Repository interface {
	AddItem(ctx context.Context, userID, productID string, quantity int32) (int32, error)
	GetItems(ctx context.Context, userID string) (map[string]int32, error)
	GetChecked(ctx context.Context, userID string) (map[string]bool, error)
	SetChecked(ctx context.Context, userID, productID string, checked bool) error
	ClearCart(ctx context.Context, userID string) error
}

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

func itemsKey(userID string) string   { return "cart:items:" + userID }
func checkedKey(userID string) string { return "cart:checked:" + userID }

// AddItem 累加数量，同时自动勾选
func (r *RedisRepository) AddItem(ctx context.Context, userID, productID string, quantity int32) (int32, error) {
	newQty, err := r.client.HIncrBy(ctx, itemsKey(userID), productID, int64(quantity)).Result()
	if err != nil {
		return 0, err
	}
	if err := r.client.SAdd(ctx, checkedKey(userID), productID).Err(); err != nil {
		return 0, err
	}
	return int32(newQty), nil
}

// GetItems 返回 product_id -> quantity 映射
func (r *RedisRepository) GetItems(ctx context.Context, userID string) (map[string]int32, error) {
	raw, err := r.client.HGetAll(ctx, itemsKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]int32, len(raw))
	for k, v := range raw {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			continue
		}
		result[k] = int32(n)
	}
	return result, nil
}

// GetChecked 返回 product_id -> checked 映射
func (r *RedisRepository) GetChecked(ctx context.Context, userID string) (map[string]bool, error) {
	members, err := r.client.SMembers(ctx, checkedKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(members))
	for _, m := range members {
		result[m] = true
	}
	return result, nil
}

// SetChecked 勾选或取消勾选
func (r *RedisRepository) SetChecked(ctx context.Context, userID, productID string, checked bool) error {
	if checked {
		return r.client.SAdd(ctx, checkedKey(userID), productID).Err()
	}
	return r.client.SRem(ctx, checkedKey(userID), productID).Err()
}

// ClearCart 清空购物车
func (r *RedisRepository) ClearCart(ctx context.Context, userID string) error {
	return r.client.Del(ctx, itemsKey(userID), checkedKey(userID)).Err()
}
