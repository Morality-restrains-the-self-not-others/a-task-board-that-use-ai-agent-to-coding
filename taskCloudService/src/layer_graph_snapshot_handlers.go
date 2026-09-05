package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"tracelog"
)

func isContainerLayerGraphSub(sub string) bool {
	s := strings.Trim(sub, "/")
	return s == "compute/container-layer-graph" ||
		strings.HasPrefix(s, "compute/container-layer-graph/") ||
		strings.Contains(s, "/container-layer-graph/") ||
		strings.HasSuffix(s, "/container-layer-graph")
}

func pathKV(path, key string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == key {
			return strings.TrimSpace(parts[i+1])
		}
	}
	return ""
}

func resolveLayerGraphScope(r *http.Request, headerWorkspace, headerTask string) (workspaceID, taskID, commentID string) {
	workspaceID = pathKV(r.URL.Path, "workspace_id")
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(headerWorkspace)
	}
	taskID = pathKV(r.URL.Path, "task_id")
	if taskID == "" {
		taskID = strings.TrimSpace(headerTask)
	}
	commentID = commentIDFromComputeRequest(r, nil)
	return workspaceID, taskID, commentID
}

func handleContainerLayerGraphFromDB(w http.ResponseWriter, r *http.Request, tenantID, workspaceID, taskID string) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed, use GET")
		return
	}
	ws, task, commentID := resolveLayerGraphScope(r, workspaceID, taskID)
	if strings.TrimSpace(tenantID) == "" {
		tenantID = pathKV(r.URL.Path, "tenant_id")
	}
	if ws == "" || task == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "workspace_id and task_id are required")
		return
	}
	if commentID == "" {
		writeErrorJSON(w, r, http.StatusBadRequest, "缺少评论ID")
		return
	}
	row, found, err := getLayerGraphSnapshot(ws, task, commentID)
	if err != nil {
		tracelog.LogForwardStage(r.Context(), "layer_graph_snapshot_get_err", map[string]any{
			"error": err.Error(), "workspace_id": ws, "task_id": task, "comment_id": commentID,
		})
		writeErrorJSON(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if found && strings.TrimSpace(row.GraphJSON) != "" {
		var doc map[string]any
		if err := json.Unmarshal([]byte(row.GraphJSON), &doc); err == nil && layerGraphDocHasLayers(doc) {
			ensureGitPrRepliesFromGraphDoc(r.Context(), tenantID, ws, task, commentID, doc)
			doc["source"] = "saas_db"
			writeJSON(w, http.StatusOK, doc)
			return
		}
	}
	// 快照尚未落库（容器未 PUSH / persist 失败）时回退 live Gateway，避免前端空树。
	tracelog.LogForwardStage(r.Context(), "layer_graph_snapshot_live_fallback", map[string]any{
		"workspace_id": ws, "task_id": task, "comment_id": commentID, "found": found,
	})
	proxyContainerGatewayRequest(w, r)
}

func layerGraphJSONEqual(a, b string) bool {
	var da, db any
	if err := json.Unmarshal([]byte(a), &da); err != nil {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	if err := json.Unmarshal([]byte(b), &db); err != nil {
		return false
	}
	ra, errA := json.Marshal(da)
	rb, errB := json.Marshal(db)
	if errA != nil || errB != nil {
		return false
	}
	return string(ra) == string(rb)
}

func layerGraphDocHasLayers(doc map[string]any) bool {
	if doc == nil {
		return false
	}
	layers, ok := doc["layers"].([]any)
	return ok && len(layers) > 0
}

var publishLayerGraphSnapshotPersisted = func(ctx context.Context, data map[string]interface{}, key string) error {
	return publishDomainEvent(ctx, "LayerGraphSnapshotPersisted", data, key)
}

func persistLayerGraphPush(ctx context.Context, cfgRow *CloudServerConfig, layers, jobs []any, extra map[string]any, tenantID, workspaceID, taskID string) error {
	commentID := ""
	companyID := strings.TrimSpace(tenantID)
	wsID := strings.TrimSpace(workspaceID)
	if cfgRow != nil {
		commentID = strings.TrimSpace(cfgRow.CommentID)
		if c := strings.TrimSpace(cfgRow.CompanyID); c != "" {
			companyID = c
		}
		if w := strings.TrimSpace(cfgRow.WorkspaceID); w != "" {
			wsID = w
		}
		if strings.TrimSpace(taskID) == "" {
			taskID = strings.TrimSpace(cfgRow.TaskID)
		}
	}
	raw, err := marshalLayerGraphDocument(layers, jobs, extra)
	if err != nil {
		return err
	}
	existing, found, err := getLayerGraphSnapshot(wsID, taskID, commentID)
	if err != nil {
		return err
	}
	if found && layerGraphJSONEqual(existing.GraphJSON, raw) {
		tracelog.LogForwardStage(ctx, "layer_graph_snapshot_unchanged", map[string]any{
			"workspace_id": wsID, "task_id": taskID, "comment_id": commentID,
		})
		return nil
	}
	if err := upsertLayerGraphSnapshot(layerGraphSnapshotRow{
		CompanyID: companyID, WorkspaceID: wsID, TaskID: taskID, CommentID: commentID, GraphJSON: raw,
	}); err != nil {
		return err
	}
	eventKey := wsID + ":" + taskID + ":" + commentID
	if err := publishLayerGraphSnapshotPersisted(ctx, map[string]interface{}{
		"workspace_id": wsID,
		"task_id":      taskID,
		"comment_id":   commentID,
		"company_id":   companyID,
	}, eventKey); err != nil {
		tracelog.LogForwardStage(ctx, "layer_graph_snapshot_event_err", map[string]any{
			"error": err.Error(), "workspace_id": wsID, "task_id": taskID, "comment_id": commentID,
		})
	}
	return nil
}
