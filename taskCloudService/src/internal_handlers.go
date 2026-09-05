package main

import (
	"net/http"
	"strings"
)

func requireInternalSecret(r *http.Request) bool {
	if cfg.InternalSecret == "" {
		return true
	}
	return r.Header.Get("X-Internal-Secret") == cfg.InternalSecret
}

func handleInternalCloudServerConfig(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/cloud-server-config/")
	path = strings.Trim(path, "/")

	if strings.HasPrefix(path, "histories") {
		handleInternalHistories(w, r)
		return
	}

	switch {
	case path == "lookup" && r.Method == http.MethodGet:
		handleInternalLookupConfig(w, r)
	case path == "list-by-task" && r.Method == http.MethodGet:
		handleInternalListByTask(w, r)
	case path == "container-target" && r.Method == http.MethodGet:
		handleInternalContainerTarget(w, r)
	case path == "validate-ai-comment-post" && r.Method == http.MethodPost:
		handleInternalValidateAICommentPost(w, r)
	case path == "import" && r.Method == http.MethodPost:
		handleInternalImportConfigs(w, r)
	case path == "import-histories" && r.Method == http.MethodPost:
		handleInternalImportHistories(w, r)
	case path == "clear-after-stop" && r.Method == http.MethodPost:
		handleInternalClearAfterStop(w, r)
	case path == "container-unreachable" && r.Method == http.MethodPost:
		handleInternalContainerUnreachable(w, r)
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
	}
}

func handleInternalLookupConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	workspaceID := r.URL.Query().Get("workspace_id")
	taskID := r.URL.Query().Get("task_id")
	commentID := strings.TrimSpace(r.URL.Query().Get("comment_id"))
	cscID := strings.TrimSpace(r.URL.Query().Get("csc_id"))
	cfg, err := loadCloudServerConfigForGatewayForward(tenantID, workspaceID, taskID, commentID, cscID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusNotFound, "cloud server config not found")
		return
	}
	writeJSON(w, http.StatusOK, cloudServerConfigToJSON(cfg))
}

func handleInternalContainerTarget(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	workspaceID := r.URL.Query().Get("workspace_id")
	taskID := r.URL.Query().Get("task_id")
	traceID := r.URL.Query().Get("trace_id")
	overrideURL := strings.TrimSpace(r.URL.Query().Get("container_page_url"))
	commentID := strings.TrimSpace(r.URL.Query().Get("comment_id"))
	cscID := strings.TrimSpace(r.URL.Query().Get("csc_id"))
	cfg, err := loadCloudServerConfigForGatewayForward(tenantID, workspaceID, taskID, commentID, cscID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusNotFound, "cloud server config not found")
		return
	}
	baseURL, token, tokErrCode, tokErrDetail, credStatus := resolveContainerTargetDiag(cfg, overrideURL)
	if baseURL == "" {
		cscIDLog := ""
		cmtLog := ""
		if cfg != nil {
			cscIDLog = cfg.ID
			cmtLog = cfg.CommentID
		}
		logWarn("event=container_target_missing_base_url task_id="+taskID+" comment_id="+cmtLog+" csc_id="+cscIDLog, traceID)
		writeErrorJSON(w, r, http.StatusConflict, "容器尚未注册可用业务地址，请先完成启动与 exchange-refresh")
		return
	}
	if token == "" {
		body := map[string]interface{}{"detail": "缺少容器 access_token"}
		if tokErrCode != "" {
			body["error_code"] = tokErrCode
		}
		if tokErrDetail != "" {
			body["credential_detail"] = tokErrDetail
		}
		if credStatus > 0 {
			body["credential_status"] = credStatus
		}
		writeJSON(w, http.StatusConflict, body)
		return
	}
	streamBackend := detectStreamBackend(baseURL, token, traceID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"base_url":              baseURL,
		"access_token":          token,
		"stream_backend":        streamBackend,
		"server_url":            cfg.ServerURL,
		"business_api_endpoint": cfg.BusinessAPIEndpoint,
	})
}

