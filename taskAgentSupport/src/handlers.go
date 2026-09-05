package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"gatewaycors"
)

type routeMatch struct {
	TenantID    string
	WorkspaceID string
	TaskID      string
	CommentID   string
	Action      string
}

func parseCloudInboundPath(path string) (routeMatch, bool) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 11 {
		return routeMatch{}, false
	}
	if parts[0] != "api" || parts[1] != "tenant" || parts[3] != "workspace" || parts[5] != "task" {
		return routeMatch{}, false
	}
	if parts[7] != "comment" {
		return routeMatch{}, false
	}
	cid := strings.TrimSpace(parts[8])
	if cid == "" || cid == "-" {
		return routeMatch{}, false
	}
	if parts[9] != "cloud" {
		return routeMatch{}, false
	}

	match := routeMatch{
		TenantID:    parts[2],
		WorkspaceID: parts[4],
		TaskID:      parts[6],
		CommentID:   cid,
	}
	i := 10
	if i >= len(parts) {
		return routeMatch{}, false
	}

	if i+1 < len(parts) && parts[i] == "server-container-token" {
		match.Action = strings.Trim(parts[i+1], "/")
		return match, match.Action != ""
	}
	if i+1 < len(parts) && parts[i] == "relay-to-trae" && parts[i+1] == "status-push" {
		match.Action = "relay-status-push"
		return match, true
	}
	if parts[i] == "model-budget-usage" {
		match.Action = "model-budget-usage"
		return match, true
	}
	return routeMatch{}, false
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"taskAgentSupport"}`))
}

func handleCloudInbound(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}

	match, ok := parseCloudInboundPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "failed to read body"})
		return
	}

	body := map[string]any{}
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json body"})
			return
		}
	}
	if cid := strings.TrimSpace(match.CommentID); cid != "" {
		existing, _ := body["comment_id"].(string)
		if strings.TrimSpace(existing) == "" {
			body["comment_id"] = cid
		}
	}

	var statusCode int
	var respBody []byte
	var forwardErr error
	switch {
	case forwardsToCredentialService(match.Action):
		statusCode, respBody, forwardErr = forwardToCredentialService(
			r.Context(),
			match.Action,
			match.TenantID,
			match.WorkspaceID,
			match.TaskID,
			match.CommentID,
			body,
		)
	case forwardsToCloudService(match.Action):
		statusCode, respBody, forwardErr = forwardToCloudService(
			r.Context(),
			match.Action,
			match.TenantID,
			match.WorkspaceID,
			match.TaskID,
			match.CommentID,
			body,
		)
	default:
		// 2026-07-29: Django internal_dispatch _VIEW_BY_ACTION is empty — all actions
		// are either migrated to Go (CredentialService/CloudService) or unknown. Return 404 directly (OPT-027).
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": fmt.Sprintf("unknown action: %s", match.Action)})
		return
	}
	if forwardErr != nil {
		log.Printf("[taskAgentSupport] forward error action=%s task=%s: %v", match.Action, match.TaskID, forwardErr)
		writeJSON(w, http.StatusBadGateway, map[string]string{"detail": forwardErr.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write(respBody)
}

// forwardsToCredentialService：令牌/凭证/任务详情/层 OAuth → taskCredentialService（:8015）。
func forwardsToCredentialService(action string) bool {
	switch action {
	case "exchange-refresh",
		"refresh-access",
		"task-detail",
		"repo-clone-credentials",
		"layer-github-oauth-access-tokens":
		return true
	default:
		return false
	}
}

// forwardsToCloudService：可达性/心跳/克隆进度/层图推送/功能参数/relay 状态/预算用量/运行时事件 → taskCloudService（:8018）。
func forwardsToCloudService(action string) bool {
	switch action {
	case "register-reachability",
		"heartbeat",
		"git-clone-progress",
		"boot-progress",
		"layer-graph-push",
		"layer-changes-push",
		"feature-params-env",
		"relay-status-push",
		"model-budget-usage",
		"request-machine-release",
		"runtime-event",
		"job-stream-push",
		"job-step-full-push":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", gatewaycors.AllowHeaders)
			w.Header().Set("Access-Control-Expose-Headers", gatewaycors.ExposeHeaders)
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mountRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/health/", handleHealth)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			handleHealth(w, r)
			return
		}
		handleCloudInbound(w, r)
	})
}
