package cache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func GetOllamaEmbedding(ollamaURL, prompt string) ([]float32, error) {
	endpoint := fmt.Sprintf("%s/api/embeddings", ollamaURL)
	ereq := EmbeddingRequest{Model: "nomic-embed-text", Prompt: prompt}

	payloadBytes, err := json.Marshal(ereq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending embedding request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embedding API returned status code: %d", resp.StatusCode)
	}

	var eresp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&eresp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return eresp.Embedding, nil
}
