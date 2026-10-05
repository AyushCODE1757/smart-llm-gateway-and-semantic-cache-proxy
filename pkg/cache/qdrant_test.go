package cache

import (
	"context"
	"testing"
)

func TestSemanticSearch(t *testing.T) {
	ctx := context.Background()

	cache, err := NewQdrantCache(
		"localhost",
		6334,
		"llm_prompts",
		0.90,
	)
	if err != nil {
		t.Fatalf("failed to create Qdrant cache: %v", err)
	}

	defer cache.Close()

	// Create embedding for the cached prompt.
	storedEmbedding, err := GetOllamaEmbedding(
		"http://localhost:11434",
		"What is the capital of Japan?",
	)
	if err != nil {
		t.Fatalf("failed to generate stored embedding: %v", err)
	}

	// Insert test cache entry.
	err = cache.AddTestPoint(
		ctx,
		1,
		storedEmbedding,
		"The capital of Japan is Tokyo.",
	)
	if err != nil {
		t.Fatalf("failed to insert test point: %v", err)
	}

	// Generate embedding for a semantically similar query.
	queryEmbedding, err := GetOllamaEmbedding(
		"http://localhost:11434",
		"Which city is the capital of Japan?",
	)
	if err != nil {
		t.Fatalf("failed to generate query embedding: %v", err)
	}

	// Search Qdrant.
	result, err := cache.SimilaritySearch(ctx, queryEmbedding)
	if err != nil {
		t.Fatalf("semantic search failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected semantic cache HIT, got MISS")
	}

	if result.Score < 0.90 {
		t.Fatalf(
			"expected similarity score >= 0.90, got %f",
			result.Score,
		)
	}

	if result.Value != "The capital of Japan is Tokyo." {
		t.Fatalf(
			"unexpected cached value: %s",
			result.Value,
		)
	}

	t.Logf("Semantic HIT")
	t.Logf("Score: %f", result.Score)
	t.Logf("Value: %s", result.Value)
}
