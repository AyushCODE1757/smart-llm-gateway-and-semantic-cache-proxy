package cache

import (
	"llm-gateway/pkg/types"
	"testing"
)

func TestBuildCachePromptDeterministic(t *testing.T) {
	req := types.ChatCompletionRequest{
		Model:       "llama3.2",
		Temperature: 0.7,
		Messages: []types.ChatMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "What is the capital of Japan?",
			},
		},
	}

	first := BuildCachePrompt(req)
	second := BuildCachePrompt(req)

	if first != second {
		t.Fatal("expected deterministic cache prompt")
	}

	key1 := GenerateCacheKey(first)
	key2 := GenerateCacheKey(second)

	if key1 != key2 {
		t.Fatal("expected identical requests to generate identical cache keys")
	}

	t.Logf("Cache Key: %s", key1)
}
