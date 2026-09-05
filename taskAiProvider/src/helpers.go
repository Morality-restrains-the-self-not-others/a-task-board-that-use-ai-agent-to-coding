package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	return dec.Decode(dst)
}

func parsePathID(path, prefix string) (int64, bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return 0, false
	}
	parts := strings.Split(rest, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func pathAction(path, prefix string) (id int64, action string, ok bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return 0, "", false
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	if len(parts) >= 2 {
		action = parts[1]
	}
	return id, action, true
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}

func logInfo(format string, args ...any) { log.Printf("[taskAiProvider] "+format, args...) }

// logWarn/logError emit structured JSON so Loki can correlate by trace_id
// (OPT-20260901-025). Callers pass the request ctx; background extractors
// thread a ctx (or context.Background) captured at schedule time.
func logWarn(ctx context.Context, format string, args ...any) {
	tracelog.EmitWithTrace(
		tracelog.TraceIDFromContext(ctx),
		"warn",
		fmt.Sprintf("[taskAiProvider] "+format, args...),
		"ai-provider",
		nil,
	)
}

func logError(ctx context.Context, format string, args ...any) {
	tracelog.EmitWithTrace(
		tracelog.TraceIDFromContext(ctx),
		"error",
		fmt.Sprintf("[taskAiProvider] "+format, args...),
		"ai-provider",
		nil,
	)
}

func copyHeader(dst, src http.Header, keys ...string) {
	for _, k := range keys {
		if v := src.Get(k); v != "" {
			dst.Set(k, v)
		}
	}
}

func drainAndClose(c io.Closer) {
	if c != nil {
		_ = c.Close()
	}
}
