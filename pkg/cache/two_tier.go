package cache

import (
	"context"
	"fmt"
	"time"
)

type CacheResult struct {
	Value  string
	Hit    bool
	Source string
}

type TwoTierCache struct {
	redis  *RedisCache
	qdrant *QdrantCache
	ollama string
}

func NewTwoTierCache(
	redis *RedisCache,
	qdrant *QdrantCache,
	ollama string,
) *TwoTierCache {
	return &TwoTierCache{
		redis:  redis,
		qdrant: qdrant,
		ollama: ollama,
	}
}

func (c *TwoTierCache) GetCachedResponse(
	ctx context.Context,
	key string,
	prompt string) (*CacheResult, error) {
	// Tier one Cache check
	value, hit, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if hit {
		return &CacheResult{
			Value:  value,
			Hit:    true,
			Source: "redis",
		}, nil
	}
	embedding, err := GetOllamaEmbedding(c.ollama, prompt)
	if err != nil {
		return nil, err
	}
	result, err := c.qdrant.SimilaritySearch(ctx, embedding)
	if err != nil {
		return nil, err
	}
	if result != nil {
		return &CacheResult{
			Value:  result.Value,
			Hit:    true,
			Source: "qdrant",
		}, nil
	}
	return &CacheResult{
		Hit:    false,
		Source: "miss",
	}, nil

}

func (c *TwoTierCache) SetCachedResponse(
	ctx context.Context,
	key string,
	prompt string,
	response string,
) error {
	// Tier 1: Redis
	if err := c.redis.Set(ctx, key, response); err != nil {
		return fmt.Errorf("failed to store response in Redis: %w", err)
	}

	// Generate Embedding for sematic cache
	embedding, err := GetOllamaEmbedding(c.ollama, prompt)

	if err != nil {
		return fmt.Errorf("failed to generate cache embedding: %w", err)
	}

	// Tier 2: Qdrant
	id := uint64(time.Now().UnixNano()) // later replace with deterministic id's

	if err := c.qdrant.AddPoint(ctx, id, embedding, response); err != nil {
		return fmt.Errorf("failed to store response in Qdrant: %w", err)
	}
	return nil

}
