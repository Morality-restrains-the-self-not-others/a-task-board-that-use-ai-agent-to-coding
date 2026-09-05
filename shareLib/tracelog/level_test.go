package tracelog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeLevel(t *testing.T) {
	cases := map[string]string{
		"INFO":     "info",
		"info":     "info",
		"Warn":     "warn",
		"WARN":     "warn",
		"warning":  "warn",
		"WARNING":  "warn",
		"ERROR":    "error",
		"error":    "error",
		"DEBUG":    "debug",
		"critical": "error",
		"FATAL":    "error",
		"trace":    "debug",
		"":         "info",
		"  Info ":  "info",
	}
	for in, want := range cases {
		if got := NormalizeLevel(in); got != want {
			t.Fatalf("NormalizeLevel(%q)=%q want %q", in, got, want)
		}
	}
}

func TestEmitPayload_NormalizesLevel(t *testing.T) {
	out := captureStdout(t, func() {
		Init("task-service-test")
		EmitComponent("WARNING", "sample", "unit", "trace-level12345678", nil)
	})
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		t.Fatalf("json: %v out=%q", err, out)
	}
	if payload["level"] != "warn" {
		t.Fatalf("level = %v want warn", payload["level"])
	}
}
