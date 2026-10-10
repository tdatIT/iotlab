package cache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisImpl struct {
	client redis.UniversalClient
}

// NewRedisCache returns a Cache backed by the given Redis client.
func NewRedisCache(client redis.UniversalClient) Cache {
	return &redisImpl{client: client}
}

func (r redisImpl) Get(ctx context.Context, key string) ([]byte, bool) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) { // Key does not exist
			return nil, false
		}
		slog.Error("error while getting value from redis",
			slog.String("key", key),
			slog.String("error", err.Error()))
		return nil, false
	}

	return []byte(val), true
}

func (r redisImpl) Set(ctx context.Context, key string, value []byte) error {
	return r.client.Set(ctx, key, string(value), 0).Err()
}

func (r redisImpl) SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, string(value), ttl).Err()
}

func (r redisImpl) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
