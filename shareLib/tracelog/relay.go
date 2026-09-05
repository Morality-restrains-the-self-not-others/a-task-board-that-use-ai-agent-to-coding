package tracelog

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

// ForwardChildLine writes a child process log line for runAll/Promtail collection.
func ForwardChildLine(line, defaultService string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	if forwardStructuredLine(line) {
		return
	}
	svc := strings.TrimSpace(defaultService)
	if svc == "" {
		svc = "onlineServiceJS"
	}
	emitStructuredLine(svc, "info", line, nil)
}

func forwardStructuredLine(line string) bool {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		return false
	}
	if _, ok := fields["service"]; !ok {
		return false
	}
	_, _ = os.Stdout.Write([]byte(line))
	_, _ = os.Stdout.Write([]byte("\n"))
	return true
}

func emitStructuredLine(service, level, msg string, extra map[string]string) {
	payload := map[string]string{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   NormalizeLevel(level),
		"service": service,
		"msg":     msg,
	}
	if tid := normalizeTraceID(os.Getenv("TRACE_ID")); tid != "" {
		payload["trace_id"] = tid
		payload["otel_trace_id"] = OtelTraceIDHex(tid)
	}
	for k, v := range extra {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			payload[k] = v
		}
	}
	appendAidevFields(payload)
	_ = json.NewEncoder(os.Stdout).Encode(payload)
}

// WithTraceEnv returns env with TRACE_ID set when tid is valid.
func WithTraceEnv(env map[string]string, tid string) map[string]string {
	out := make(map[string]string, len(env)+1)
	for k, v := range env {
		out[k] = v
	}
	if normalized := normalizeTraceID(tid); normalized != "" {
		out["TRACE_ID"] = normalized
	}
	return out
}
