package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gatewaycors"
	"snowflake"
)

// --- HTTP Helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
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
		"service": "taskTaskService",
	})
}

// (handleMetrics replaced by tracelog.MetricsHandler in main.go)

func genID(prefix string) string {
	// OPT-20260814-018: 对齐 35_snowflake_id_generation.md，使用统一 shareLib/snowflake。
	// 存量 ID（unixNano*1000+seq 形态）不改写；新建记录统一雪花形态，保留前缀便于检索。
	return fmt.Sprintf("%s_%s", prefix, snowflake.GenerateIDString())
}
