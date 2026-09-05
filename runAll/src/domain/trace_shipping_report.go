package domain

import "strconv"

// TraceShippingServiceResult is per-service trace log shipping verification.
type TraceShippingServiceResult struct {
	ServiceName   string `json:"service_name"`
	ServiceStatus string `json:"service_status"`
	LocalProbeOK  bool   `json:"local_probe_ok"`
	LokiVisible   bool   `json:"loki_visible"`
	LokiLogCount  int    `json:"loki_log_count"`
	GrafanaURL    string `json:"grafana_url,omitempty"`
	Issue         string `json:"issue,omitempty"`
}

// TraceShippingSummary aggregates verification counts.
type TraceShippingSummary struct {
	Total           int `json:"total"`
	ConfiguredTotal int `json:"configured_total"`
	LocalOK         int `json:"local_ok"`
	LokiOK          int `json:"loki_ok"`
	Failed          int `json:"failed"`
}

// TraceShippingReport is the full trace log shipping verification result.
type TraceShippingReport struct {
	ProbeTraceID       string                       `json:"probe_trace_id,omitempty"`
	VerifiedAt         string                       `json:"verified_at,omitempty"`
	VerificationStatus string                       `json:"verification_status,omitempty"` // pending | verified | stale
	LokiReady          bool                         `json:"loki_ready"`
	LokiError          string                       `json:"loki_error,omitempty"`
	GrafanaTraceURL    string                       `json:"grafana_trace_url,omitempty"`
	PromtailReload     string                       `json:"promtail_reload,omitempty"` // ok | not_running | failed | skipped
	Summary            TraceShippingSummary         `json:"summary"`
	Services           []TraceShippingServiceResult `json:"services"`
}

// NewProbeTraceID returns a Loki-safe probe trace id for a verification batch.
func NewProbeTraceID(nowUnixMilli int64) string {
	return "runall-ship-" + strconv.FormatInt(nowUnixMilli, 10)
}
