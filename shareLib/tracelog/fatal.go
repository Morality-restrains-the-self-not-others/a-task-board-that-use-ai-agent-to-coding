package tracelog

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Fatal writes one JSON error line (level=error) then exits with status 1.
// Prefer this over log.Fatalf so Promtail/Grafana var-level=error can see startup failures.
func Fatal(service, msg string, err error) {
	if s := strings.TrimSpace(service); s != "" {
		serviceName = s
	}
	full := strings.TrimSpace(msg)
	if err != nil {
		if full == "" {
			full = err.Error()
		} else {
			full = fmt.Sprintf("%s: %v", full, err)
		}
	}
	if full == "" {
		full = "fatal"
	}
	payload := map[string]string{
		"ts":       time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"level":    "error",
		"msg":      full,
		"service":  serviceName,
		"trace_id": "",
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
	os.Exit(1)
}

// Fatalf formats msg then calls Fatal(service, msg, nil).
func Fatalf(service, format string, args ...any) {
	Fatal(service, fmt.Sprintf(format, args...), nil)
}
