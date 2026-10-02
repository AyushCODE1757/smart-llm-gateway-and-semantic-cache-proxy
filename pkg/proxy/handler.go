package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llm-gateway/pkg/cache"
	"llm-gateway/pkg/types"
	"net/http"
)

// GateWayProxy holds config for our route
type GateWayProxy struct {
	OllamaURL string
	Cache     cache.SemanticCache
}

// NewGateWayProxy Initializes the GateWayProxy
func NewGateWayProxy(url string, cache cache.SemanticCache) *GateWayProxy {
	return &GateWayProxy{OllamaURL: url, Cache: cache}
}

func (p *GateWayProxy) HandleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var reqBody types.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Error while Parsing the request", http.StatusBadRequest)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			_ = fmt.Sprintf("Error Closing body: %d", http.StatusInternalServerError)
			return
		}
	}(r.Body)
	payloadBytes, err := json.Marshal(reqBody)
	if err != nil {
		http.Error(w, "Failed to encode upstream payload", http.StatusInternalServerError)
		return
	}

	upStreamReq, err := http.NewRequestWithContext(r.Context(), "POST", p.OllamaURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Failed to create upstream Request", http.StatusInternalServerError)
		return
	}
	upStreamReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	upStreamResp, err := client.Do(upStreamReq)
	if err != nil {
		http.Error(w, "Failed to execute upstream Request", http.StatusInternalServerError)
		return
	}

	defer upStreamResp.Body.Close()
	for key, values := range upStreamResp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	status := upStreamResp.StatusCode
	w.WriteHeader(status)
	rc := http.NewResponseController(w)
	buf := make([]byte, 1024)
	for {
		n, err := upStreamResp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				break
			}
			if flushErr := rc.Flush(); flushErr != nil {
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Upstream stream interrupted: %v\n", err)
			}
			break
		}

	}
}
