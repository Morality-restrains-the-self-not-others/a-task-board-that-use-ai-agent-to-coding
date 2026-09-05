package main

import (
	"context"
	"net/http"
	"runAll/src/domain"
	"strings"
	"time"
)

func registerObservabilityHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/observability", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil || runner.cfg == nil {
			writeJSONError(w, "runner is required")
			return
		}
		obs := runner.cfg.Observability
		lokiExplore, err := domain.GrafanaLokiExploreLink(obs.GrafanaURL)
		if err != nil {
			writeJSONError(w, err.Error())
			return
		}
		tempoExplore, err := domain.GrafanaTempoExploreLink(obs.GrafanaURL, "")
		if err != nil {
			writeJSONError(w, err.Error())
			return
		}
		payload := map[string]string{
			"grafana_url":           obs.GrafanaURL,
			"loki_url":              obs.LokiURL,
			"grafana_loki_explore":  lokiExplore,
			"grafana_tempo_explore": tempoExplore,
			"trace_dashboard_uid":   obs.TraceDashboardUID,
			"log_file_root":         strings.TrimSpace(runner.cfg.Logging.FileRoot),
		}
		if runner.cfg.LocalPromtailEnabled() {
			payload["log_shipping"] = "local_promtail_remote_loki"
			payload["local_promtail_enabled"] = "true"
			if pushURL, err := domain.LokiPushURL(obs.LokiURL); err == nil {
				payload["loki_push_url"] = pushURL
			}
		}
		writeJSON(w, payload)
	})
	mux.HandleFunc("/api/trace-shipping/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		report := runner.TraceShippingReportForUI()
		if report == nil {
			writeJSON(w, map[string]interface{}{
				"status":  "pending",
				"message": "尚未执行 trace 投递验证，请点击「立即验证」",
			})
			return
		}
		if report.VerificationStatus == traceShippingStatusPending {
			writeJSON(w, map[string]interface{}{
				"status":              "pending",
				"message":             "尚未执行 trace 投递验证，请点击「立即验证」",
				"verification_status": report.VerificationStatus,
				"summary":             report.Summary,
				"services":            report.Services,
			})
			return
		}
		writeJSON(w, report)
	})
	mux.HandleFunc("/api/trace-shipping/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil || runner.cfg == nil {
			writeJSONError(w, "runner is required")
			return
		}
		wait := parseTraceShippingWait(r.URL.Query().Get("wait_seconds"), traceShippingDefaultWait)
		if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("skip_wait")), "true") {
			wait = 0
		}
		ctx, cancel := context.WithTimeout(r.Context(), wait+35*time.Second)
		defer cancel()
		report, err := runner.VerifyTraceShipping(ctx, wait)
		if err != nil {
			writeJSONErrorWithStatus(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, report)
	})
	mux.HandleFunc("/api/smoke/internal-apis/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		writeJSON(w, runner.GetInternalAPIsSmokeReport())
	})
	mux.HandleFunc("/api/smoke/internal-apis/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), internalAPIsSmokeDefaultTimeout)
		defer cancel()
		report := runner.VerifyInternalAPIsSmoke(ctx, "manual")
		writeJSON(w, report)
	})
	mux.HandleFunc("/api/observability/clear-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		result := runner.ClearAllObservability(r.Context())
		writeJSON(w, map[string]any{
			"status":                  result.Status,
			"memory_services_cleared": result.MemoryServicesCleared,
			"files_truncated":         result.FilesTruncated,
			"loki_reset":              result.LokiReset,
			"promtail_reset":          result.PromtailReset,
			"tempo_reset":             result.TempoReset,
			"prometheus_reset":        result.PrometheusReset,
		})
	})
	mux.HandleFunc("/api/observability/grafana-trace", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil || runner.cfg == nil {
			writeJSONError(w, "runner is required")
			return
		}

		traceID := strings.TrimSpace(r.URL.Query().Get("trace_id"))
		serviceName := strings.TrimSpace(r.URL.Query().Get("service"))
		if traceID == "" && serviceName != "" && runner.logRepository != nil {
			entries := runner.logRepository.Tail(serviceName, 200)
			for i := len(entries) - 1; i >= 0; i-- {
				if tid := domain.ExtractTraceIDFromLogMessage(entries[i].Message); tid != "" {
					traceID = tid
					break
				}
			}
		}
		if traceID == "" {
			writeJSONError(w, "trace_id is required (or provide service with trace logs)")
			return
		}

		link, err := domain.GrafanaTraceLink(
			runner.cfg.Observability.GrafanaURL,
			runner.cfg.Observability.TraceDashboardUID,
			traceID,
		)
		if err != nil {
			writeJSONError(w, err.Error())
			return
		}
		writeJSON(w, map[string]string{
			"trace_id": traceID,
			"url":      link,
		})
	})
}
