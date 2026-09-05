package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// emitStructuredFatal writes a JSON error line for Promtail/Loki level=error filters, then exits.
func emitStructuredFatal(msg string, err error) {
	full := msg
	if err != nil {
		full = fmt.Sprintf("%s: %v", msg, err)
	}
	payload := map[string]string{
		"ts":       time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"level":    "error",
		"msg":      full,
		"service":  "runall",
		"trace_id": "",
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
	os.Exit(1)
}
