package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"
)

type logEntry struct {
	TS      string `json:"ts"`
	Level   string `json:"level"`
	Msg     string `json:"msg"`
	Service string `json:"service"`
	TraceID string `json:"trace_id,omitempty"`
}

var serviceName = "taskAIComment"
var structuredLogger = log.New(os.Stderr, "", 0)

func logInfo(msg, traceID string) {
	emitLog("info", msg, traceID)
}

func emitLog(level, msg, traceID string) {
	entry := logEntry{
		TS:      time.Now().UTC().Format(time.RFC3339),
		Level:   strings.ToLower(strings.TrimSpace(level)),
		Msg:     msg,
		Service: serviceName,
		TraceID: traceID,
	}
	b, _ := json.Marshal(entry)
	structuredLogger.Println(string(b))
}
