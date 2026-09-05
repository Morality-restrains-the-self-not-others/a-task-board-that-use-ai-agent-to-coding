package main

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"
)

func handleCloudTaskRoutes(w http.ResponseWriter, r *http.Request, cloudSubPath string) {
	tenantID := getAuthTenant(r)
	workspaceID := r.Header.Get("X-Workspace-Id")
	taskID := r.Header.Get("X-Task-Id")
	sub := strings.Trim(strings.TrimPrefix(cloudSubPath, "cloud/"), "/")

	switch {
	case strings.HasPrefix(sub, "server-container-token/"):
		action := strings.Trim(strings.TrimPrefix(sub, "server-container-token/"), "/")
		handleContainerInboundToken(w, r, tenantID, workspaceID, taskID, action)
	case sub == "relay-to-trae/status-push" || strings.HasPrefix(sub, "relay-to-trae/status-push/"):
		handleContainerInboundScoped(w, r, tenantID, workspaceID, taskID, "relay-status-push")
	case sub == "model-budget-usage" || strings.HasPrefix(sub, "model-budget-usage/"):
		handleContainerInboundScoped(w, r, tenantID, workspaceID, taskID, "model-budget-usage")
	case sub == "server-config" || strings.HasPrefix(sub, "server-config/"):
		handleServerConfigRoutes(w, r, tenantID, workspaceID, taskID)
	case strings.HasPrefix(sub, "server-userdata-verify/"):
		handleServerUserdataVerify(w, r, tenantID, workspaceID, taskID, strings.TrimPrefix(sub, "server-userdata-verify/"))
	case sub == "repo-reclone" || strings.HasPrefix(sub, "repo-reclone/"):
		handleRepoReclone(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/container-task-ui-context" || strings.HasPrefix(sub, "compute/container-task-ui-context/"):
		handleContainerTaskUIContext(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/ensure-client-ingress" || strings.HasPrefix(sub, "compute/ensure-client-ingress/"):
		handleEnsureClientIngress(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/server-start-history" || strings.HasPrefix(sub, "compute/server-start-history/"):
		handleServerStartHistory(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/previous-server-config" || strings.HasPrefix(sub, "compute/previous-server-config/"):
		handlePreviousServerConfig(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/workspace-runtime-indicators" || strings.HasPrefix(sub, "compute/workspace-runtime-indicators/"):
		handleWorkspaceRuntimeIndicators(w, r, tenantID, workspaceID)
	case sub == "compute/workspace-machine-policy" || strings.HasPrefix(sub, "compute/workspace-machine-policy/"):
		handleWorkspaceMachinePolicy(w, r, tenantID, workspaceID)
	case sub == "compute/workspace-machine-summary" || strings.HasPrefix(sub, "compute/workspace-machine-summary/"):
		handleWorkspaceMachineSummary(w, r, tenantID, workspaceID)
	case sub == "compute/comment-container-bindings" || strings.HasPrefix(sub, "compute/comment-container-bindings/"):
		if taskID == "" {
			taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
		}
		rest := strings.TrimPrefix(sub, "compute/comment-container-bindings")
		rest = strings.TrimPrefix(rest, "/")
		handleCommentContainerBindingsRoutes(w, r, tenantID, taskID, workspaceID, rest)
	case sub == "compute/server-runtime-status" || strings.HasPrefix(sub, "compute/server-runtime-status/"):
		handleServerRuntimeStatus(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/server-startup-status" || strings.HasPrefix(sub, "compute/server-startup-status/"):
		handleServerStartupStatusPoll(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/server-content" || strings.HasPrefix(sub, "compute/server-content/"):
		handleServerContent(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/github-credential-status" || strings.HasPrefix(sub, "compute/github-credential-status/"):
		handleGithubCredentialStatus(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/github-credential-approve" || strings.HasPrefix(sub, "compute/github-credential-approve/"):
		handleGithubCredentialApprove(w, r, tenantID, workspaceID, taskID)
	case sub == "model-budgets" || strings.HasPrefix(sub, "model-budgets/"):
		// v64: /api/cloud/ 约定下恢复任务级 model-budgets[/raise]（ead5583 移除 /api/tenant/ 时未重新挂载）
		rest := strings.Trim(strings.TrimPrefix(sub, "model-budgets"), "/")
		if rest == "raise" {
			handleUserRaiseTaskModelBudget(w, r, tenantID, workspaceID, taskID)
		} else {
			handleUserTaskModelBudgets(w, r, tenantID, workspaceID, taskID)
		}
	case sub == "compute/workbench-link" || strings.HasPrefix(sub, "compute/workbench-link/"):
		handleWorkbenchLink(w, r, tenantID, workspaceID, taskID)
	case sub == "compute/feature-params-env-preview" || strings.HasPrefix(sub, "compute/feature-params-env-preview/"):
		handleFeatureParamsEnvPreview(w, r, tenantID, workspaceID, taskID)
	case isContainerLayerGraphSub(sub):
		handleContainerLayerGraphFromDB(w, r, tenantID, workspaceID, taskID)
	case isContainerJobExecutionLogSub(sub):
		handleContainerJobExecutionLogFromDB(w, r, tenantID, workspaceID, taskID)
	case isContainerOutboundComputeSub(sub):
		proxyContainerGatewayRequest(w, r)
	case sub == "compute/start-vm" || strings.HasPrefix(sub, "compute/start-vm/"):
		handleStartVmNative(w, r, tenantID, workspaceID)
	case sub == "compute/start-vm-auto" || strings.HasPrefix(sub, "compute/start-vm-auto/"):
		handleStartVmAutoNative(w, r, tenantID, workspaceID)
	case sub == "compute/stop-vm" || strings.HasPrefix(sub, "compute/stop-vm/"):
		handleStopVmNative(w, r, tenantID, workspaceID)
	case strings.HasPrefix(sub, "compute/"):
		logWarn("compute action not yet ported: "+sub, r.Header.Get("X-Trace-Id"))
		writeErrorMapJSON(w, r, http.StatusNotImplemented, map[string]interface{}{
			"status":  "error",
			"message": "compute action not yet ported to taskCloudService: " + sub,
		})
	default:
		writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{"error": "task cloud route not found", "path": sub})
	}
}

func handleServerUserdataVerify(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID, secret string) {
	secret = strings.TrimSpace(strings.TrimSuffix(secret, "/"))
	if secret == "" || len(secret) > 64 {
		writeErrorJSON(w, r, http.StatusNotFound, "invalid token")
		return
	}
	ok, err := verifyCloudServerUserdata(tenantID, workspaceID, taskID, secret)
	if err != nil {
		logWarn("userdata verify db error: "+err.Error(), r.Header.Get("X-Trace-Id"))
		writeErrorJSON(w, r, http.StatusInternalServerError, "verification failed")
		return
	}
	if !ok {
		writeErrorJSON(w, r, http.StatusNotFound, "invalid or consumed token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleContainerTaskUIContext(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少任务ID"})
		return
	}
	if tenantID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "无法获取租户ID"})
		return
	}
	if workspaceID == "" {
		workspaceID = resolveWorkspaceIDForTask(tenantID, taskID)
	}
	if workspaceID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "无法获取工作区ID"})
		return
	}
	commentID := commentIDFromComputeRequest(r, nil)
	if rejectMissingComputeCommentID(w, r, commentID) {
		return
	}
	payload := buildContainerTaskUIContextScoped(tenantID, workspaceID, taskID, commentID)
	writeJSON(w, http.StatusOK, payload)
}

func buildContainerTaskUIContext(tenantID, workspaceID, taskID string) map[string]interface{} {
	return buildContainerTaskUIContextScoped(tenantID, workspaceID, taskID, "")
}

func buildContainerTaskUIContextScoped(tenantID, workspaceID, taskID, commentID string) map[string]interface{} {
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		counts := loadTaskRunningCounts(tenantID, workspaceID, taskID)
		return map[string]interface{}{
			"status":                        "success",
			"has_server_config":             false,
			"container_endpoint_registered": false,
			"container_page_url":            "",
			"container_vscode_url":          "",
			"running_machine_count":         counts.Machines,
			"running_container_count":       counts.Containers,
		}
	}
	cfg, err := resolveScopedCloudServerConfig(tenantID, workspaceID, taskID, commentID)
	if err != nil || cfg == nil {
		return map[string]interface{}{
			"status":                        "success",
			"has_server_config":             false,
			"container_endpoint_registered": false,
			"container_page_url":            "",
			"container_vscode_url":          "",
		}
	}
	serverURL := strings.TrimSpace(cfg.ServerURL)
	status := effectiveMachineRuntimeStatus(cfg.InstanceID, cfg.LastRuntimeStatus)
	// Align with kanban container_running: orphan server_url after stop/release is not "registered".
	registered := containerReachabilityCountsAsRunning(cfg.InstanceID, status, serverURL)
	if serverURL != "" && !registered {
		_, _ = clearContainerReachabilityOnConfig(cfg, cfg.CompanyID, cfg.WorkspaceID, cfg.TaskID)
		return map[string]interface{}{
			"status":                        "success",
			"has_server_config":             true,
			"container_endpoint_registered": false,
			"container_page_url":            "",
			"container_vscode_url":          "",
		}
	}
	pageURL := buildContainerPageURL(cfg)
	vscodeURL := buildContainerVscodeURL(cfg)
	// Check if container was recently connected (cold-open "双向已连接" hint, OPT-20260719-034).
	lastHeartbeat := ""
	if cfg.ID != "" {
		_ = db.QueryRow(`SELECT COALESCE(last_heartbeat_at,'') FROM cloud_server_configs WHERE id=?`, cfg.ID).Scan(&lastHeartbeat)
	}
	return map[string]interface{}{
		"status":                        "success",
		"has_server_config":             true,
		"container_endpoint_registered": registered,
		"container_page_url":            pageURL,
		"container_vscode_url":          vscodeURL,
		"last_heartbeat_at":             lastHeartbeat,
	}
}

