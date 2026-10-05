package cache

import (
	"context"
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
