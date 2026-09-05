package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type relayRegisterBody struct {
	TenantID              string `json:"tenant_id"`
	WorkspaceID           string `json:"workspace_id"`
	TaskID                string `json:"task_id"`
	TaskAPIEndpointOrigin string `json:"task_api_endpoint_origin"`
	AccessToken           string `json:"access_token"`
	RegisterOnly          bool   `json:"register_only"`
}

type relayStartBody struct {
	TenantID    string            `json:"tenant_id"`
	WorkspaceID string            `json:"workspace_id"`
	TaskID      string            `json:"task_id"`
	Env         map[string]string `json:"env"`
}

func readRequestJSON(r *http.Request) (map[string]any, []byte, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) == 0 {
		return map[string]any{}, raw, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, raw, err
	}
	if parsed == nil {
		parsed = map[string]any{}
	}
	return parsed, raw, nil
}

// resolveRelayStartImage resolves docker image ref for relay start.
// When neither image nor installed_image_id is provided, returns empty image (legacy run.sh).
// When installed_image_id is provided, Cloud resolve-image must succeed.
func resolveRelayStartImage(ctx context.Context, tenantID string, bodyMap map[string]any) (image string, installedImageID string, status int, body []byte) {
	image = strField(bodyMap, "image")
	installedImageID = strField(bodyMap, "installed_image_id")
	if installedImageID == "" {
		installedImageID = strField(bodyMap, "installedImageId")
	}
	if image == "" && installedImageID == "" {
		return "", "", http.StatusOK, nil
	}
	if image != "" {
		return image, installedImageID, http.StatusOK, nil
	}
	q := url.Values{}
	q.Set("tenant_id", strings.TrimSpace(tenantID))
	q.Set("installed_image_id", installedImageID)
	resStatus, resBody := cloudGetJSON(
		ctx,
		"/api/internal/image/resolve?"+q.Encode(),
		"cloud_relay_resolve_image",
	)
	if resStatus != http.StatusOK {
		return "", installedImageID, resStatus, resBody
	}
	var resolved map[string]any
	_ = json.Unmarshal(resBody, &resolved)
	image = strField(resolved, "image")
	if image == "" {
		payload, _ := json.Marshal(map[string]string{
			"status":  "error",
			"message": "所选镜像未配置可拉取地址（image_url）",
		})
		return "", installedImageID, http.StatusBadRequest, payload
	}
	return image, installedImageID, http.StatusOK, nil
}

func relayStrField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				if t := strings.TrimSpace(s); t != "" {
					return t
				}
			}
		}
	}
	return ""
}

func relayBoolField(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	default:
		return false
	}
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte("fallback-request-id"))
	}
	return hex.EncodeToString(buf)
}

func relayScopeFromBody(m map[string]any, match relayScope) scope {
	tenant := relayStrField(m, "tenant_id", "tenantId")
	if tenant == "" {
		tenant = match.TenantID
	}
	wid := relayStrField(m, "workspace_id", "workspaceId")
	if wid == "" {
		wid = match.WorkspaceID
	}
	tid := relayStrField(m, "task_id", "taskId")
	if tid == "" {
		tid = match.TaskID
	}
	cid := relayStrField(m, "comment_id", "COMMENT_ID")
	return scope{TenantID: tenant, WorkspaceID: wid, TaskID: tid, CommentID: cid}
}

