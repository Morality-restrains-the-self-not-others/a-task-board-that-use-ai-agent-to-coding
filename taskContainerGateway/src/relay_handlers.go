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

type relayScope struct {
	TenantID    string
	WorkspaceID string
	TaskID      string
	SubAction   string
}

func parseRelayToTraePath(path string) (relayScope, bool) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	// api/tenant/{t}/workspace/{w}/task/{task}/cloud/compute/relay-to-trae/{action}
	if len(parts) < 11 {
		return relayScope{}, false
	}
	if parts[0] != "api" || parts[1] != "tenant" || parts[3] != "workspace" || parts[5] != "task" ||
		parts[7] != "cloud" || parts[8] != "compute" || parts[9] != "relay-to-trae" {
		return relayScope{}, false
	}
	action := strings.TrimSuffix(parts[10], "/")
	if action == "" {
		return relayScope{}, false
	}
	return relayScope{
		TenantID:    parts[2],
		WorkspaceID: parts[4],
		TaskID:      parts[6],
		SubAction:   action,
	}, true
}

func relayGoPath(subAction string) (method, goPath string, ok bool) {
	switch subAction {
	case "health":
		return http.MethodGet, "/health", true
	case "status":
		return http.MethodGet, "/v1/status", true
	case "clear-logs":
		return http.MethodPost, "", true // path filled with tenant/workspace/task below
	default:
		return "", "", false
	}
}

func relayClearLogsGoPath(match relayScope) string {
	return fmt.Sprintf(
		"/v1/tenant/%s/workspace/%s/task/%s/clear-logs",
		url.PathEscape(match.TenantID),
		url.PathEscape(match.WorkspaceID),
		url.PathEscape(match.TaskID),
	)
}

func forwardToRelay(ctx context.Context, method, goPath string, query string, body []byte) (int, []byte) {
	base := strings.TrimRight(strings.TrimSpace(cfg.RelayToTraeURL), "/")
	if base == "" {
		payload, _ := json.Marshal(map[string]string{"detail": "relayToTrae URL 未配置"})
		return http.StatusServiceUnavailable, payload
	}
	target := base + goPath
	if query != "" {
		target += "?" + query
	}
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		payload, _ := json.Marshal(map[string]string{"detail": fmt.Sprintf("relay request error: %v", err)})
		return http.StatusBadGateway, payload
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if secret := strings.TrimSpace(cfg.RelayToTraeSecret); secret != "" {
		req.Header.Set("X-Relay-To-Trae-Secret", secret)
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: cfg.RelayTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		payload, _ := json.Marshal(map[string]string{"detail": fmt.Sprintf("relay unreachable: %v", err)})
		return http.StatusBadGateway, payload
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		payload, _ := json.Marshal(map[string]string{"detail": "failed to read relay response"})
		return http.StatusBadGateway, payload
	}
	if len(respBody) == 0 {
		respBody = []byte("{}")
	}
	return resp.StatusCode, respBody
}

func handleRelayToTrae(w http.ResponseWriter, r *http.Request) {
	match, ok := parseRelayToTraePath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if method, lifecycle := relayLifecycleAction(match.SubAction); lifecycle {
		if r.Method != method {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
			return
		}
		switch match.SubAction {
		case "register":
			handleRelayRegister(w, r, match)
		case "start":
			handleRelayStart(w, r, match)
		case "stop":
			handleRelayStop(w, r, match)
		case "token-init":
			handleRelayTokenInit(w, r, match)
		case "env-prepare":
			handleRelayEnvPrepare(w, r, match)
		case "repo-credentials-precheck":
			handleRelayRepoCredentialsPrecheck(w, r, match)
		}
		return
	}

	method, goPath, supported := relayGoPath(match.SubAction)
	if !supported {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"detail": "relay action not proxied by gateway; use saas-backend: " + match.SubAction,
		})
		return
	}
	if r.Method != method {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	ctx := r.Context()
	scope := scope{TenantID: match.TenantID, WorkspaceID: match.WorkspaceID, TaskID: match.TaskID}
	_, status, body := authorizeContainerRequest(ctx, r, scope)
	if status != http.StatusOK {
		writeRawJSON(w, status, body)
		return
	}
	query := r.URL.RawQuery
	if match.SubAction == "status" && !strings.Contains(query, "task_id=") {
		if query != "" {
			query += "&"
		}
		query += "task_id=" + url.QueryEscape(match.TaskID)
	}
	if match.SubAction == "clear-logs" {
		goPath = relayClearLogsGoPath(match)
		query = ""
	}
	upStatus, upBody := forwardToRelay(ctx, method, goPath, query, nil)
	writeRawJSON(w, upStatus, upBody)
}
