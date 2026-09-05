package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type validatedContainerScope struct {
	CompanyID   string
	WorkspaceID string
	TaskID      string
	ExpiresAt   string
}

type heartbeatSession struct {
	lastContainerSeq int
	lastSaasSeq      int
	lastContainerAck int
	lastProbeOK      bool
	lastDownlinkOK   bool
	lastProbeAckVal  int
	lastProbeAckSet  bool
	lastProbeAt      time.Time
	probeInFlight    bool
}

var (
	hbMu       sync.Mutex
	hbSessions = map[string]*heartbeatSession{}
)

func inboundBodyString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	v := trim(fmt.Sprintf("%v", body[key]))
	if v == "" || v == "<nil>" {
		return ""
	}
	return v
}

func inboundHintHost(raw string) string {
	raw = trim(raw)
	if raw == "" || raw == "<nil>" {
		return ""
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return u.Hostname()
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		return host
	}
	return raw
}

func isUsableInboundPublicHost(host string) bool {
	host = trim(host)
	if host == "" {
		return false
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost", "::1", "0.0.0.0":
		return false
	}
	return true
}

func resolveInboundCommentCSC(scope *validatedContainerScope, body map[string]any, clientIP string) (*CloudServerConfig, string) {
	if scope == nil {
		return nil, "缺少评论ID"
	}
	cid := inboundBodyString(body, "comment_id")
	if cid == "" {
		cid = commentIDFromContainerName(scope.TaskID, inboundBodyString(body, "container_name"))
	}
	if cid != "" {
		cfgRow, err := loadCloudServerConfigForComment(scope.CompanyID, scope.WorkspaceID, scope.TaskID, cid)
		if err != nil || cfgRow == nil {
			return nil, "未找到该评论对应的服务器配置"
		}
		return cfgRow, ""
	}
	hints := []string{
		inboundBodyString(body, "public_ip"),
		inboundHintHost(inboundBodyString(body, "server_url")),
		inboundHintHost(inboundBodyString(body, "business_api_endpoint")),
		trim(clientIP),
	}
	for _, h := range hints {
		if !isUsableInboundPublicHost(h) {
			continue
		}
		cfgRow, err := loadCloudServerConfigByPublicIPForTask(scope.CompanyID, scope.WorkspaceID, scope.TaskID, h)
		if err != nil || cfgRow == nil {
			continue
		}
		logInfo("inbound CSC resolved by public_ip="+h+" comment_id="+cfgRow.CommentID+" task="+scope.TaskID, "")
		return cfgRow, ""
	}
	cfgRow, n, err := loadUniqueCommentCloudServerConfigWithInstance(scope.CompanyID, scope.WorkspaceID, scope.TaskID)
	if err == nil && cfgRow != nil && n == 1 {
		logInfo("inbound CSC resolved by unique comment instance comment_id="+cfgRow.CommentID+" task="+scope.TaskID, "")
		return cfgRow, ""
	}
	return nil, "缺少评论ID"
}