func relayLifecycleEventData(sc scope, extra map[string]any) map[string]any {
	data := map[string]any{
		"tenant_id":    sc.TenantID,
		"workspace_id": sc.WorkspaceID,
		"task_id":      sc.TaskID,
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

func handleRelayRegister(w http.ResponseWriter, r *http.Request, match relayScope) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	ctx := r.Context()
	bodyMap, _, err := readRequestJSON(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	sc := relayScopeFromBody(bodyMap, match)
	if sc.TaskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "缺少 task_id"})
		return
	}
	_, status, vbody := authorizeContainerRequest(ctx, r, sc)
	if status != http.StatusOK {
		writeRawJSON(w, status, vbody)
		return
	}

	taskAPIOrigin := relayStrField(bodyMap, "task_api_endpoint_origin", "TASK_API_ENDPOINT_ORIGIN")
	accessToken := relayStrField(bodyMap, "access_token", "ACCESS_TOKEN")
	registerOnly := relayBoolField(bodyMap, "register_only")

	publishRelayLifecycleEvent(ctx, "RELAY_REGISTER_ATTEMPTED", relayLifecycleEventData(sc, nil), sc.TaskID)

	if accessTokenNeedsServerIssue(accessToken) && !registerOnly {
		tok, tStatus, tBody := credentialTokenInit(ctx, sc)
		if tStatus != http.StatusOK {
			publishRelayLifecycleEvent(ctx, "RELAY_REGISTER_FAILED", relayLifecycleEventData(sc, map[string]any{
				"error_code": "token_init_failed",
			}), sc.TaskID)
			writeRawJSON(w, tStatus, tBody)
			return
		}
		accessToken = tok.AccessToken
	} else if registerOnly {
		accessToken = ""
	}

	if taskAPIOrigin == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "missing task_api_endpoint_origin"})
		return
	}
	if accessToken == "" && !registerOnly {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "missing access_token"})
		return
	}

	payload := map[string]string{
		"tenant_id":                sc.TenantID,
		"workspace_id":             sc.WorkspaceID,
		"task_id":                  sc.TaskID,
		"task_api_endpoint_origin": taskAPIOrigin,
	}
	if accessToken != "" {
		payload["access_token"] = accessToken
	}
	fwdBody, _ := json.Marshal(payload)
	upStatus, upBody := forwardToRelay(ctx, http.MethodPost, "/v1/register", "", fwdBody)
	if upStatus >= 400 {
		publishRelayLifecycleEvent(ctx, "RELAY_REGISTER_FAILED", relayLifecycleEventData(sc, map[string]any{
			"http_status": upStatus,
		}), sc.TaskID)
	} else {
		publishRelayLifecycleEvent(ctx, "RELAY_REGISTER_SUCCEEDED", relayLifecycleEventData(sc, nil), sc.TaskID)
	}
	writeRawJSON(w, upStatus, upBody)
}

func buildRelayStartEnv(sc scope, envOverrides map[string]string, accessToken string, traceID string) map[string]string {
	env := map[string]string{
		"TASK_API_ENDPOINT_ORIGIN":     strings.TrimSpace(cfg.RelayTaskAPIOrigin),
		"BUSINESS_API_ENDPOINT_ORIGIN": strings.TrimSpace(cfg.RelayBusinessAPIOrigin),
	}
	for k, v := range envOverrides {
		key := strings.TrimSpace(k)
		val := strings.TrimSpace(v)
		if key == "" || val == "" {
			continue
		}
		// ACCESS_TOKEN 一律由服务端签发（credentialTokenInit），禁止客户端占位符/明文覆盖。
		if strings.EqualFold(key, "ACCESS_TOKEN") {
			continue
		}
		env[key] = val
	}
	env["ACCESS_TOKEN"] = accessToken
	if traceID != "" {
		env["TRACE_ID"] = traceID
		env["X_TRACE_ID"] = traceID
	}
	return env
}

