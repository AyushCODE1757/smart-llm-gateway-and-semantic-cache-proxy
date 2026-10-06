package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"llm-gateway/pkg/cache"
	"llm-gateway/pkg/telemetry"
	"llm-gateway/pkg/types"
)

// GateWayProxy holds config for our route.
type GateWayProxy struct {
	OllamaURL string
	Cache     *cache.TwoTierCache
	Metrics   *telemetry.MetricsChannel
}

// NewGateWayProxy initializes the GateWayProxy.
func NewGateWayProxy(
	url string,
	cache *cache.TwoTierCache,
	metricChannel *telemetry.MetricsChannel) *GateWayProxy {
	return &GateWayProxy{
		OllamaURL: url,
		Cache:     cache,
		Metrics:   metricChannel,
	}
}

func (p *GateWayProxy) HandleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestStart := time.Now()
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
		fmt.Printf(
			"Cache HIT | Source: %s\n",
			cacheResult.Source,
		)
		p.recordCacheHitMetric(
			requestStart,
			reqBody.Model,
			cacheResult.Source,
		)
		if reqBody.Stream {
			p.streamCachedResponse(
				w,
				reqBody.Model,
				cacheResult.Value,
			)
			return
		}
		p.writeCachedResponse(w, reqBody.Model, cacheResult.Value)
		return
	}

	fmt.Println("Cache MISS | Forwarding request to Ollama")
	if reqBody.Stream {
		reqBody.StreamOptions = &types.StreamOptions{
			IncludeUsage: true,
		}
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
	if reqBody.Stream {
		p.handleStreamingMiss(
			w,
			upstreamResp,
			cacheKey,
			cachePrompt,
			reqBody.Model,
			requestStart,
		)
		return
	}
	p.handleNonStreamingMiss(
		w,
		r,
		upstreamResp,
		cacheKey,
		cachePrompt,
	)
}

// Cache Hit Non-Streaming

func (p *GateWayProxy) writeCachedResponse(w http.ResponseWriter, model string, value string) {
	w.Header().Set("Content-Type", "application/json")
	response := types.ChatCompletionResponse{
		Id:      "cache-hit",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []types.ChatCompletionChoice{
			{
				Index: 0,
				Message: types.ChatMessage{
					Role:    "assistant",
					Content: value,
				},
				FinishReason: "stop",
			},
		},
		Usage: types.UsageMetrics{},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf(
			"Failed to encode cache response: %v\n",
			err,
		)
	}

}

// Cache Hit Streaming
func (p *GateWayProxy) streamCachedResponse(w http.ResponseWriter, model string, value string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	rc := http.NewResponseController(w)

	role := "assistant"
	firstChunk := types.ChatCompletionStreamResponse{
		ID:      "cache-hit",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []types.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: types.ChatDelta{
					Role: role,
				},
				FinishReason: nil,
			},
		},
	}
	p.writeSSEChunk(w, rc, firstChunk)
	contentChunk := types.ChatCompletionStreamResponse{
		ID:      "cache-hit",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []types.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: types.ChatDelta{
					Content: value,
				},
				FinishReason: nil,
			},
		},
	}
	p.writeSSEChunk(w, rc, contentChunk)
	finishReason := "stop"
	finalChunk := types.ChatCompletionStreamResponse{
		ID:      "cache-hit",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []types.ChatCompletionStreamChoice{
			{
				Index: 0,
				Delta: types.ChatDelta{
					Role: role,
				},
				FinishReason: &finishReason,
			},
		},
	}
	p.writeSSEChunk(w, rc, finalChunk)
	fmt.Fprint(w, "data: [DONE]\n\n")
	_ = rc.Flush()
}

// Cache Miss 	Streaming

