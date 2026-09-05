package main

import (
	"log/slog"

	"tracelog"
)

var serviceName string

func initLogging(name string) { serviceName = name }

func logInfo(msg string, traceID string)  { emitLog(slog.LevelInfo, msg, traceID) }
func logError(msg string, traceID string) { emitLog(slog.LevelError, msg, traceID) }
func logWarn(msg string, traceID string)  { emitLog(slog.LevelWarn, msg, traceID) }

func emitLog(level slog.Level, msg, traceID string) {
	args := []any{"service", serviceName}
	if extra := traceLogAttrs(traceID); len(extra) > 0 {
		args = append(args, extra...)
	}
	slog.Log(nil, level, msg, args...)
}

func traceLogAttrs(traceID string) []any {
	tid := tracelog.NormalizeTraceID(traceID)
	if tid == "" {
		return nil
	}
	return []any{
		"trace_id", tid,
		"otel_trace_id", tracelog.OtelTraceIDHex(tid),
	}
}
