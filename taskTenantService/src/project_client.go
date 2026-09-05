package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"tracelog"
)

func projectPOST(ctx context.Context, path string, body map[string]interface{}) (int, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskProjectServiceURL), "/")
	if base == "" {
		return 200, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	// OPT-20260821-012: 把入站 trace/span 透传到 taskProjectService。
	tracelog.ApplyOutboundHeaders(req, ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Id", "internal")
	if cfg.GatewayInternalSecret != "" {
		req.Header.Set("X-TaskGateway-Internal-Secret", cfg.GatewayInternalSecret)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	return resp.StatusCode, nil
}

func ensureWorkspaceAccess(ctx context.Context, tenantID, workspaceID, companyMemberID string, isAdmin bool) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" || companyMemberID == "" || strings.TrimSpace(cfg.TaskProjectServiceURL) == "" {
		return
	}
	perm := "view"
	if isAdmin {
		perm = "edit"
	}
	status, err := projectPOST(ctx, fmt.Sprintf("/api/projects/workspace-access/tenant_id/%s/set-permission", tenantID), map[string]interface{}{
		"workspace_id":      workspaceID,
		"company_member_id": companyMemberID,
		"permission":        perm,
	})
	if err != nil || (status >= 400 && status != 409) {
		logWarn(fmt.Sprintf("workspace access create failed status=%d err=%v", status, err), "")
	}
}

func workspaceName(ctx context.Context, tenantID, workspaceID string) string {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskProjectServiceURL), "/")
	if base == "" || strings.TrimSpace(workspaceID) == "" {
		return ""
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/projects/workspaces/tenant_id/%s/%s",
		base, url.PathEscape(tenantID), url.PathEscape(workspaceID)), nil)
	if err != nil {
		return ""
	}
	// OPT-20260821-012: 把入站 trace/span 透传到 taskProjectService。
	tracelog.ApplyOutboundHeaders(req, ctx)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	_ = json.Unmarshal(raw, &out)
	if n, ok := out["name"].(string); ok {
		return n
	}
	return ""
}
