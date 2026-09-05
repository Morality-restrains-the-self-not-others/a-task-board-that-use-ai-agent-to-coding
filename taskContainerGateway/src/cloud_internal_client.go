package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

const (
	relayPhaseTokenInitSucceeded = "token_init_succeeded"
	relayPhaseStartAccepted      = "start_accepted"
)

// cloudInternalPost posts JSON to taskCloudService internal APIs.
func cloudInternalPost(ctx context.Context, path string, payload map[string]any) error {
	base := strings.TrimRight(strings.TrimSpace(cfg.CloudServiceURL), "/")
	if base == "" {
		return fmt.Errorf("task cloud service url not configured")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(cfg.CloudInternalSecret); secret != "" {
		req.Header.Set("X-Internal-Secret", secret)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		tracelog.LogForwardStage(ctx, "cloud_internal", map[string]any{
			"path": path, "ok": false, "detail": err.Error(),
		})
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		tracelog.LogForwardStage(ctx, "cloud_internal", map[string]any{
			"path": path, "ok": false, "cloud_status": resp.StatusCode,
		})
		return errHTTPStatus{status: resp.StatusCode, body: string(body)}
	}
	tracelog.LogForwardStage(ctx, "cloud_internal", map[string]any{
		"path": path, "ok": true, "cloud_status": resp.StatusCode,
	})
	return nil
}

// upsertRelayStartupSessionViaCloud writes Redis session via Cloud serialize/save.
// Failures are logged but do not fail the caller HTTP response (best-effort hot path).
func upsertRelayStartupSessionViaCloud(ctx context.Context, payload map[string]any) {
	if strings.TrimSpace(cfg.CloudServiceURL) == "" {
		log.Printf("[taskContainerGateway] skip relay startup session upsert: cloud url empty")
		return
	}
	if err := cloudInternalPost(ctx, "/api/internal/relay-startup-session/upsert/", payload); err != nil {
		log.Printf("[taskContainerGateway] relay startup session upsert failed: %v", err)
	}
}

func relayStartupSessionTokenInitPayload(sc scope, workflowID string) map[string]any {
	return map[string]any{
		"workflow_id":       workflowID,
		"tenant_id":         sc.TenantID,
		"workspace_id":      sc.WorkspaceID,
		"task_id":           sc.TaskID,
		"phase":             relayPhaseTokenInitSucceeded,
		"token_initialized": true,
	}
}

func relayStartupSessionStartAcceptedPayload(sc scope, workflowID, requestID string) map[string]any {
	return map[string]any{
		"workflow_id":       workflowID,
		"tenant_id":         sc.TenantID,
		"workspace_id":      sc.WorkspaceID,
		"task_id":           sc.TaskID,
		"phase":             relayPhaseStartAccepted,
		"token_initialized": true,
		"request_id":        requestID,
	}
}
