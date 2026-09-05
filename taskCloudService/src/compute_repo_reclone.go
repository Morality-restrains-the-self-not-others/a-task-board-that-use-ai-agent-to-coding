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
	"time"

	"tracelog"
)

func handleRepoReclone(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse body
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 32*1024))
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "无法读取请求体")
		return
	}
	var body map[string]interface{}
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "无效的JSON请求体")
			return
		}
	}

	repoURL := strings.TrimSpace(strField(body, "repo_url"))
	if repoURL == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "repo_url 不能为空")
		return
	}
	if len(repoURL) > 4096 {
		writeErrorJSON(w, r, http.StatusBadRequest, "repo_url 过长")
		return
	}

	commentID := commentIDFromComputeRequest(r, body)
	if rejectMissingComputeCommentID(w, r, commentID) {
		return
	}
	cfg, err := resolveScopedCloudServerConfig(tenantID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		writeErrorJSON(w, r, http.StatusNotFound, "未找到任务服务器配置")
		return
	}

	baseURL, token := resolveContainerTarget(cfg, "")
	if baseURL == "" {
		writeErrorJSON(w, r, http.StatusConflict, "容器尚未注册 server_url，请先完成启动与 exchange-refresh")
		return
	}
	if token == "" {
		writeErrorMapJSON(w, r, http.StatusConflict, map[string]interface{}{
			"detail":     "缺少容器 access_token",
			"error_code": "CONTAINER_TOKEN_MISSING",
		})
		return
	}

	// Forward to container's /api/repos/reclone
	containerURL := strings.TrimRight(baseURL, "/") + "/api/repos/reclone"
	payload := map[string]interface{}{"repo_url": repoURL}
	if cloneAlias := strings.TrimSpace(strField(body, "clone_alias")); cloneAlias != "" {
		// Light sanitisation: reject path traversal
		safe := strings.TrimSpace(strings.ReplaceAll(cloneAlias, "\\", "/"))
		if strings.Contains(safe, "..") || strings.HasPrefix(safe, "/") {
			writeErrorJSON(w, r, http.StatusBadRequest, "无效的 clone_alias")
			return
		}
		payload["clone_alias"] = safe
	}
	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, containerURL, bytes.NewReader(payloadBytes))
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, "构建转发请求失败")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Access-Token", token)

	proxyHTTP := &http.Client{Timeout: 5 * time.Minute}
	resp, err := proxyHTTP.Do(req)
	if err != nil {
		logWarn("repo reclone forward failed: "+err.Error(), r.Header.Get("X-Trace-Id"))
		writeErrorJSON(w, r, http.StatusBadGateway, "容器转发失败: "+err.Error())
		return
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	var respJSON map[string]interface{}
	if len(respBytes) > 0 {
		json.Unmarshal(respBytes, &respJSON)
	}
	if respJSON == nil {
		respJSON = map[string]interface{}{}
	}

	status := resp.StatusCode
	if status >= 400 {
		detail, _ := respJSON["detail"].(string)
		if detail == "" {
			detail = fmt.Sprintf("HTTP %d", status)
		}
		writeErrorJSON(w, r, http.StatusBadGateway, detail)
		return
	}

	out := map[string]interface{}{"ok": true, "result": respJSON}
	if status == http.StatusAccepted {
		out["async"] = true
	}
	writeJSON(w, http.StatusOK, out)
}

// projectServiceGET makes a GET to taskProjectService internal API.
func projectServiceGET(ctx context.Context, path string, query url.Values) (*http.Response, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.ProjectServiceURL), "/")
	if base == "" {
		return nil, fmt.Errorf("project service url not configured")
	}
	u := base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-User-Id", "internal")
	// OPT-20260821-012: 把入站 X-Trace-Id / span 透传到 taskProjectService。
	tracelog.ApplyOutboundHeaders(req, ctx)
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}
