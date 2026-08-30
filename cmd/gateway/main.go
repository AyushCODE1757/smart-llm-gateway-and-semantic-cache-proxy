package main

import (
	"fmt"
	"llm-gateway/pkg/cache"
	"log"
)

func main() {
	emb, err := cache.GetOllamaEmbedding("http://localhost:11434", "Hello world")
	if err != nil {
		log.Fatalf("Embedding failed: %v", err)
	}
	fmt.Printf("Vector generated successfully! Length: %d dimensions\n", len(emb))
}
