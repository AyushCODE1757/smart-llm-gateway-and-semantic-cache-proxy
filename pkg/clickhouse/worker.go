package clickhouse

import (
	"context"
	"llm-gateway/pkg/telemetry"
	"log"
	"sync"
)

func StartMetricsWorker(
	ctx context.Context,
	client *Client,
	metrics *telemetry.MetricsChannel,
	workerCount int,
) *sync.WaitGroup {
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			log.Printf("Metrics worker %d started", workerID)

			for {
				select {
				case <-ctx.Done():
					log.Printf("Metrics Worker %d Stopped", workerID)
					return

				case event := <-metrics.Events:
					if err := client.InsertMetric(ctx, event); err != nil {
						log.Printf("Worker %d: Failed to insert metric: %v", workerID, err)
					}

				}
			}

		}(i + 1)
		go func() {
			wg.Wait()
			log.Println("All metrics workers stopped")
		}()

	}
	return &wg
}
