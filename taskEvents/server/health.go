package server

import (
	"encoding/json"
	"net/http"

	"taskEvents/broker"
	"taskEvents/domain"
)

// HealthState is exposed by /api/health/.
type HealthState struct {
	Status           string           `json:"status"`
	Domain           string           `json:"domain"`
	Transport        string           `json:"transport"`
	SubscribedEvents []string         `json:"subscribed_events"`
	MQConnected      bool             `json:"mq_connected,omitempty"`
	DLTCounters      map[string]int64 `json:"dlt_counters,omitempty"`
}

// NewHealthHandler returns a JSON liveness handler (process is up).
func NewHealthHandler(domainName string, transport domain.TransportKind, events []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(HealthState{
			Status:           "ok",
			Domain:           domainName,
			Transport:        string(transport),
			SubscribedEvents: events,
			DLTCounters:      broker.DLTCounters(),
		})
	}
}

// NewReadinessHandler returns readiness based on broker connectivity.
func NewReadinessHandler(domainName string, transport domain.TransportKind, events []string, mq domain.MQConnectivity) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		connHealth := domain.NewConnectionHealth(mq, transport)
		code := http.StatusOK
		if !connHealth.Connected {
			code = http.StatusServiceUnavailable
		}
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(HealthState{
			Status:           connHealth.ReadinessStatus(),
			Domain:           domainName,
			Transport:        string(transport),
			SubscribedEvents: events,
			MQConnected:      connHealth.Connected,
			DLTCounters:      broker.DLTCounters(),
		})
	}
}
