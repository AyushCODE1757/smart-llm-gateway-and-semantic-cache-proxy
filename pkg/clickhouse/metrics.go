package clickhouse

import (
	"context"
	"fmt"

	"llm-gateway/pkg/telemetry"
)

func (c *Client) InsertMetric(ctx context.Context, event telemetry.MetricEvent) error {
	query := `
		INSERT INTO default.llm_metrics
		(
			request_id,
			model,
			cache_status,
			cache_source,
			ttft_ms,
			total_latency_ms,
			prompt_tokens,
			completion_tokens,
			total_tokens,
			created_at
		)
		VALUES
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	err := c.conn.Exec(
		ctx,
		query,
		event.RequestID,
		event.Model,
		event.CacheStatus,
		event.CacheSource,
		event.TTFT.Milliseconds(),
		event.TotalLatency.Milliseconds(),
		event.PromptTokens,
		event.CompletionTokens,
		event.TotalTokens,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert metric: %w", err)
	}
	return nil

}
