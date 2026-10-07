package clickhouse

import (
	"context"
	"llm-gateway/pkg/telemetry"
	"log"
)

func StartMetricsWorker(
	ctx context.Context,
	client *Client,
	metrics *telemetry.MetricsChannel,
) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Metrics Worker Stopped")
				return

			case event := <-metrics.Events:
				if err := client.InsertMetric(ctx, event); err != nil {
					log.Printf("Failed to insert metric: %v", err)
				}

			}
		}

	}()
}
