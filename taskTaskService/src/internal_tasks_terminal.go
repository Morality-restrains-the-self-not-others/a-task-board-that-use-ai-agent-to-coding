package main

import (
	"fmt"
	"net/http"
	"strings"
)

// handleInternalTasksTerminalKinds serves POST /api/internal/tasks/terminal-kinds/
// Body: {"task_ids": ["id1","id2",...]}
// Returns: {"kinds": {"id1": "cancelled", "id2": "", "id3": "completed"}}
// Empty kind = task exists but is not terminal. Missing key = task not found.
// Max 500 task_ids per request.
// 供 taskCloudService 入站守卫 / 终态 CSC 对账：取消或完成后仍有心跳时释放机器。
func handleInternalTasksTerminalKinds(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, 400, "invalid JSON")
		return
	}
	rawIDs, _ := body["task_ids"].([]interface{})
	if len(rawIDs) == 0 {
		writeJSON(w, 200, map[string]interface{}{"kinds": map[string]interface{}{}})
		return
	}

	ids := make([]string, 0, len(rawIDs))
	seen := map[string]bool{}
	for _, raw := range rawIDs {
		tid := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if tid == "" || tid == "<nil>" || seen[tid] {
			continue
		}
		seen[tid] = true
		ids = append(ids, tid)
		if len(ids) >= 500 {
			break
		}
	}

	kinds, err := lookupTaskTerminalKinds(ids)
	if err != nil {
		writeError(w, r, 500, err.Error())
		return
	}
	out := make(map[string]interface{}, len(kinds))
	for id, kind := range kinds {
		out[id] = kind
	}
	writeJSON(w, 200, map[string]interface{}{"kinds": out})
}

type taskTerminalRow struct {
	ID               string
	TenantID         string
	WorkspaceID      string
	ProgressColumnID string
	Completed        bool
}

func lookupTaskTerminalKinds(ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := db.Query(
		`SELECT id, tenant_id, workspace_id, COALESCE(progress_column_id,''), completed
		 FROM task_tasks WHERE id IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []taskTerminalRow
	groups := map[string][]string{} // tenant/workspace → column IDs
	for rows.Next() {
		var rec taskTerminalRow
		var completed int
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.WorkspaceID, &rec.ProgressColumnID, &completed); err != nil {
			return nil, err
		}
		rec.Completed = completed != 0
		recs = append(recs, rec)
		col := strings.TrimSpace(rec.ProgressColumnID)
		if col == "" {
			continue
		}
		gkey := rec.TenantID + "/" + rec.WorkspaceID
		groups[gkey] = append(groups[gkey], col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	namesByGroup := map[string]map[string]string{}
	for gkey, cols := range groups {
		parts := strings.SplitN(gkey, "/", 2)
		tenantID, workspaceID := parts[0], ""
		if len(parts) > 1 {
			workspaceID = parts[1]
		}
		namesByGroup[gkey] = progressColumnNameByIDFn(tenantID, workspaceID, cols)
	}

	for _, rec := range recs {
		gkey := rec.TenantID + "/" + rec.WorkspaceID
		name := ""
		if names := namesByGroup[gkey]; names != nil {
			name = names[rec.ProgressColumnID]
		}
		out[rec.ID] = resolveTerminalKind(name, rec.Completed)
	}
	return out, nil
}
