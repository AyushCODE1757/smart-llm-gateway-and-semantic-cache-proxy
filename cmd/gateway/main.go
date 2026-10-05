package main

import (
	"context"
	"fmt"
	"llm-gateway/pkg/cache"
	"llm-gateway/pkg/proxy"
	"log"
	"net/http"
	"time"
)

func main() {
	ctx := context.Background()
	// 1. Redis Cache
	redisCache := cache.NewRedisCache("localhost:6379", 30*time.Second)
	defer redisCache.Close()

	// 2. Qdrant Cache
	qdrantCache, err := cache.NewQdrantCache("localhost", 6334, "llm_prompts", 0.9)
	if err != nil {
		log.Fatalf("Failed to initialize Qdrant cache: %v", err)
	}
	defer qdrantCache.Close()

	// 3. Initialize Qdrant Collection
	if err := qdrantCache.InitCollection(ctx); err != nil {
		log.Fatalf("Failed to initialize Qdrant collection: %v", err)
	}

	fmt.Println("Successfully connected to Qdrant!")

	// 4. Two Tier Cache
	twoTierCache := cache.NewTwoTierCache(redisCache, qdrantCache, "http://localhost:11434")

	// 5. Gateway Proxy
	gatewayProxy := proxy.NewGateWayProxy("http://localhost:11434/v1/chat/completions", twoTierCache)

	// 6. Http Route
	http.HandleFunc(
		"/v1/chat/completions",
		gatewayProxy.HandleCompletions,
	)

	// 7. Start the Gateway
	fmt.Println("LLM Gateway is listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

}
