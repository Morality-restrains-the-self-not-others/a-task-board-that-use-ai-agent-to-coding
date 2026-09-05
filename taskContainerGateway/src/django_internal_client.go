package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type errHTTPStatus struct {
	status int
	body   string
}

func (e errHTTPStatus) Error() string {
	return e.body
}

const relayToTraeSSEStatus = "relay_to_trae_status"

func buildRelayToTraeStatusData(relayPayload map[string]any) map[string]any {
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
	return map[string]any{
		"status":        relayToTraeSSEStatus,
		"event_name":    "server_status_update",
		"message":       msg,
		"progress":      0,
		"relay_payload": relayPayload,
	}
}

// publishRelayStatusSSE publishes relay status via Kafka SSE_MESSAGE (same path as Cloud),
// bypassing Django gunicorn.
func publishRelayStatusSSE(ctx context.Context, taskID string, relayPayload map[string]any) {
	tid := strings.TrimSpace(taskID)
	if tid == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	statusData := buildRelayToTraeStatusData(relayPayload)
	if err := publishSSEMessage(ctx, tid, statusData); err != nil {
		slog.Warn("publish-relay-status-sse kafka failed", "task_id", tid, "error", err)
	}
}

// ensureOpenRuntimeSession asks Cloud to open a runtime session and create
// CloudServerConfig (relay-local) so container callbacks like register-reachability
// can resolve TOKEN_SCOPE.
func ensureOpenRuntimeSession(ctx context.Context, sc scope) error {
	return cloudInternalPost(ctx, "/api/internal/runtime-session/open/", map[string]any{
		"tenant_id":                  sc.TenantID,
		"workspace_id":               sc.WorkspaceID,
		"task_id":                    sc.TaskID,
		"runtime_source":             "relay_local",
		"ensure_cloud_server_config": true,
	})
}
