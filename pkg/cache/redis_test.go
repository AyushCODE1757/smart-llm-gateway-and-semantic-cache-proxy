package cache

import (
	"context"
	"testing"
	"time"
)

func TestRedisCache(t *testing.T) {
	ctx := context.Background()

	cache := NewRedisCache("localhost:6379", 30*time.Second)
	defer cache.Close()

	key := GenerateCacheKey("redis test prompt")
	value := `{"message":"hello from redis"}`

	// SET
	err := cache.Set(ctx, key, value)
	if err != nil {
		t.Fatalf("SET failed: %v", err)
	}

	// GET
	got, found, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	if !found {
		t.Fatal("expected cache HIT, got MISS")
	}

	if got != value {
		t.Fatalf("expected %q, got %q", value, got)
	}

	t.Log("Redis SET/GET: PASS")
}
