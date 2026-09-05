package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"runAll/src/domain"
	"sort"
	"time"
)

func registerStatusHandlers(mux *http.ServeMux, store *StatusStore, runner *Runner, shutdownFunc context.CancelFunc, exitReason *runAllExitReason) {
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner != nil {
			runner.reconcileFailedServiceHealthThrottled(r.Context())
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		payload := map[string]any{
			"services": buildStatusPayload(store, runner),
		}
		// Include proxy config so the UI can render the toggle correctly
		if runner != nil && runner.cfg != nil {
			payload["use_proxy"] = runner.cfg.UseProxy
		}
		if runner != nil {
			payload["dag_boot_in_progress"] = runner.IsDAGBootInProgress()
			payload["poll_interval_ms"] = statusPollIntervalMs
			if snap := runner.ActiveBulkProgress(); snap != nil {
				payload["active_bulk_progress"] = snap
			}
			payload["execution_queue"] = runner.ExecutionQueue()
		}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			log.Printf("[ui] json encode error: %v", err)
		}
	})
	mux.HandleFunc("/api/shutdown-self", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		log.Printf("[api] shutdown-self requested - closing runAll without stopping managed services")
		// Record the exit trigger so a silent shutdown is attributable from logs.
		if exitReason != nil {
			exitReason.record("shutdown-self", "/api/shutdown-self")
		}
		// Set flag to skip shutting down managed services
		if runner != nil {
			runner.SetSkipShutdownServices(true)
		}
		writeJSON(w, map[string]string{"status": "ok", "message": "runAll shutting down, managed services will continue running"})
		// Trigger shutdown asynchronously after sending response
		go func() {
			time.Sleep(100 * time.Millisecond)
			if shutdownFunc != nil {
				shutdownFunc()
			}
		}()
	})
	mux.HandleFunc("/api/log-collection-status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		provider := runner.LogStatsProvider()
		if provider == nil {
			writeJSONError(w, "log stats provider is not available")
			return
		}
		allStats := provider.AllStats()

		// Ensure every configured service has an entry, even if it has produced zero logs.
		if runner.cfg != nil {
			for _, svc := range runner.cfg.Flatten() {
				if _, ok := allStats[svc.Name]; !ok {
					allStats[svc.Name] = domain.LogCollectionStats{ServiceName: svc.Name}
				}
			}
		}

		// Build per-service payloads with additional runtime context.
		result := make([]logCollectionEntry, 0, len(allStats))
		for name, stats := range allStats {
			entry := logCollectionEntry{
				ServiceName: name,
				TotalLines:  stats.TotalLines,
				TotalBytes:  stats.TotalBytes,
			}
			if !stats.FirstLogAt.IsZero() {
				entry.FirstLogAt = stats.FirstLogAt.Format(time.RFC3339)
			}
			if !stats.LastLogAt.IsZero() {
				entry.LastLogAt = stats.LastLogAt.Format(time.RFC3339)
			}

			// A service is "actively collecting" if it has produced a log in the last 5 minutes.
			entry.IsActive = !stats.LastLogAt.IsZero() && time.Since(stats.LastLogAt) < 5*time.Minute

			if st := runner.store.Get(name); st != nil {
				entry.ServiceStatus = string(st.Status)
				entry.Group = st.Group
			}
			entry.HasLogFile = runner.fileLogSink != nil

			result = append(result, entry)
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].ServiceName < result[j].ServiceName
		})

		writeJSON(w, map[string]interface{}{
			"services":        result,
			"total_services":  len(result),
			"active_services": countCollectionActive(result),
			"has_file_sink":   runner.fileLogSink != nil,
		})
	})
}

type logCollectionEntry struct {
	ServiceName   string `json:"service_name"`
	TotalLines    int64  `json:"total_lines"`
	TotalBytes    int64  `json:"total_bytes"`
	FirstLogAt    string `json:"first_log_at,omitempty"`
	LastLogAt     string `json:"last_log_at,omitempty"`
	ServiceStatus string `json:"service_status"`
	Group         string `json:"group"`
	HasLogFile    bool   `json:"has_log_file"`
	IsActive      bool   `json:"is_active"`
}

func countCollectionActive(entries []logCollectionEntry) int {
	n := 0
	for _, e := range entries {
		if e.IsActive {
			n++
		}
	}
	return n
}

// proxyConfigPayload is the JSON response for GET /api/proxy-config.
