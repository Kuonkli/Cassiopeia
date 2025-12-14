package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

type CacheRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Increment(ctx context.Context, key string) (int64, error)
	GetJSON(ctx context.Context, key string, dest interface{}) error
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

type redisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) CacheRepository {
	return &redisRepository{client: client}
}

func (r *redisRepository) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil // Ключ не найден - это не ошибка
	}
	return val, err
}

func (r *redisRepository) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	var err error
	switch v := value.(type) {
	case string:
		err = r.client.Set(ctx, key, v, expiration).Err()
	case []byte:
		err = r.client.Set(ctx, key, v, expiration).Err()
	default:
		// Сериализуем в JSON
		jsonData, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		err = r.client.Set(ctx, key, jsonData, expiration).Err()
	}
	return err
}

func (r *redisRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *redisRepository) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := r.client.Exists(ctx, key).Result()
	return exists > 0, err
}

func (r *redisRepository) Increment(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *redisRepository) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil // Ключ не найден
		}
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

func (r *redisRepository) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(ctx, key, jsonData, expiration).Err()
}

// Дополнительные методы для работы с паттернами
func (r *redisRepository) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.client.Keys(ctx, pattern).Result()
}

func (r *redisRepository) FlushAll(ctx context.Context) error {
	return r.client.FlushAll(ctx).Err()
}
