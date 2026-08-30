package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llm-gateway/pkg/types"
	"net/http"
)

// GateWayProxy holds config for our route
type GateWayProxy struct {
	OllamaURL string
}

// NewGateWayProxy Initializes the GateWayProxy
func NewGateWayProxy(url string) *GateWayProxy {
	return &GateWayProxy{OllamaURL: url}
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

	upStreamReq, err := http.NewRequest("POST", p.OllamaURL, bytes.NewBuffer(payloadBytes))
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
	io.Copy(w, upStreamResp.Body)

}
