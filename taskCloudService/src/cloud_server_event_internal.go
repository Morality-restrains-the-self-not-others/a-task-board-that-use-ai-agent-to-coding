package main

import (
	"errors"
	"net/http"
	"strings"
)

func handleInternalCloudServerEventRoutes(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/cloud-server-events/")
	path = strings.Trim(path, "/")

	switch {
	case path == "latest-pending-start" && r.Method == http.MethodGet:
		handleInternalLatestPendingStart(w, r)
	case path == "status" && r.Method == http.MethodGet:
		handleInternalCloudServerEventStatus(w, r)
	case path == "update-status" && r.Method == http.MethodPost:
		handleInternalUpdateCloudServerEventStatus(w, r)
	case path == "update-data" && r.Method == http.MethodPost:
		handleInternalUpdateCloudServerEventData(w, r)
	case path == "import" && r.Method == http.MethodPost:
		handleInternalImportCloudServerEvents(w, r)
	default:
		writeErrorJSON(w, r, http.StatusNotFound, "not found")
	}
}

// handleInternalCloudServerEventStatus 按 event_id 返回启动事件状态（OPT-20260818-015）：
// 消费者在 LatestPendingStartEvent 未命中时据此判断「已成功处理」从而幂等跳过重放。
func handleInternalCloudServerEventStatus(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(r.URL.Query().Get("event_id"))
	if eventID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "event_id required")
		return
	}
	ev, err := loadCloudServerEvent(eventID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusNotFound, "cloud server event not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"event_id": ev.ID, "status": ev.Status})
}

func handleInternalLatestPendingStart(w http.ResponseWriter, r *http.Request) {
	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	if companyID == "" || taskID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "company_id and task_id required")
		return
	}
	id, status, err := latestPendingStartEvent(companyID, taskID)
	if err != nil {
		writeErrorJSON(w, r, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": status})
}

func handleInternalUpdateCloudServerEventStatus(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	eventID := strField(body, "event_id")
	status := strField(body, "status")
	errMsg := strField(body, "error_message")
	fromStatus := strField(body, "from_status")
	if eventID == "" || status == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "event_id and status required")
		return
	}
	if err := updateCloudServerEventStatus(eventID, status, errMsg, fromStatus); err != nil {
		if errors.Is(err, errCloudServerEventStatusConflict) {
			// OPT-20260818-015 剩余 gap：并发消费者同时 claim 时，仅一方能把
			// pending→processing 原子迁移成功；另一方收到 409 即幂等跳过。
			writeErrorJSON(w, r, http.StatusConflict, "event status changed")
			return
		}
		writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func handleInternalUpdateCloudServerEventData(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	eventID := strField(body, "event_id")
	rawData, ok := body["event_data"].(map[string]interface{})
	if eventID == "" || !ok {
		writeErrorJSON(w, r, http.StatusBadRequest, "event_id and event_data required")
		return
	}
	if err := updateCloudServerEventData(eventID, rawData); err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func handleInternalImportCloudServerEvents(w http.ResponseWriter, r *http.Request) {
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	rawEvents, _ := body["events"].([]interface{})
	if len(rawEvents) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "imported": 0})
		return
	}
	rows := make([]map[string]interface{}, 0, len(rawEvents))
	for _, item := range rawEvents {
		if m, ok := item.(map[string]interface{}); ok {
			rows = append(rows, m)
		}
	}
	count, err := importCloudServerEvents(rows)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "imported": count})
}

func statusPayloadFromEvent(ev *CloudServerEvent, companyID, taskID string) map[string]interface{} {
	if ev == nil {
		return map[string]interface{}{
			"status": "error", "message": "未找到对应的启动事件", "progress": 0,
		}
	}
	switch ev.Status {
	case cloudServerEventStatusPending:
		return map[string]interface{}{
			"status":       "processing",
			"message":      "服务器启动请求已提交，等待处理...",
			"progress":     60,
			"event_id":     ev.ID,
			"event_status": ev.Status,
		}
	case cloudServerEventStatusProcessing:
		return map[string]interface{}{
			"status":       "processing",
			"message":      "服务器正在启动中...",
			"progress":     80,
			"event_id":     ev.ID,
			"event_status": ev.Status,
		}
	case cloudServerEventStatusSuccess:
		vmInfo := map[string]interface{}{
			"instance_id": "",
			"public_ip":   "",
			"server_url":  "",
		}
		if cfg, err := loadCloudServerConfig(companyID, ev.WorkspaceID, taskID); err == nil && cfg != nil {
			vmInfo["instance_id"] = cfg.InstanceID
			vmInfo["public_ip"] = cfg.PublicIP
			vmInfo["server_url"] = cfg.ServerURL
		}
		return map[string]interface{}{
			"status":         "success",
			"message":        "服务器启动成功！",
			"progress":       100,
			"event_id":       ev.ID,
			"event_status":   ev.Status,
			"vm_info":        vmInfo,
			"runtime_status": "Running",
		}
	case cloudServerEventStatusError:
		msg := strings.TrimSpace(ev.ErrorMessage)
		if msg == "" {
			msg = "服务器启动失败"
		}
		return map[string]interface{}{
			"status":       "error",
			"message":      msg,
			"progress":     0,
			"event_id":     ev.ID,
			"event_status": ev.Status,
		}
	default:
		return map[string]interface{}{
			"status":       "processing",
			"message":      "等待服务器启动...",
			"progress":     50,
			"event_id":     ev.ID,
			"event_status": ev.Status,
		}
	}
}
