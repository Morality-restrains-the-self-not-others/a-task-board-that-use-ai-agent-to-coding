package domain

import "context"

// ObservabilityStorageResetter clears AiMonitor Loki/Tempo/Prometheus data volumes.
type ObservabilityStorageResetter interface {
	Reset(ctx context.Context) (StorageResetOutcome, error)
}

// StorageResetOutcome reports per-backend reset status ("ok" or "error: ...").
type StorageResetOutcome struct {
	LokiReset       string
	PromtailReset   string
	TempoReset      string
	PrometheusReset string
}

func (o StorageResetOutcome) AllOK() bool {
	return isResetOK(o.LokiReset) &&
		isResetOK(o.PromtailReset) &&
		isResetOK(o.TempoReset) &&
		isResetOK(o.PrometheusReset)
}

func isResetOK(status string) bool {
	return status == "ok" || status == "skipped"
}
