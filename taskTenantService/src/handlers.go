package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gatewaycors"
	"tracelog"
)

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// traceIDForError resolves a trace id for error-body injection:
// X-Trace-Id header first (gateway-bridged), then context trace id.
func traceIDForError(r *http.Request) string {
	if r == nil {
		return ""
	}
	if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
		return tid
	}
	return strings.TrimSpace(tracelog.TraceIDFromContext(r.Context()))
}

// writeError writes a uniform {status, error, message, trace_id?} error body.
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	body := map[string]interface{}{"status": "error", "error": message, "message": message}
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorDetail writes a {status, error, detail, message, trace_id?} error body.
// Frontend reads data.detail first (taskAuth FE error contract), so detail-first.
func writeErrorDetail(w http.ResponseWriter, r *http.Request, status int, detail string) {
	body := map[string]interface{}{
		"status":  "error",
		"error":   detail,
		"detail":  detail,
		"message": detail,
	}
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorMap writes a composite error body, injecting trace_id and filling
// in status/message (from error or detail) when absent while preserving any
// extra keys the caller provided.
func writeErrorMap(w http.ResponseWriter, r *http.Request, status int, body map[string]interface{}) {
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	if _, ok := body["status"]; !ok {
		body["status"] = "error"
	}
	if _, ok := body["message"]; !ok {
		if msg, ok := body["error"]; ok {
			body["message"] = msg
		} else if detail, ok := body["detail"]; ok {
			body["message"] = detail
		}
	}
	writeJSON(w, status, body)
}

func readJSONBody(r *http.Request) (map[string]interface{}, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]interface{}{}, nil
	}
	// UseNumber keeps snowflake/bigint IDs exact (default float64 loses precision > 2^53).
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var body map[string]interface{}
	if err := dec.Decode(&body); err != nil {
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
	case json.Number:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func boolField(body map[string]interface{}, key string) bool {
	v, ok := body[key]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true") || t == "1"
	case float64:
		return t != 0
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			f, err2 := t.Float64()
			return err2 == nil && f != 0
		}
		return n != 0
	default:
		return false
	}
}

func intField(body map[string]interface{}, key string, def int) int {
	v, ok := body[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return def
		}
		return int(n)
	case string:
		n := 0
		for _, c := range t {
			if c < '0' || c > '9' {
				return def
			}
			n = n*10 + int(c-'0')
		}
		if t == "" {
			return def
		}
		return n
	default:
		return def
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
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "taskTenantService",
	})
}
