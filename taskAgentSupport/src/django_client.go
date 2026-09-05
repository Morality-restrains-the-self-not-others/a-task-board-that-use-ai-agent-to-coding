package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

// 2026-07-29: forwardToDjango removed (OPT-027) — all task-agent-support actions are now
// served directly by taskCredentialService or taskCloudService. Django internal_dispatch
// _VIEW_BY_ACTION was empty, all 14 actions returned 410 (MIGRATED).

func nestedTenantCloudPrefix(base, tenantID, workspaceID, taskID, commentID string) string {
	cid := strings.TrimSpace(commentID)
	if cid == "" || cid == "-" {
		return ""
	}
	return fmt.Sprintf("%s/api/tenant/%s/workspace/%s/task/%s/comment/%s/cloud",
		strings.TrimRight(base, "/"), tenantID, workspaceID, taskID, cid)
}

func forwardToCredentialService(ctx context.Context, action string, tenantID, workspaceID, taskID, commentID string, body map[string]any) (int, []byte, error) {
	if body == nil {
		body = map[string]any{}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}

	prefix := nestedTenantCloudPrefix(cfg.TaskCredentialServiceURL, tenantID, workspaceID, taskID, commentID)
	if prefix == "" {
		return 0, nil, fmt.Errorf("comment_id is required for credential inbound path")
	}
	url := prefix + "/server-container-token/" + action + "/"
	return doForwardPost(ctx, url, raw, timeoutForAction(action))
}

// cloudServiceInboundPath 按 /api/cloud/ 约定路径（kv-last）构建 taskCloudService 转发 URL。
// 44efa06 约定迁移后 taskCloudService 仅挂载 /api/cloud/ 约定路径（含 kv-last scope），
// 旧 /api/tenant/{tid}/workspace/{wid}/task/{tk}/cloud/... 形态一律 404 —— 此前
// container-inbound-token 恢复后容器回调在这里 404（OPT-20260809-024 同批修复）。
func cloudServiceInboundPath(action, tenantID, workspaceID, taskID, commentID string) string {
	cid := strings.TrimSpace(commentID)
	if cid == "" || cid == "-" {
		return ""
	}
	scope := fmt.Sprintf(
		"tenant_id/%s/workspace_id/%s/task_id/%s/comment_id/%s",
		tenantID,
		workspaceID,
		taskID,
		cid,
	)
	switch action {
	case "relay-status-push":
		return fmt.Sprintf("%s/api/cloud/relay-to-trae/status-push/%s", cfg.TaskCloudServiceURL, scope)
	case "model-budget-usage":
		return fmt.Sprintf("%s/api/cloud/model-budget-usage/%s", cfg.TaskCloudServiceURL, scope)
	default:
		// feature-params-env 与既有 heartbeat 等均在 server-container-token 下
		return fmt.Sprintf("%s/api/cloud/server-container-token/%s/%s", cfg.TaskCloudServiceURL, action, scope)
	}
}

func forwardToCloudService(ctx context.Context, action string, tenantID, workspaceID, taskID, commentID string, body map[string]any) (int, []byte, error) {
	if body == nil {
		body = map[string]any{}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	url := cloudServiceInboundPath(action, tenantID, workspaceID, taskID, commentID)
	if url == "" {
		return 0, nil, fmt.Errorf("comment_id is required for cloud inbound path")
	}
	return doForwardPost(ctx, url, raw, timeoutForAction(action))
}

func doForwardPost(ctx context.Context, url string, raw []byte, timeoutSec float64) (int, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := time.Duration(timeoutSec * float64(time.Second))
	client := &http.Client{Timeout: timeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	tracelog.ApplyOutboundHeaders(req, ctx)
	if cfg.InternalSecret != "" {
		req.Header.Set("X-TaskAgentSupport-Internal-Secret", cfg.InternalSecret)
	}

	resp, err := client.Do(req)
	if err != nil {
		payload, _ := json.Marshal(map[string]string{
			"detail": fmt.Sprintf("django internal upstream error: %v", err),
		})
		return http.StatusBadGateway, payload, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusBadGateway, []byte(`{"detail":"failed to read django response"}`), nil
	}
	if len(respBody) == 0 {
		respBody = []byte("{}")
	}
	return resp.StatusCode, respBody, nil
}
