package cache

import (
	"context"
	"testing"
)

func TestSemanticSearch(t *testing.T) {
	ctx := context.Background()

	client, err := NewQDrantClient("localhost")
	if err != nil {
		t.Fatalf("failed to create Qdrant client: %v", err)
	}

	cache := &QdrantCache{
		client:         client,
		collectionName: "llm_prompts",
		threshold:      0.90,
	}

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

	// Generate embedding for semantically similar query.
	queryEmbedding, err := GetOllamaEmbedding(
		"http://localhost:11434",
		"What is the population of Mars?",
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

	t.Logf("Semantic HIT")
	t.Logf("Score: %f", result.Score)
	t.Logf("Value: %s", result.Value)
}
