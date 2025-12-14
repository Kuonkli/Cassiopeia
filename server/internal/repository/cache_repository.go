// internal/repository/cache_repository.go
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type CacheRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetJSON(ctx context.Context, key string, dest interface{}) error
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Increment(ctx context.Context, key string) (int64, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	FlushAll(ctx context.Context) error
}

type cacheRepository struct {
	client *redis.Client
}

func NewCacheRepository(client *redis.Client) CacheRepository {
	return &cacheRepository{client: client}
}

func (r *cacheRepository) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Ключ не найден - это не ошибка
	}
	return val, err
}

func (r *cacheRepository) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	var err error
	switch v := value.(type) {
	case string:
		err = r.client.Set(ctx, key, v, expiration).Err()
	case []byte:
		err = r.client.Set(ctx, key, v, expiration).Err()
	default:
		jsonData, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		err = r.client.Set(ctx, key, jsonData, expiration).Err()
	}
	return err
}

func (r *cacheRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *cacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.client.Exists(ctx, key).Result()
	return exists > 0, err
}

func (r *cacheRepository) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // Ключ не найден
		}
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

func (r *cacheRepository) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(ctx, key, jsonData, expiration).Err()
}

func (r *cacheRepository) Increment(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *cacheRepository) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

func (r *cacheRepository) FlushAll(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}
