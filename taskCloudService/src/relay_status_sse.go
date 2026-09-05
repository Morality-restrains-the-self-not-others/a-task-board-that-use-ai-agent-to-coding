package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	relayToTraeSSEStatus = "relay_to_trae_status"

	// Coalesce high-frequency status-push SSE publishes per task.
	relayStatusSSEDebounce = 400 * time.Millisecond
)

type relayStatusSSEPending struct {
	taskID  string
	payload map[string]any
	traceID string
	timer   *time.Timer
}

var (
	relayStatusSSEMu       sync.Mutex
	relayStatusSSEByTaskID = map[string]*relayStatusSSEPending{}
)

func buildRelayToTraeStatusData(relayPayload map[string]any) map[string]interface{} {
	if relayPayload == nil {
		relayPayload = map[string]any{}
	}
	msg := strings.TrimSpace(fmt.Sprintf("%v", relayPayload["error"]))
	if msg == "" || msg == "<nil>" {
		msg = strings.TrimSpace(fmt.Sprintf("%v", relayPayload["ui_url"]))
	}
	if msg == "<nil>" {
		msg = ""
	}
	if len(msg) > 500 {
		msg = msg[:500]
	}
	return map[string]interface{}{
		"status":        relayToTraeSSEStatus,
		"event_name":    "server_status_update",
		"message":       msg,
		"progress":      0,
		"relay_payload": relayPayload,
	}
}

// publishRelayStatusSSE publishes relay status via Kafka SSE_MESSAGE (task-events → task-sse),
// bypassing Django gunicorn hot path.
func publishRelayStatusSSE(ctx context.Context, taskID string, relayPayload map[string]any, traceID string) {
	tid := strings.TrimSpace(taskID)
	if tid == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	statusData := buildRelayToTraeStatusData(relayPayload)
	if tidTrace := strings.TrimSpace(traceID); tidTrace != "" {
		statusData["trace_id"] = tidTrace
	}
	if err := publishSSEMessage(ctx, tid, statusData); err != nil {
		log.Printf("[taskCloudService] publish-relay-status-sse kafka failed task_id=%s err=%v", tid, err)
	}
}

func fireRelayStatusSSEPending(expected *relayStatusSSEPending) {
	relayStatusSSEMu.Lock()
	cur, ok := relayStatusSSEByTaskID[expected.taskID]
	if !ok || cur != expected {
		relayStatusSSEMu.Unlock()
		return
	}
	delete(relayStatusSSEByTaskID, expected.taskID)
	payload := cur.payload
	tr := cur.traceID
	tid := cur.taskID
	relayStatusSSEMu.Unlock()
	publishRelayStatusSSE(context.Background(), tid, payload, tr)
}

// scheduleRelayStatusSSEDebounced coalesces SSE publishes for the same task_id.
// Redis converge stays synchronous in applyRelayStatusPushLocal.
func scheduleRelayStatusSSEDebounced(taskID string, relayPayload map[string]any, traceID string) {
	tid := strings.TrimSpace(taskID)
	if tid == "" {
		return
	}
	payloadCopy := map[string]any{}
	for k, v := range relayPayload {
		payloadCopy[k] = v
	}
	trace := strings.TrimSpace(traceID)

	relayStatusSSEMu.Lock()
	defer relayStatusSSEMu.Unlock()
	if existing, ok := relayStatusSSEByTaskID[tid]; ok {
		existing.payload = payloadCopy
		existing.traceID = trace
		if existing.timer != nil {
			existing.timer.Stop()
		}
		existing.timer = time.AfterFunc(relayStatusSSEDebounce, func() {
			fireRelayStatusSSEPending(existing)
		})
		return
	}
	pending := &relayStatusSSEPending{
		taskID:  tid,
		payload: payloadCopy,
		traceID: trace,
	}
	pending.timer = time.AfterFunc(relayStatusSSEDebounce, func() {
		fireRelayStatusSSEPending(pending)
	})
	relayStatusSSEByTaskID[tid] = pending
}

func flushRelayStatusSSEDebounceForTest(taskID string) {
	tid := strings.TrimSpace(taskID)
	relayStatusSSEMu.Lock()
	pending, ok := relayStatusSSEByTaskID[tid]
	if !ok {
		relayStatusSSEMu.Unlock()
		return
	}
	delete(relayStatusSSEByTaskID, tid)
	if pending.timer != nil {
		pending.timer.Stop()
	}
	payload := pending.payload
	tr := pending.traceID
	relayStatusSSEMu.Unlock()
	publishRelayStatusSSE(context.Background(), tid, payload, tr)
}
