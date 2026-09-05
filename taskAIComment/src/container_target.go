package main

import (
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

var probeHTTP = &http.Client{Timeout: 7 * time.Second}
var credentialHTTP = &http.Client{Timeout: 5 * time.Second}

func normalizeURLOrigin(raw string) (origin, token string) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", ""
	}
	if !strings.HasPrefix(text, "http://") && !strings.HasPrefix(text, "https://") {
		text = "http://" + text
	}
	u, err := url.Parse(text)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", ""
	}
	origin = strings.TrimRight(fmt.Sprintf("%s://%s", u.Scheme, u.Host), "/")
	if q := u.Query().Get("access_token"); q != "" {
		token = strings.TrimSpace(q)
	}
	if token == "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 && parts[0] == "ui" {
			token, _ = url.PathUnescape(parts[1])
			token = strings.TrimSpace(token)
		}
	}
	return origin, token
}

func fetchTokenByScope(tenantID, workspaceID, taskID, commentID string) string {
	base := strings.TrimRight(cfg.CredentialServiceURL, "/")
	if base == "" {
		return ""
	}
	reqURL := base + "/v1/token/by-scope?tenant_id=" + url.QueryEscape(tenantID) +
		"&workspace_id=" + url.QueryEscape(workspaceID) +
		"&task_id=" + url.QueryEscape(taskID)
	if cid := strings.TrimSpace(commentID); cid != "" {
		reqURL += "&comment_id=" + url.QueryEscape(cid)
	}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	resp, err := credentialHTTP.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return ""
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if json.Unmarshal(raw, &data) != nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", data["access_token"]))
}

func resolveContainerTarget(cfg *cloudServerConfig) (baseURL, token string) {
	if cfg == nil {
		return "", ""
	}
	serverOrigin, serverToken := normalizeURLOrigin(cfg.ServerURL)
	bizOrigin, bizToken := normalizeURLOrigin(cfg.BusinessAPIEndpoint)
	baseURL = serverOrigin
	if baseURL == "" {
		baseURL = bizOrigin
	}
	token = serverToken
	if token == "" {
		token = bizToken
	}
	if token == "" && cfg.CompanyID != "" && cfg.TaskID != "" {
		token = fetchTokenByScope(cfg.CompanyID, cfg.WorkspaceID, cfg.TaskID, cfg.CommentID)
	}
	return baseURL, token
}

func detectStreamBackend(baseURL, token, traceID string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	tok := strings.TrimSpace(token)
	if base == "" || tok == "" {
		return "legacy"
	}
	req, err := http.NewRequest(http.MethodGet, base+"/api/requirements/task-gate", nil)
	if err != nil {
		return "legacy"
	}
	req.Header.Set("X-Access-Token", tok)
	req.Header.Set("Accept", "application/json")
	tracelog.ApplyOutboundHeaders(req, context.Background())
	resp, err := probeHTTP.Do(req)
	if err != nil {
		return "legacy"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "legacy"
	}
	raw, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	if json.Unmarshal(raw, &data) != nil {
		return "legacy"
	}
	if _, ok := data["clone_done"]; ok {
		return "trae"
	}
	return "legacy"
}

func legacyInstructURL(baseURL, tenantID, workspaceID, taskID string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	t := url.PathEscape(tenantID)
	w := url.PathEscape(workspaceID)
	tk := url.PathEscape(taskID)
	return fmt.Sprintf("%s/api/tenant/%s/workspace/%s/task/%s/ai/instructs/", base, t, w, tk)
}

func instructErrorAPIRef(baseURL, tenantID, workspaceID, taskID, streamBackend string) string {
	if streamBackend == "trae" {
		return strings.TrimRight(baseURL, "/") + "/api/jobs (Trae Online Service，skill.md)"
	}
	return legacyInstructURL(baseURL, tenantID, workspaceID, taskID)
}