func buildContainerPageURL(cfg *CloudServerConfig) string {
	if cfg == nil {
		return ""
	}
	_, token := resolveContainerTarget(cfg, "")
	if token == "" {
		return ""
	}
	origin := businessOrigin(cfg.BusinessAPIEndpoint)
	if origin == "" {
		return ""
	}
	safeTok := url.PathEscape(token)
	tenant := strings.TrimSpace(cfg.CompanyID)
	workspace := strings.TrimSpace(cfg.WorkspaceID)
	task := strings.TrimSpace(cfg.TaskID)
	if tenant != "" && workspace != "" && task != "" {
		return origin +
			"/ui/tenant/" + url.PathEscape(tenant) +
			"/workspace/" + url.PathEscape(workspace) +
			"/task/" + url.PathEscape(task) +
			"/" + safeTok
	}
	return origin + "/ui/" + safeTok
}

func buildContainerVscodeURL(cfg *CloudServerConfig) string {
	if cfg == nil {
		return ""
	}
	stored := strings.TrimSpace(cfg.ContainerVscodeURL)
	if stored != "" {
		if u, err := url.Parse(stored); err == nil && u.Scheme != "" && u.Host != "" {
			return stored
		}
	}
	origin := businessOrigin(cfg.BusinessAPIEndpoint)
	if origin == "" {
		return ""
	}
	return origin + "/"
}

func businessOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	return scheme + "://" + u.Host
}

func handleServerStartHistory(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	items, err := listCloudServerConfigHistories(tenantID, workspaceID, taskID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "获取历史服务器启动记录成功",
		"total":   len(items),
		"records": items,
	})
}

func handlePreviousServerConfig(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	item, err := loadLatestCloudServerConfigHistory(tenantID, workspaceID, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status":        "success",
				"message":       "未找到服务器配置历史记录",
				"server_config": nil,
			})
			return
		}
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"server_config": enrichPreviousServerConfig(tenantID, previousServerConfigFromHistory(item), item),
	})
}

func previousServerConfigFromHistory(h *CloudServerConfigHistory) map[string]interface{} {
	if h == nil {
		return nil
	}
	cpu, mem := resolveHistoryCPUMemory(h)
	return map[string]interface{}{
		"platform":          h.Platform,
		"platform_id":       h.PlatformID,
		"instance_id":       h.InstanceID,
		"instance_type_id":  h.InstanceTypeID,
		"security_group_id": h.SecurityGroupID,
		"vswitch_id":        h.VswitchID,
		"region":            h.Region,
		"zone_id":           h.ZoneID,
		"workspace_id":      h.WorkspaceID,
		"authorization_id":  h.AuthorizationID,
		"error_reason":      h.ErrorReason,
		"created_at":        h.CreatedAt,
		"hardware_config": map[string]interface{}{
			"cpu_cores":  cpu,
			"memory_gb":  mem,
			"storage_gb": h.StorageGB,
		},
	}
}

func handleServerStartupStatusPoll(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if taskID == "" {
		taskID = strings.TrimSpace(r.URL.Query().Get("task_id"))
	}
	if taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少任务ID"})
		return
	}
	if tenantID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "无法获取租户ID"})
		return
	}
	if workspaceID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "缺少租户或工作区上下文"})
		return
	}

	queryEventID := strings.TrimSpace(r.URL.Query().Get("event_id"))
	queryCommentID := strings.TrimSpace(r.URL.Query().Get("comment_id"))
	ev, err := loadLatestStartEvent(tenantID, taskID, queryEventID, queryCommentID)
	if err != nil {
		if queryEventID != "" {
			writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{
				"status":   "error",
				"message":  "未找到对应的启动事件",
				"event_id": queryEventID,
			})
			return
		}
		writeErrorMapJSON(w, r, http.StatusNotFound, map[string]interface{}{
			"status":  "error",
			"message": "任务ID不存在",
		})
		return
	}

	writeJSON(w, http.StatusOK, statusPayloadFromEvent(ev, tenantID, taskID))
}
