package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"gatewaycors"
	"snowflake"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeErrorJSON sends an error response with trace_id extracted from the request.
// Use this for 5xx/4xx error responses to enable frontend data-traceId binding.
func writeErrorJSON(w http.ResponseWriter, r *http.Request, status int, detail string) {
	body := map[string]string{"detail": detail}
	if r != nil {
		if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
			body["trace_id"] = tid
			w.Header().Set("X-Trace-Id", tid)
		}
	}
	writeJSON(w, status, body)
}

// writeErrorMapJSON is like writeErrorJSON but accepts a pre-built error map
// (e.g. {"error": "...", "status": "error", ...}) and injects trace_id.
// Use this when the error response shape is more complex than just {"detail": msg}.
func writeErrorMapJSON(w http.ResponseWriter, r *http.Request, status int, body map[string]interface{}) {
	if r != nil {
		if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
			body["trace_id"] = tid
			w.Header().Set("X-Trace-Id", tid)
		}
	}
	writeJSON(w, status, body)
}

// traceIDFromRequest extracts the trace_id from the request header.
// Returns empty string if not present.
func traceIDFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Trace-Id"))
}

func readJSONBody(r *http.Request) (map[string]interface{}, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	return body, nil
}

func strField(body map[string]interface{}, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

// normalizeAuthorizationIDField extracts snowflake authorization_id without float64 scientific notation corruption.
func normalizeAuthorizationIDField(body map[string]interface{}, key string) string {
	v, ok := body[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if strings.ContainsAny(s, "eE") {
			return ""
		}
		return s
	case json.Number:
		return strings.TrimSpace(t.String())
	case float64:
		// JSON numbers >2^53 lose integer precision as float64; prefer platform fallback when unusable.
		if t != float64(int64(t)) {
			return ""
		}
		return strconv.FormatInt(int64(t), 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeaders)
			w.Header().Set("Access-Control-Expose-Headers", gatewaycors.ExposeHeaders)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "taskCloudService"})
}

// (handleMetrics replaced by tracelog.MetricsHandler in main.go)

func genID(prefix string) string {
	// 对齐 35_snowflake_id_generation.md。存量溢出负 ID 不改写。
	return fmt.Sprintf("%s_%s", prefix, snowflake.GenerateIDString())
}