func handleRelayStart(w http.ResponseWriter, r *http.Request, match relayScope) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	ctx := r.Context()
	bodyMap, _, err := readRequestJSON(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	sc := relayScopeFromBody(bodyMap, match)
	if sc.TaskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "缺少 task_id"})
		return
	}
	_, status, vbody := authorizeContainerRequest(ctx, r, sc)
	if status != http.StatusOK {
		writeRawJSON(w, status, vbody)
		return
	}

	tok, tStatus, tBody := credentialTokenInit(ctx, sc)
	if tStatus != http.StatusOK {
		publishRelayLifecycleEvent(ctx, "RELAY_START_DISPATCH_FAILED", relayLifecycleEventData(sc, map[string]any{
			"error_code": "token_init_failed",
		}), sc.TaskID)
		writeRawJSON(w, tStatus, tBody)
		return
	}

	// 与 Django relay_to_trae_start 对齐：启动前确保 CloudServerConfig 存在，
	// 否则 register-reachability 会因 TOKEN_SCOPE_NOT_FOUND 失败。
	if err := ensureOpenRuntimeSession(ctx, sc); err != nil {
		log.Printf("[taskContainerGateway] open-runtime-session failed task=%s: %v", sc.TaskID, err)
		publishRelayLifecycleEvent(ctx, "RELAY_START_DISPATCH_FAILED", relayLifecycleEventData(sc, map[string]any{
			"error_code": "open_runtime_session_failed",
			"detail":     err.Error(),
		}), sc.TaskID)
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"status":  "error",
			"message": "无法准备运行时会话: " + err.Error(),
		})
		return
	}

	envOverrides := map[string]string{}
	if envRaw, ok := bodyMap["env"].(map[string]any); ok {
		for k, v := range envRaw {
			if s, ok := v.(string); ok {
				envOverrides[k] = s
			}
		}
	}

	imageRef, installedImageID, imgStatus, imgBody := resolveRelayStartImage(ctx, sc.TenantID, bodyMap)
	if imgStatus != http.StatusOK {
		publishRelayLifecycleEvent(ctx, "RELAY_START_DISPATCH_FAILED", relayLifecycleEventData(sc, map[string]any{
			"error_code":  "image_resolve_failed",
			"http_status": imgStatus,
		}), sc.TaskID)
		writeRawJSON(w, imgStatus, imgBody)
		return
	}

	traceID := traceIDFromContext(ctx)
	requestID := newRequestID()
	workflowID := newRequestID()
	runtimeEnv := buildRelayStartEnv(sc, envOverrides, tok.AccessToken, traceID)
	payload := map[string]any{
		"tenant_id":    sc.TenantID,
		"workspace_id": sc.WorkspaceID,
		"task_id":      sc.TaskID,
		"env":          runtimeEnv,
	}
	if imageRef != "" {
		payload["image"] = imageRef
	}
	if installedImageID != "" {
		payload["installed_image_id"] = installedImageID
	}
	fwdBody, _ := json.Marshal(payload)

	publishRelayLifecycleEvent(ctx, "RELAY_START_ACCEPTED", relayLifecycleEventData(sc, map[string]any{
		"request_id":  requestID,
		"workflow_id": workflowID,
	}), requestID)

	// Direct Redis write via Cloud (same keys as converge); reduces Django transition dependency.
	upsertRelayStartupSessionViaCloud(ctx, relayStartupSessionStartAcceptedPayload(sc, workflowID, requestID))

	go func(parent context.Context, body []byte, scope scope, reqID, wfID string) {
		asyncCtx := context.WithoutCancel(parent)
		publishRelayLifecycleEvent(asyncCtx, "RELAY_START_ATTEMPTED", relayLifecycleEventData(scope, map[string]any{
			"request_id":  reqID,
			"workflow_id": wfID,
		}), reqID)
		upStatus, upBody := forwardToRelay(asyncCtx, http.MethodPost, "/v1/start", "", body)
		if upStatus >= 400 {
			publishRelayLifecycleEvent(asyncCtx, "RELAY_START_DISPATCH_FAILED", relayLifecycleEventData(scope, map[string]any{
				"request_id":  reqID,
				"workflow_id": wfID,
				"http_status": upStatus,
			}), reqID)
			publishRelayStatusSSE(asyncCtx, scope.TaskID, map[string]any{
				"running":           false,
				"online_service_up": false,
				"error":             extractRelayError(upBody),
			})
			return
		}
		publishRelayLifecycleEvent(asyncCtx, "RELAY_START_DISPATCH_SUCCEEDED", relayLifecycleEventData(scope, map[string]any{
			"request_id":  reqID,
			"workflow_id": wfID,
		}), reqID)
	}(ctx, fwdBody, sc, requestID, workflowID)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":      "accepted",
		"request_id":  requestID,
		"workflow_id": workflowID,
		"task_id":     sc.TaskID,
	})
}

func handleRelayStop(w http.ResponseWriter, r *http.Request, match relayScope) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	ctx := r.Context()
	sc := scope{TenantID: match.TenantID, WorkspaceID: match.WorkspaceID, TaskID: match.TaskID}
	_, status, vbody := authorizeContainerRequest(ctx, r, sc)
	if status != http.StatusOK {
		writeRawJSON(w, status, vbody)
		return
	}

	publishRelayLifecycleEvent(ctx, "RELAY_STOP_REQUESTED", relayLifecycleEventData(sc, nil), sc.TaskID)

	go func(parent context.Context, scope scope) {
		asyncCtx := context.WithoutCancel(parent)
		upStatus, _ := forwardToRelay(asyncCtx, http.MethodPost, "/v1/stop", "", nil)
		if upStatus >= 400 {
			publishRelayLifecycleEvent(asyncCtx, "RELAY_STOP_FAILED", relayLifecycleEventData(scope, map[string]any{
				"http_status": upStatus,
			}), scope.TaskID)
			return
		}
		publishRelayLifecycleEvent(asyncCtx, "RELAY_STOP_SUCCEEDED", relayLifecycleEventData(scope, map[string]any{
			"reason": "relay_stop",
		}), scope.TaskID)
	}(ctx, sc)

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"message": "stop 已触发，正在异步执行",
	})
}

