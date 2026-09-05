package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"
)

// Structured JSON logging format: ts, level, msg, service, trace_id
type LogEntry struct {
	TS      string `json:"ts"`
	Level   string `json:"level"`
	Msg     string `json:"msg"`
	Service string `json:"service"`
	TraceID string `json:"trace_id,omitempty"`
}

var serviceName string
var logger = log.New(os.Stderr, "", 0)

func initLogging(name string) {
	serviceName = name
}

func logInfo(msg string, traceID string) {
	emit("info", msg, traceID)
}

func logError(msg string, traceID string) {
	emit("error", msg, traceID)
}

func logWarn(msg string, traceID string) {
	emit("warn", msg, traceID)
}

func emit(level, msg, traceID string) {
	entry := LogEntry{
		TS:      time.Now().UTC().Format(time.RFC3339),
		Level:   strings.ToLower(strings.TrimSpace(level)),
		Msg:     msg,
		Service: serviceName,
		TraceID: traceID,
	}
	b, _ := json.Marshal(entry)
	logger.Println(string(b))
}
