package telemetry

type MetricsChannel struct {
	Events chan MetricEvent
}

func NewMetricsChannel(bufferSize int) *MetricsChannel {
	return &MetricsChannel{
		Events: make(chan MetricEvent, bufferSize),
	}
}

func (m *MetricsChannel) Publish(event MetricEvent) bool {
	select {
	case m.Events <- event:
		return true
	default:
		return false
	}
}
