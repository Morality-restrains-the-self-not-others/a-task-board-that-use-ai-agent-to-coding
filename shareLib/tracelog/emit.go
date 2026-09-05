package tracelog

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

// Emit writes a JSON log line with optional trace id from env TRACE_ID.
func Emit(level, msg, component string, fields map[string]string) {
	EmitWithTrace(normalizeTraceID(os.Getenv("TRACE_ID")), level, msg, component, fields)
}

// EmitWithTrace writes a JSON log line with an explicit trace id when valid.
func EmitWithTrace(tid, level, msg, component string, fields map[string]string) {
	emitPayload(tid, level, msg, component, fields)
}

// EmitComponent writes a JSON log line for Kafka/event consumers (explicit trace id argument).
func EmitComponent(level, msg, component, traceID string, fields map[string]string) {
	emitPayload(normalizeTraceID(traceID), level, msg, component, fields)
}

func emitPayload(tid, level, msg, component string, fields map[string]string) {
	payload := map[string]string{
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		"level":   NormalizeLevel(level),
		"service": serviceName,
		"msg":     msg,
	}
	if component != "" {
		payload["component"] = component
	}
	if tid != "" {
		payload["trace_id"] = tid
		payload["otel_trace_id"] = OtelTraceIDHex(tid)
	}
	for k, v := range fields {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			payload[k] = v
		}
	}
	appendAidevFields(payload)
	_ = json.NewEncoder(os.Stdout).Encode(payload)
}