func handleContainerInboundToken(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, action string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "failed to read body")
		return
	}
	body := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	access := strings.TrimSpace(fmt.Sprintf("%v", body["access_token"]))
	if access == "" || access == "<nil>" {
		writeErrorJSON(w, r, http.StatusBadRequest, "access_token 必填")
		return
	}
	scope, status, detail := validateContainerAccessToken(r.Context(), access)
	if status != http.StatusOK {
		writeErrorMapJSON(w, r, status, map[string]interface{}{"detail": detail, "error_code": "TOKEN_ACCESS_INVALID"})
		return
	}
	if !containerPathMatchesScope(scope, tenantID, workspaceID, taskID) {
		writeErrorJSON(w, r, http.StatusForbidden, "URL 中的租户/工作空间/任务与令牌不匹配")
		return
	}
	cfgRow, msg := resolveInboundCommentCSC(scope, body, resolveClientIP(r))
	if msg != "" {
		code := http.StatusBadRequest
		if msg != "缺少评论ID" {
			code = http.StatusNotFound
		}
		writeErrorJSON(w, r, code, msg)
		return
	}
	if refuseInboundIfTaskTerminal(w, r, r.Context(), cfgRow, tenantID, workspaceID, taskID, action) {
		return
	}

	switch action {
	case "register-reachability":
		handleRegisterReachability(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "heartbeat":
		handleContainerHeartbeat(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID, access)
	case "git-clone-progress":
		handleGitCloneProgress(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "boot-progress":
		handleBootProgress(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "runtime-event":
		handleRuntimeEvent(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "layer-graph-push":
		handleLayerGraphPush(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "layer-changes-push":
		handleLayerChangesPush(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "job-stream-push":
		handleJobStreamPush(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "job-step-full-push":
		handleJobStepFullPush(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "feature-params-env":
		handleFeatureParamsEnv(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID, access)
	case "request-machine-release":
		handleRequestMachineRelease(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	case "git-pr-reply":
		handleGitPrReply(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID)
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "unknown action: "+action)
	}
}

// handleContainerInboundScoped 处理不在 server-container-token 下的容器 inbound
// （relay-to-trae/status-push、model-budget-usage）：鉴权后由 Cloud 编排 Django thin internal。
func handleContainerInboundScoped(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, action string) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "failed to read body")
		return
	}
	body := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	access := strings.TrimSpace(fmt.Sprintf("%v", body["access_token"]))
	if access == "" || access == "<nil>" {
		writeErrorJSON(w, r, http.StatusBadRequest, "access_token 必填")
		return
	}
	scope, status, detail := validateContainerAccessToken(r.Context(), access)
	if status != http.StatusOK {
		writeErrorMapJSON(w, r, status, map[string]interface{}{"detail": detail, "error_code": "TOKEN_ACCESS_INVALID"})
		return
	}
	if !containerPathMatchesScope(scope, tenantID, workspaceID, taskID) {
		writeErrorJSON(w, r, http.StatusForbidden, "URL 中的租户/工作空间/任务与令牌不匹配")
		return
	}
	switch action {
	case "model-budget-usage":
		handleModelBudgetUsage(w, r.Context(), body, tenantID, workspaceID, taskID)
	case "relay-status-push":
		cfgRow, msg := resolveInboundCommentCSC(scope, body, resolveClientIP(r))
		if msg != "" {
			code := http.StatusBadRequest
			if msg != "缺少评论ID" {
				code = http.StatusNotFound
			}
			writeErrorJSON(w, r, code, msg)
			return
		}
		if refuseInboundIfTaskTerminal(w, r, r.Context(), cfgRow, tenantID, workspaceID, taskID, action) {
			return
		}
		handleRelayStatusPush(w, r.Context(), cfgRow, body, tenantID, workspaceID, taskID, access, r.Header.Get("X-Trace-Id"))
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "unknown action: "+action)
	}
}

func validateContainerAccessToken(ctx context.Context, accessToken string) (*validatedContainerScope, int, string) {
	base := strings.TrimRight(cfg.CredentialServiceURL, "/")
	if base == "" {
		return nil, http.StatusBadGateway, "credential service not configured"
	}
	payload, _ := json.Marshal(map[string]string{"access_token": accessToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/token/validate", strings.NewReader(string(payload)))
	if err != nil {
		return nil, http.StatusBadGateway, err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := djangoHTTP.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, "credential validate failed: " + err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		_ = json.Unmarshal(raw, &errBody)
		detail := "无效的 access_token"
		if d, ok := errBody["detail"].(string); ok && strings.TrimSpace(d) != "" {
			detail = d
		}
		return nil, resp.StatusCode, detail
	}
	var out struct {
		Valid       bool   `json:"valid"`
		CompanyID   string `json:"company_id"`
		WorkspaceID string `json:"workspace_id"`
		TaskID      string `json:"task_id"`
		ExpiresAt   string `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || !out.Valid {
		return nil, http.StatusUnauthorized, "无效的 access_token"
	}
	return &validatedContainerScope{
		CompanyID:   out.CompanyID,
		WorkspaceID: out.WorkspaceID,
		TaskID:      out.TaskID,
		ExpiresAt:   out.ExpiresAt,
	}, http.StatusOK, ""
}

func containerPathMatchesScope(scope *validatedContainerScope, tenantID, workspaceID, taskID string) bool {
	if scope == nil {
		return false
	}
	return strings.TrimSpace(scope.CompanyID) == strings.TrimSpace(tenantID) &&
		strings.TrimSpace(scope.WorkspaceID) == strings.TrimSpace(workspaceID) &&
		strings.TrimSpace(scope.TaskID) == strings.TrimSpace(taskID)
}

func optionalHTTPURL(v any) (string, bool, string) {
	raw := strings.TrimSpace(fmt.Sprintf("%v", v))
	if raw == "" || raw == "<nil>" {
		return "", false, ""
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", true, "must be http/https URL"
	}
	return raw, true, ""
}

func parseNonNegInt(v any) *int {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case float64:
		n := int(t)
		if n < 0 {
			return nil
		}
		return &n
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil || n < 0 {
			return nil
		}
		return &n
	case json.Number:
		n64, err := t.Int64()
		if err != nil || n64 < 0 {
			return nil
		}
		n := int(n64)
		return &n
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return nil
		}
		return &n
	}
}

func nilOrInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

// commentIDFromContainerName 从评论级容器名解析 comment_id。
// 规范名：taskId 已含 task_ 时为 {taskId}_{commentId}；否则 task_{taskId}_{commentId}。
// 同时兼容存量双前缀 task_task_*。
func commentIDFromContainerName(taskID, containerName string) string {
	taskID = trim(taskID)
	containerName = trim(containerName)
	if taskID == "" || containerName == "" {
		return ""
	}
	prefixes := make([]string, 0, 2)
	if strings.HasPrefix(taskID, "task_") {
		prefixes = append(prefixes, taskID+"_")
		prefixes = append(prefixes, "task_"+taskID+"_") // legacy double prefix
	} else {
		prefixes = append(prefixes, "task_"+taskID+"_")
	}
	for _, prefix := range prefixes {
		if !strings.HasPrefix(containerName, prefix) {
			continue
		}
		cid := strings.TrimPrefix(containerName, prefix)
		if cid == "" || strings.ContainsAny(cid, "/\\ \t") {
			continue
		}
		return cid
	}
	return ""
}

func hbKey(cfgRow *CloudServerConfig) string {
	return cfgRow.ID + ":" + cfgRow.TaskID
}

func resetHeartbeatSession(cfgRow *CloudServerConfig) {
	hbMu.Lock()
	defer hbMu.Unlock()
	delete(hbSessions, hbKey(cfgRow))
}

func resetHeartbeatSessionsForTask(taskID string) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}
	suffix := ":" + taskID
	hbMu.Lock()
	defer hbMu.Unlock()
	for k := range hbSessions {
		if strings.HasSuffix(k, suffix) {
			delete(hbSessions, k)
		}
	}
}