func handleInternalImportHistories(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	rawItems, _ := body["histories"].([]interface{})
	if len(rawItems) == 0 {
		writeErrorJSON(w, r, http.StatusBadRequest, "histories required")
		return
	}
	rows := make([]CloudServerConfigHistory, 0, len(rawItems))
	for _, item := range rawItems {
		m, ok := item.(map[string]interface{})
		if !ok {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid history row")
			return
		}
		rows = append(rows, CloudServerConfigHistory{
			ID:                  strField(m, "id"),
			CompanyID:           strField(m, "company_id"),
			WorkspaceID:         strField(m, "workspace_id"),
			TaskID:              strField(m, "task_id"),
			Platform:            strField(m, "platform"),
			PlatformID:          intField(m, "platform_id", 1),
			InstanceID:          strField(m, "instance_id"),
			InstanceTypeID:      strField(m, "instance_type_id"),
			SecurityGroupID:     strField(m, "security_group_id"),
			VswitchID:           strField(m, "vswitch_id"),
			Region:              strField(m, "region"),
			ZoneID:              strField(m, "zone_id"),
			AuthorizationID:     strField(m, "authorization_id"),
			PublicIP:            strField(m, "public_ip"),
			ServerURL:           strField(m, "server_url"),
			BusinessAPIEndpoint: strField(m, "business_api_endpoint"),
			ErrorReason:         strField(m, "error_reason"),
			StopReason:          strField(m, "stop_reason"),
			RuntimeSource:       strField(m, "runtime_source"),
			LaunchRequestID:     strField(m, "launch_request_id"),
			CpuCores:            intField(m, "cpu_cores", 1),
			MemoryGB:            intField(m, "memory_gb", 1),
			StorageGB:           intField(m, "storage_gb", 40),
		})
	}
	count, err := importCloudServerConfigHistories(rows)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "imported": count})
}

func handleInternalValidateAICommentPost(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Task-Test-Skip-AI-Comment-Validate") == "1" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	taskID := strField(body, "task_id")
	traceID := strField(body, "trace_id")
	if traceID == "" {
		traceID = r.Header.Get("X-Trace-Id")
	}
	if tenantID == "" || workspaceID == "" || taskID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "tenant_id, workspace_id, task_id required")
		return
	}
	status, payload := validateAICommentPost(tenantID, workspaceID, taskID, body, traceID)
	writeJSON(w, status, payload)
}

func handleInternalImportConfigs(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	rawItems, _ := body["configs"].([]interface{})
	if len(rawItems) == 0 {
		if alt, ok := body["rows"].([]interface{}); ok {
			rawItems = alt
		}
	}
	if len(rawItems) == 0 {
		writeErrorJSON(w, r, http.StatusBadRequest, "configs required")
		return
	}
	rows := make([]CloudServerConfig, 0, len(rawItems))
	for _, item := range rawItems {
		m, ok := item.(map[string]interface{})
		if !ok {
			writeErrorJSON(w, r, http.StatusBadRequest, "invalid config row")
			return
		}
		cfgRow := CloudServerConfig{
			ID:                  strField(m, "id"),
			CompanyID:           strField(m, "company_id"),
			WorkspaceID:         strField(m, "workspace_id"),
			TaskID:              strField(m, "task_id"),
			CommentID:           strField(m, "comment_id"),
			Platform:            strField(m, "platform"),
			InstanceID:          strField(m, "instance_id"),
			SecurityGroupID:     strField(m, "security_group_id"),
			VswitchID:           strField(m, "vswitch_id"),
			Region:              strField(m, "region"),
			ZoneID:              strField(m, "zone_id"),
			AuthorizationID:     strField(m, "authorization_id"),
			PublicIP:            strField(m, "public_ip"),
			ServerURL:           strField(m, "server_url"),
			BusinessAPIEndpoint: strField(m, "business_api_endpoint"),
			ContainerVscodeURL:  strField(m, "container_vscode_url"),
			ErrorReason:         strField(m, "error_reason"),
			LaunchRequestID:     strField(m, "launch_request_id"),
			ClientToken:         strField(m, "client_token"),
		}
		if strings.TrimSpace(cfgRow.InstanceID) != "" {
			syncLastRuntimeStatusOnInstanceChange(&cfgRow, "")
		}
		rows = append(rows, cfgRow)
	}
	count, err := importCloudServerConfigs(rows)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "imported": count})
}
