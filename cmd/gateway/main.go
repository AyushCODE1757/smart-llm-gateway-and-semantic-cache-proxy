package main

import (
	"context"
	"fmt"
	"llm-gateway/pkg/cache"
	"llm-gateway/pkg/clickhouse"
	"llm-gateway/pkg/proxy"
	"llm-gateway/pkg/telemetry"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}
	clickhouseClient, err := clickhouse.NewClient()
	if err != nil {
		log.Fatalf("ClickHouse client creation failed: %v", err)
	}

	if err := clickhouseClient.Ping(context.Background()); err != nil {
		log.Fatalf("ClickHouse ping failed: %v", err)
	}

	log.Println("✅ ClickHouse connection successful")

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

	// 5. Initialize Metrics Channel

	metrics := telemetry.NewMetricsChannel(1000)

	// 6. Start Async Worker
	clickhouse.StartMetricsWorker(ctx, clickhouseClient, metrics, 3)

	// 7. Gateway Proxy
	gatewayProxy := proxy.NewGateWayProxy("http://localhost:11434/v1/chat/completions", twoTierCache, metrics)

	// 8. Http Route
	http.HandleFunc(
		"/v1/chat/completions",
		gatewayProxy.HandleCompletions,
	)

	// 9. Start the Gateway
	fmt.Println("LLM Gateway is listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

}
