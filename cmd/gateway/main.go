package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"llm-gateway/pkg/cache"
	"llm-gateway/pkg/proxy"
)

func main() {
	// 1. Connect to local Qdrant (No 'err' returned here)
	qCache := cache.NewSemanticCache("localhost", 6334, "llm_prompts")
	defer qCache.Close()

	if err := qCache.InitCollection(context.Background()); err != nil {
		log.Fatalf("Failed to initialize Qdrant collection: %v", err)
	}
	fmt.Println("Successfully connected to Qdrant via gRPC!")

	// 3. Initialize the Gateway Proxy (Passing the cache)
	gatewayProxy := proxy.NewGateWayProxy("http://localhost:11434/v1/chat/completions", *qCache)

	// 4. Setup HTTP Routes
	http.HandleFunc("/v1/chat/completions", gatewayProxy.HandleCompletions)

	// 5. Start the Server
	fmt.Println("LLM Gateway is listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
