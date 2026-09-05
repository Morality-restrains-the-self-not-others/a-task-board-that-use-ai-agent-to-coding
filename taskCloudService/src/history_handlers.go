package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func handleInternalHistories(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/cloud-server-config/histories")
	path = strings.Trim(path, "/")
	switch {
	case path == "" || path == "/":
		switch r.Method {
		case http.MethodGet:
			handleInternalListHistories(w, r)
		case http.MethodPost:
			handleInternalCreateHistory(w, r)
		default:
			writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		}
	case path == "close-open" && r.Method == http.MethodPost:
		handleInternalCloseOpenHistories(w, r)
	default:
		if strings.HasSuffix(path, "/") {
			path = strings.TrimSuffix(path, "/")
		}
		if r.Method == http.MethodPatch || r.Method == http.MethodPut {
			handleInternalPatchHistory(w, r, path)
			return
		}
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
	}
}

func handleInternalListHistories(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	workspaceID := r.URL.Query().Get("workspace_id")
	taskID := r.URL.Query().Get("task_id")
	openOnly := r.URL.Query().Get("open_only") == "1"
	items, err := listCloudServerConfigHistoriesFiltered(tenantID, workspaceID, taskID, openOnly)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"histories": items, "count": len(items)})
}

func handleInternalCreateHistory(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	h := historyFromBody(body)
	if h.ID == "" {
		h.ID = genID("csh")
	}
	if err := upsertCloudServerConfigHistory(h); err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cloudServerConfigHistoryToJSON(&h))
}

func handleInternalPatchHistory(w http.ResponseWriter, r *http.Request, historyID string) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	h, err := loadCloudServerConfigHistoryByID(historyID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusNotFound, "history not found")
		return
	}
	applyHistoryPatch(h, body)
	if err := upsertCloudServerConfigHistory(*h); err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cloudServerConfigHistoryToJSON(h))
}

func handleInternalCloseOpenHistories(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	tenantID := strField(body, "tenant_id")
	workspaceID := strField(body, "workspace_id")
	taskID := strField(body, "task_id")
	instanceID := strField(body, "instance_id")
	reason := strField(body, "stop_reason")
	if reason == "" {
		reason = "superseded_by_new_start"
	}
	patch := map[string]interface{}{}
	for _, key := range []string{"server_url", "business_api_endpoint", "public_ip", "stopped_at", "stop_reason"} {
		if v, ok := body[key]; ok {
			patch[key] = v
		}
	}
	if _, ok := patch["stop_reason"]; !ok {
		patch["stop_reason"] = reason
	}
	count, err := closeOpenCloudServerConfigHistories(tenantID, workspaceID, taskID, patch, instanceID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "closed": count})
}

func historyFromBody(body map[string]interface{}) CloudServerConfigHistory {
	h := CloudServerConfigHistory{
		ID:                  strField(body, "id"),
		CompanyID:           strField(body, "company_id"),
		WorkspaceID:         strField(body, "workspace_id"),
		TaskID:              strField(body, "task_id"),
		Platform:            defaultStr(strField(body, "platform"), "mock"),
		PlatformID:          intField(body, "platform_id", 1),
		InstanceID:          strField(body, "instance_id"),
		InstanceTypeID:      strField(body, "instance_type_id"),
		SecurityGroupID:     strField(body, "security_group_id"),
		VswitchID:           strField(body, "vswitch_id"),
		Region:              strField(body, "region"),
		ZoneID:              strField(body, "zone_id"),
		AuthorizationID:     strField(body, "authorization_id"),
		PublicIP:            strField(body, "public_ip"),
		ServerURL:           strField(body, "server_url"),
		BusinessAPIEndpoint: strField(body, "business_api_endpoint"),
		ErrorReason:         strField(body, "error_reason"),
		StopReason:          strField(body, "stop_reason"),
		RuntimeSource:       strField(body, "runtime_source"),
		LaunchRequestID:     strField(body, "launch_request_id"),
		CpuCores:            intField(body, "cpu_cores", 1),
		MemoryGB:            intField(body, "memory_gb", 1),
		StorageGB:           intField(body, "storage_gb", 40),
	}
	if spec, ok := getCachedInstanceTypeSpec(strings.TrimSpace(h.InstanceTypeID)); ok && spec.cpuCores > 0 {
		h.CpuCores = spec.cpuCores
		if m := memoryGBFromInstanceTypeSpec(spec); m > 0 {
			h.MemoryGB = m
		}
	}
	return h
}

func applyHistoryPatch(h *CloudServerConfigHistory, body map[string]interface{}) {
	for _, pair := range []struct {
		key string
		dst *string
	}{
		{"server_url", &h.ServerURL},
		{"business_api_endpoint", &h.BusinessAPIEndpoint},
		{"public_ip", &h.PublicIP},
		{"stop_reason", &h.StopReason},
		{"instance_id", &h.InstanceID},
		{"error_reason", &h.ErrorReason},
	} {
		if v, ok := body[pair.key]; ok {
			*pair.dst = trim(fmt.Sprintf("%v", v))
		}
	}
	if v, ok := body["stopped_at"]; ok && v != nil && fmt.Sprintf("%v", v) != "" {
		s := trim(fmt.Sprintf("%v", v))
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			h.StoppedAt = &t
		} else if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
			h.StoppedAt = &t
		}
	}
}

func intField(body map[string]interface{}, key string, defaultVal int) int {
	v, ok := body[key]
	if !ok || v == nil {
		return defaultVal
	}
	var n int
	if _, err := fmt.Sscanf(fmt.Sprintf("%v", v), "%d", &n); err != nil {
		return defaultVal
	}
	return n
}
