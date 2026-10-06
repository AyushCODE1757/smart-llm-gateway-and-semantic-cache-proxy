package telemetry

import "testing"

func TestMetricsChannel(t *testing.T) {
	metrics := NewMetricsChannel(2)

	event := MetricEvent{
		RequestID: "test-123",
		Model:     "llama3.2:latest",
	}

	if !metrics.Publish(event) {
		t.Fatal("expected first metric to be published")
	}

	if !metrics.Publish(event) {
		t.Fatal("expected second metric to be published")
	}

	if metrics.Publish(event) {
		t.Fatal("expected third metric to be rejected when buffer is full")
	}
}
