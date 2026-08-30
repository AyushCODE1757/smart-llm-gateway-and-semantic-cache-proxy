package main

import (
	"fmt"
	"llm-gateway/pkg/proxy"
	"log"
	"net/http"
)

func main() {
	ollamaProxy := proxy.NewGateWayProxy("http://localhost:11434/v1/chat/completions")
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", ollamaProxy.HandleCompletions)
	port := ":8080"
	fmt.Printf("Starting LLM Gateway on port %s...\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server Failed: %v", err)
	}
}