func relayLifecycleAction(subAction string) (method string, lifecycle bool) {
	switch subAction {
	case "register", "start", "stop", "token-init", "repo-credentials-precheck":
		return http.MethodPost, true
	case "env-prepare":
		return http.MethodGet, true
	default:
		return "", false
	}
}

func relayEnvPreview() map[string]string {
	return map[string]string{
		"TASK_API_ENDPOINT_ORIGIN":     strings.TrimSpace(cfg.RelayTaskAPIOrigin),
		"BUSINESS_API_ENDPOINT_ORIGIN": strings.TrimSpace(cfg.RelayBusinessAPIOrigin),
	}
}

func handleRelayEnvPrepare(w http.ResponseWriter, r *http.Request, match relayScope) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	ctx := r.Context()
	sc := scope{TenantID: match.TenantID, WorkspaceID: match.WorkspaceID, TaskID: match.TaskID}
	if sc.TaskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "缺少 task_id"})
		return
	}
	_, status, vbody := authorizeContainerRequest(ctx, r, sc)
	if status != http.StatusOK {
		writeRawJSON(w, status, vbody)
		return
	}
	env := relayEnvPreview()
	env["ACCESS_TOKEN"] = task2appAccessTokenPlaceholder
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"env":    env,
	})
}

func handleRelayTokenInit(w http.ResponseWriter, r *http.Request, match relayScope) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	ctx := r.Context()
	bodyMap, _, err := readRequestJSON(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	sc := relayScopeFromBody(bodyMap, match)
	if sc.TaskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "缺少 task_id"})
		return
	}
	_, status, vbody := authorizeContainerRequest(ctx, r, sc)
	if status != http.StatusOK {
		writeRawJSON(w, status, vbody)
		return
	}

	publishRelayLifecycleEvent(ctx, "RELAY_TOKEN_INIT_ATTEMPTED", relayLifecycleEventData(sc, nil), sc.TaskID)

	tok, tStatus, tBody := credentialTokenInit(ctx, sc)
	if tStatus != http.StatusOK {
		publishRelayLifecycleEvent(ctx, "RELAY_TOKEN_INIT_FAILED", relayLifecycleEventData(sc, map[string]any{
			"error_code": "go_token_init_failed",
		}), sc.TaskID)
		msg := "无法初始化 access token"
		var parsed map[string]any
		if json.Unmarshal(tBody, &parsed) == nil {
			if d, ok := parsed["detail"].(string); ok && strings.TrimSpace(d) != "" {
				msg = "无法初始化 access token: " + strings.TrimSpace(d)
			}
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"status":  "error",
			"message": msg,
		})
		return
	}
	if strings.TrimSpace(tok.AccessToken) == "" {
		publishRelayLifecycleEvent(ctx, "RELAY_TOKEN_INIT_FAILED", relayLifecycleEventData(sc, map[string]any{
			"error_code": "access_token_missing",
		}), sc.TaskID)
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"status":  "error",
			"message": "无法生成 ACCESS_TOKEN",
		})
		return
	}

	publishRelayLifecycleEvent(ctx, "RELAY_TOKEN_INIT_SUCCEEDED", relayLifecycleEventData(sc, nil), sc.TaskID)
	workflowID := newRequestID()
	upsertRelayStartupSessionViaCloud(ctx, relayStartupSessionTokenInitPayload(sc, workflowID))
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "ok",
		"task_id":           sc.TaskID,
		"token_initialized": true,
		"workflow_id":       workflowID,
		"env_preview":       relayEnvPreview(),
	})
}