func (p *GateWayProxy) handleStreamingMiss(
	w http.ResponseWriter,
	upstreamResp *http.Response,
	cacheKey string,
	cachePrompt string,
	model string,
	requestStart time.Time) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	rc := http.NewResponseController(w)

	scanner := bufio.NewScanner(upstreamResp.Body)
	scanner.Buffer(
		make([]byte, 4096),
		1024*1024,
	)
	var responseBuilder strings.Builder
	var ttft time.Duration
	var firstTokenReceived bool
	var promptTokens int64
	var completionTokens int64
	var totalTokens int64
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "data: ")
		if line == "[DONE]" {
			break
		}
		var ollamaChunk struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			Model   string `json:"model"`

			Choices []struct {
				Index int `json:"index"`

				Delta struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"delta"`

				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`

			Usage struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
				TotalTokens      int64 `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(
			[]byte(line),
			&ollamaChunk,
		); err != nil {
			fmt.Printf(
				"Failed to parse Ollama stream chunk: %v\n",
				err,
			)
			continue
		}
		if len(ollamaChunk.Choices) > 0 {

			content := ollamaChunk.Choices[0].Delta.Content
			if content != "" {
				if !firstTokenReceived {
					ttft = time.Since(requestStart)
					firstTokenReceived = true
				}
				responseBuilder.WriteString(content)
				streamChunk := types.ChatCompletionStreamResponse{
					ID:      "chatcmpl-stream",
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   model,
					Choices: []types.ChatCompletionStreamChoice{
						{
							Index: 0,
							Delta: types.ChatDelta{
								Content: content,
							},
						},
					},
				}
				if err := p.writeSSEChunk(w, rc, streamChunk); err != nil {
					fmt.Printf(
						"Failed to stream chunk to client: %v\n",
						err,
					)
					return
				}

			}

		}
		if ollamaChunk.Usage.TotalTokens > 0 {
			promptTokens = ollamaChunk.Usage.PromptTokens
			completionTokens = ollamaChunk.Usage.CompletionTokens
			totalTokens = ollamaChunk.Usage.TotalTokens
		}
	}
	totalLatency := time.Since(requestStart)
	event := telemetry.MetricEvent{
		Model:            model,
		CacheStatus:      "miss",
		CacheSource:      "none",
		TTFT:             ttft,
		TotalLatency:     totalLatency,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		CreatedAt:        time.Now(),
	}
	p.Metrics.Publish(event)
	if err := scanner.Err(); err != nil {
		fmt.Printf("Upstream stream interrupted: %v\n", err)
		return
	}
	finishReason := "stop"
	finalChunk := types.ChatCompletionStreamResponse{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Choices: []types.ChatCompletionStreamChoice{
			{
				Index:        0,
				Delta:        types.ChatDelta{},
				FinishReason: &finishReason,
			},
		},
	}
	if err := p.writeSSEChunk(
		w,
		rc,
		finalChunk,
	); err != nil {
		return
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	_ = rc.Flush()

	assistantResponse := responseBuilder.String()
	if assistantResponse == "" {
		fmt.Println(
			"Empty assistant response; skipping cache",
		)
		return
	}
	cacheKeyCopy := cacheKey
	cachePromptCopy := cachePrompt
	responseCopy := assistantResponse

	go func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()
		err := p.Cache.SetCachedResponse(ctx, cacheKeyCopy, cachePromptCopy, responseCopy)
		if err != nil {
			fmt.Printf(
				"Async cache failed: %v\n",
				err,
			)
			return
		}
		fmt.Printf(
			"Async cache populated | Key: %s\n",
			cacheKeyCopy,
		)
	}()

}

// SSE Writer
func (p *GateWayProxy) writeSSEChunk(
	w http.ResponseWriter,
	rc *http.ResponseController,
	chunk types.ChatCompletionStreamResponse,
) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		w,
		"data: %s\n\n",
		data,
	); err != nil {
		return err
	}
	return rc.Flush()
}

// Non Streaming miss
func (p *GateWayProxy) handleNonStreamingMiss(
	w http.ResponseWriter,
	r *http.Request,
	upstreamResp *http.Response,
	cacheKey string,
	cachePrompt string,
) {
	responseBody, err := io.ReadAll(upstreamResp.Body)
	if err != nil {
		http.Error(
			w,
			"Failed to read upstream response",
			http.StatusBadGateway,
		)
		return
	}
	for key, values := range upstreamResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(upstreamResp.StatusCode)
	if _, err := w.Write(responseBody); err != nil {
		fmt.Printf(
			"Failed to write upstream response: %v\n",
			err,
		)
		return
	}
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
		fmt.Println("Empty assistant response; skipping cache")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()
		if err := p.Cache.SetCachedResponse(
			ctx,
			cacheKey,
			cachePrompt,
			assistantResponse,
		); err != nil {
			fmt.Printf(
				"Async cache population failed: %v\n",
				err,
			)
			return
		}
		fmt.Printf(
			"Async cache populated | Key: %s\n",
			cacheKey,
		)

	}()

}
func (p *GateWayProxy) recordCacheHitMetric(
	requestStartTime time.Time,
	model string,
	source string,
) {
	event := telemetry.MetricEvent{
		Model:        model,
		CacheStatus:  "hit",
		CacheSource:  source,
		TTFT:         0,
		TotalLatency: time.Since(requestStartTime),
		CreatedAt:    time.Now(),
	}
	p.Metrics.Publish(event)
}
