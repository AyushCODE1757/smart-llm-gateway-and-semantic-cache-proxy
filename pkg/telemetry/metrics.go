package telemetry

import "time"

type MetricEvent struct {
	RequestID        string
	Model            string
	CacheStatus      string
	CacheSource      string
	TTFT             time.Duration
	TotalLatency     time.Duration
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CreatedAt        time.Time
}
