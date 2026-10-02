package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(addr string, ttl time.Duration) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisCache{
		client: client,
		ttl:    ttl,
	}
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, bool, error) {
	value, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return value, true, nil

}

func (r *RedisCache) Set(ctx context.Context, key string, value string) error {
	return r.client.Set(ctx, key, value, r.ttl).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
