package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
)

func handleServerConfigRoutes(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/"), "/")
	switch {
	case strings.HasSuffix(path, "server-config") || strings.HasSuffix(path, "server-config/"):
		switch r.Method {
		case http.MethodGet:
			handleGetServerConfig(w, r, tenantID, workspaceID, taskID)
		case http.MethodPost:
			handleCreateServerConfig(w, r, tenantID, workspaceID, taskID)
		case http.MethodPatch, http.MethodPut:
			handlePatchServerConfig(w, r, tenantID, workspaceID, taskID)
		default:
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
	case strings.Contains(path, "server-config/histories"):
		if r.Method != http.MethodGet {
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		handleServerStartHistory(w, r, tenantID, workspaceID, taskID)
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
	}
}

func handleGetServerConfig(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	cfg, err := loadCloudServerConfig(tenantID, workspaceID, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErrorJSON(w, r, http.StatusNotFound, "cloud server config not found")
			return
		}
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cloudServerConfigToJSON(cfg))
}

func handleCreateServerConfig(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	if taskID == "" {
		taskID = strField(body, "task_id")
	}
	if workspaceID == "" {
		workspaceID = strField(body, "workspace_id")
	}
	cfg := cloudConfigFromBody(body, tenantID, workspaceID, taskID)
	if cfg.ID == "" {
		cfg.ID = genID("csc")
	}
	if cfg.CompanyID == "" {
		cfg.CompanyID = tenantID
	}
	if err := upsertCloudServerConfig(cfg); err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cloudServerConfigToJSON(&cfg))
}

func handlePatchServerConfig(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	cfg, err := loadCloudServerConfig(tenantID, workspaceID, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErrorJSON(w, r, http.StatusNotFound, "cloud server config not found")
			return
		}
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	prevServerURL := cfg.ServerURL
	applyPatchFields(cfg, body)
	// OPT-20260814-017: PATCH 携带 comment_id 即评论级写入意图；落库前用
	// commentCSCCloudMetaIllegal 校验三元组，拒绝 mock/空 platform、空 region/auth，防脏写。
	if commentID := strings.TrimSpace(strField(body, "comment_id")); commentID != "" && commentCSCCloudMetaIllegal(cfg) {
		writeErrorJSON(w, r, http.StatusBadRequest, "comment-level cloud server config requires legal platform/region/authorization_id")
		return
	}
	if err := upsertCloudServerConfig(*cfg); err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	_ = maybeMarkIdleOnServerURLClear(cfg, prevServerURL)
	_ = maybeClearIdleOnServerURLSet(cfg, prevServerURL)
	writeJSON(w, http.StatusOK, cloudServerConfigToJSON(cfg))
}

func cloudConfigFromBody(body map[string]interface{}, tenantID, workspaceID, taskID string) CloudServerConfig {
	cfg := CloudServerConfig{
		ID:                  strField(body, "id"),
		CompanyID:           firstNonEmpty(strField(body, "company_id"), tenantID),
		WorkspaceID:         firstNonEmpty(strField(body, "workspace_id"), workspaceID),
		TaskID:              firstNonEmpty(strField(body, "task_id"), taskID),
		Platform:            defaultStr(strField(body, "platform"), "aliyun"),
		InstanceID:          strField(body, "instance_id"),
		SecurityGroupID:     strField(body, "security_group_id"),
		VswitchID:           strField(body, "vswitch_id"),
		Region:              strField(body, "region"),
		ZoneID:              strField(body, "zone_id"),
		AuthorizationID:     strField(body, "authorization_id"),
		PublicIP:            strField(body, "public_ip"),
		ServerURL:           strField(body, "server_url"),
		BusinessAPIEndpoint: strField(body, "business_api_endpoint"),
		ContainerVscodeURL:  strField(body, "container_vscode_url"),
		ErrorReason:         strField(body, "error_reason"),
		LaunchRequestID:     strField(body, "launch_request_id"),
		ClientToken:         strField(body, "client_token"),
		LastRuntimeStatus:   strField(body, "last_runtime_status"),
	}
	syncLastRuntimeStatusOnInstanceChange(&cfg, "")
	return cfg
}

func applyPatchFields(cfg *CloudServerConfig, body map[string]interface{}) {
	prevInstanceID := strings.TrimSpace(cfg.InstanceID)
	for _, pair := range []struct {
		key string
		dst *string
	}{
		{"platform", &cfg.Platform},
		{"instance_id", &cfg.InstanceID},
		{"security_group_id", &cfg.SecurityGroupID},
		{"vswitch_id", &cfg.VswitchID},
		{"region", &cfg.Region},
		{"zone_id", &cfg.ZoneID},
		{"authorization_id", &cfg.AuthorizationID},
		{"public_ip", &cfg.PublicIP},
		{"server_url", &cfg.ServerURL},
		{"business_api_endpoint", &cfg.BusinessAPIEndpoint},
		{"container_vscode_url", &cfg.ContainerVscodeURL},
		{"error_reason", &cfg.ErrorReason},
		{"launch_request_id", &cfg.LaunchRequestID},
		{"client_token", &cfg.ClientToken},
		{"last_runtime_status", &cfg.LastRuntimeStatus},
	} {
		if v, ok := body[pair.key]; ok {
			*pair.dst = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	syncLastRuntimeStatusOnInstanceChange(cfg, prevInstanceID)
}

// syncLastRuntimeStatusOnInstanceChange sets Starting/Running defaults when instance_id is assigned.
func syncLastRuntimeStatusOnInstanceChange(cfg *CloudServerConfig, prevInstanceID string) {
	if cfg == nil {
		return
	}
	cur := strings.TrimSpace(cfg.InstanceID)
	if cur == "" {
		cfg.LastRuntimeStatus = ""
		return
	}
	if isMockMachineInstanceID(cur) {
		cfg.LastRuntimeStatus = machineRuntimeRunning
		return
	}
	if cur != prevInstanceID && normalizeMachineRuntimeStatus(cfg.LastRuntimeStatus) == "" {
		cfg.LastRuntimeStatus = machineRuntimeStarting
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func defaultStr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}
