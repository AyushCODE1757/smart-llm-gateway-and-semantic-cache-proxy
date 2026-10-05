package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"llm-gateway/pkg/cache"
	"llm-gateway/pkg/types"
)

// GateWayProxy holds config for our route.
type GateWayProxy struct {
	OllamaURL string
	Cache     *cache.TwoTierCache
}

// NewGateWayProxy initializes the GateWayProxy.
func NewGateWayProxy(url string, cache *cache.TwoTierCache) *GateWayProxy {
	return &GateWayProxy{
		OllamaURL: url,
		Cache:     cache,
	}
}

func (p *GateWayProxy) HandleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var reqBody types.ChatCompletionRequest

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(
			w,
			"Error while parsing the request",
			http.StatusBadRequest,
		)
		return
	}
	cachePrompt := cache.BuildCachePrompt(reqBody)
	cacheKey := cache.GenerateCacheKey(cachePrompt)
	cacheResult, err := p.Cache.GetCachedResponse(
		r.Context(),
		cacheKey,
		cachePrompt,
	)

	if err != nil {
		http.Error(
			w,
			"Cache lookup failed",
			http.StatusInternalServerError,
		)
		return
	}
	if cacheResult.Hit {

		w.Header().Set("Content-Type", "application/json")

		response := types.ChatCompletionResponse{
			Id:      "cache-hit",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   reqBody.Model,
			Choices: []types.ChatCompletionChoice{
				{
					Index: 0,
					Message: types.ChatMessage{
						Role:    "assistant",
						Content: cacheResult.Value,
					},
					FinishReason: "stop",
				},
			},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			fmt.Printf("Failed to encode cache response: %v\n", err)
		}

		return
	}
	payloadBytes, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(
			w,
			"Failed to encode upstream payload",
			http.StatusInternalServerError,
		)
		return
	}

	upstreamReq, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		p.OllamaURL,
		bytes.NewBuffer(payloadBytes),
	)

	if err != nil {
		http.Error(
			w,
			"Failed to create upstream request",
			http.StatusInternalServerError,
		)
		return
	}

	upstreamReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	upstreamResp, err := client.Do(upstreamReq)
	if err != nil {
		http.Error(
			w,
			"Failed to execute upstream request",
			http.StatusInternalServerError,
		)
		return
	}

	defer upstreamResp.Body.Close()

	responseBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		http.Error(
			w,
			"Failed to read upstream response",
			http.StatusBadGateway,
		)
		return
	}

	// --------------------------------------------------
	// 8. Forward upstream headers
	// --------------------------------------------------

	for key, values := range upstreamResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// --------------------------------------------------
	// 9. Forward response to client
	// --------------------------------------------------

	w.WriteHeader(upstreamResp.StatusCode)

	if _, err := w.Write(responseBody); err != nil {
		fmt.Printf(
			"Failed to write upstream response: %v\n",
			err,
		)
		return
	}

	// --------------------------------------------------
	// 10. Parse response for caching
	// --------------------------------------------------

	var upstreamResponse types.ChatCompletionResponse

	if err := json.Unmarshal(responseBody, &upstreamResponse); err != nil {
		fmt.Printf(
			"Failed to parse upstream response for caching: %v\n",
			err,
		)
		return
	}

	if len(upstreamResponse.Choices) == 0 {
		fmt.Println(
			"Upstream response contains no choices; skipping cache",
		)
		return
	}

	assistantResponse := upstreamResponse.Choices[0].Message.Content

	if assistantResponse == "" {
		fmt.Println(
			"Empty assistant response; skipping cache",
		)
		return
	}

	// --------------------------------------------------
	// 11. Populate Redis + Qdrant
	// --------------------------------------------------

	if err := p.Cache.SetCachedResponse(
		r.Context(),
		cacheKey,
		cachePrompt,
		assistantResponse,
	); err != nil {
		fmt.Printf(
			"Failed to populate cache: %v\n",
			err,
		)
		return
	}

	fmt.Printf(
		"Cache populated | Key: %s\n",
		cacheKey,
	)
}