func handleRelayRepoCredentialsPrecheck(w http.ResponseWriter, r *http.Request, match relayScope) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
		return
	}
	ctx := r.Context()
	bodyMap, _, err := readRequestJSON(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	sc := relayScopeFromBody(bodyMap, match)
	if sc.TaskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "message": "缺少 task_id"})
		return
	}
	_, status, vbody := authorizeContainerRequest(ctx, r, sc)
	if status != http.StatusOK {
		writeRawJSON(w, status, vbody)
		return
	}

	tok, tStatus, tBody := credentialTokenInit(ctx, sc)
	if tStatus != http.StatusOK {
		msg := "无法获取 access token"
		var parsed map[string]any
		if json.Unmarshal(tBody, &parsed) == nil {
			if d, ok := parsed["detail"].(string); ok && strings.TrimSpace(d) != "" {
				msg = "无法获取 access token: " + strings.TrimSpace(d)
			}
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"status":  "error",
			"message": msg,
		})
		return
	}

	credStatus, credBody := credentialRepoCloneCredentials(ctx, sc, tok.AccessToken)
	var payload map[string]any
	if err := json.Unmarshal(credBody, &payload); err != nil || payload == nil {
		payload = map[string]any{}
	}
	traceID := ""
	if v, ok := payload["trace_id"].(string); ok {
		traceID = strings.TrimSpace(v)
	}

	switch credStatus {
	case http.StatusOK:
		repoCount := 0
		if credRoot, ok := payload["repo_clone_credentials"].(map[string]any); ok {
			repoCount = len(credRoot)
		} else if n, ok := payload["repo_count"].(float64); ok {
			repoCount = int(n)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"message":    "仓库凭证预检通过",
			"repo_count": repoCount,
			"trace_id":   traceID,
		})
	case http.StatusConflict:
		missing, _ := payload["missing_repo_credentials"].([]any)
		if missing == nil {
			missing = []any{}
		}
		detail := strings.TrimSpace(fmt.Sprint(payload["detail"]))
		if detail == "" || detail == "<nil>" {
			detail = "任务仓库克隆凭证不完整"
		}
		errorCode := strings.TrimSpace(fmt.Sprint(payload["error_code"]))
		if errorCode == "" || errorCode == "<nil>" {
			errorCode = "REPO_CLONE_CREDENTIALS_INCOMPLETE"
		}
		writeJSON(w, http.StatusConflict, map[string]any{
			"status":                   "error",
			"message":                  detail,
			"error_code":               errorCode,
			"trace_id":                 traceID,
			"missing_repo_credentials": missing,
		})
	case http.StatusBadGateway:
		tokenFailures, _ := payload["token_refresh_failures"].([]any)
		if tokenFailures == nil {
			tokenFailures = []any{}
		}
		missing, _ := payload["missing_repo_credentials"].([]any)
		if missing == nil {
			missing = []any{}
		}
		// 仅 token 刷新失败（无凭证缺失）时降级为 warning 放行
		if len(tokenFailures) > 0 && len(missing) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"status":                 "ok",
				"message":                "仓库凭证预检通过（部分 token 刷新失败，已降级放行）",
				"repo_count":             0,
				"warning":                "token_refresh_degraded",
				"token_refresh_failures": tokenFailures,
				"trace_id":               traceID,
			})
			return
		}
		detail := strings.TrimSpace(fmt.Sprint(payload["detail"]))
		if detail == "" || detail == "<nil>" {
			detail = "任务仓库克隆 token 换发失败"
		}
		errorCode := strings.TrimSpace(fmt.Sprint(payload["error_code"]))
		if errorCode == "" || errorCode == "<nil>" {
			errorCode = "REPO_CLONE_TOKEN_REFRESH_FAILED"
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"status":                   "error",
			"message":                  detail,
			"error_code":               errorCode,
			"trace_id":                 traceID,
			"token_refresh_failures":   tokenFailures,
			"missing_repo_credentials": missing,
		})
	default:
		detail := strings.TrimSpace(fmt.Sprint(payload["detail"]))
		if detail == "" || detail == "<nil>" {
			detail = strings.TrimSpace(fmt.Sprint(payload["message"]))
		}
		if detail == "" || detail == "<nil>" {
			detail = fmt.Sprintf("仓库凭证预检失败（HTTP %d）", credStatus)
		}
		errorCode := strings.TrimSpace(fmt.Sprint(payload["error_code"]))
		if errorCode == "" || errorCode == "<nil>" {
			errorCode = "REPO_CLONE_CREDENTIALS_FAILED"
		}
		outStatus := http.StatusBadGateway
		if credStatus >= 400 && credStatus < 500 {
			outStatus = credStatus
		}
		writeJSON(w, outStatus, map[string]any{
			"status":     "error",
			"message":    detail,
			"error_code": errorCode,
			"trace_id":   traceID,
		})
	}
}

func extractRelayError(body []byte) string {
	if len(body) == 0 {
		return "relay start failed"
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return strings.TrimSpace(string(body))
	}
	for _, key := range []string{"message", "detail", "error"} {
		if v, ok := parsed[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return "relay start failed"
}
