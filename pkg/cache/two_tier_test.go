package cache

import (
	"context"
	"testing"
	"time"
)

func createTestTwoTierCache(t *testing.T) (*TwoTierCache, *RedisCache) {
	t.Helper()

	redisCache := NewRedisCache(
		"localhost:6379",
		time.Minute,
	)

	qdrantClient, err := NewQDrantClient("localhost")
	if err != nil {
		redisCache.Close()
		t.Fatalf("failed to create Qdrant client: %v", err)
	}

	qdrantCache := &QdrantCache{
		client:         qdrantClient,
		collectionName: "llm_prompts",
		threshold:      0.90,
	}

	cache := NewTwoTierCache(
		redisCache,
		qdrantCache,
		"http://localhost:11434",
	)

	return cache, redisCache
}

func TestTwoTierCache_RedisHit(t *testing.T) {
	ctx := context.Background()

	cache, redisCache := createTestTwoTierCache(t)
	defer redisCache.Close()

	key := "test:two-tier:redis-hit"

	err := redisCache.Set(
		ctx,
		key,
		"The capital of Japan is Tokyo.",
	)
	if err != nil {
		t.Fatalf("failed to populate Redis: %v", err)
	}

	result, err := cache.GetCachedResponse(
		ctx,
		key,
		"What is the capital of India?",
	)
	if err != nil {
		t.Fatalf("cache lookup failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected cache result, got nil")
	}

	if !result.Hit {
		t.Fatal("expected cache HIT, got MISS")
	}

	if result.Source != "redis" {
		t.Fatalf("expected source redis, got %s", result.Source)
	}

	if result.Value != "The capital of Japan is Tokyo." {
		t.Fatalf("unexpected cached value: %s", result.Value)
	}

	t.Logf(
		"Cache HIT | Source: %s | Value: %s",
		result.Source,
		result.Value,
	)
}

func TestTwoTierCache_QdrantHit(t *testing.T) {
	ctx := context.Background()

	cache, redisCache := createTestTwoTierCache(t)
	defer redisCache.Close()

	// This key is intentionally NOT inserted into Redis.
	key := "test:two-tier:qdrant-hit"

	result, err := cache.GetCachedResponse(
		ctx,
		key,
		"Which city is the capital of Japan?",
	)
	if err != nil {
		t.Fatalf("cache lookup failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected cache result, got nil")
	}

	if !result.Hit {
		t.Fatal("expected semantic cache HIT, got MISS")
	}

	if result.Source != "qdrant" {
		t.Fatalf("expected source qdrant, got %s", result.Source)
	}

	if result.Value != "The capital of Japan is Tokyo." {
		t.Fatalf("unexpected cached value: %s", result.Value)
	}

	t.Logf(
		"Cache HIT | Source: %s | Value: %s",
		result.Source,
		result.Value,
	)
}

func TestTwoTierCache_Miss(t *testing.T) {
	ctx := context.Background()

	cache, redisCache := createTestTwoTierCache(t)
	defer redisCache.Close()

	// This key is intentionally NOT inserted into Redis.
	key := "test:two-tier:miss"

	result, err := cache.GetCachedResponse(
		ctx,
		key,
		"What is the population of Mars?",
	)
	if err != nil {
		t.Fatalf("cache lookup failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected cache result, got nil")
	}

	if result.Hit {
		t.Fatalf(
			"expected cache MISS, got HIT from %s",
			result.Source,
		)
	}

	if result.Source != "miss" {
		t.Fatalf(
			"expected source miss, got %s",
			result.Source,
		)
	}

	t.Logf(
		"Cache MISS | Source: %s",
		result.Source,
	)
}
