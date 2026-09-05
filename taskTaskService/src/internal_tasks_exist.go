package main

import (
	"fmt"
	"net/http"
	"strings"
)

// handleInternalTasksExist serves POST /api/internal/tasks/exists/
// Body: {"task_ids": ["id1","id2",...]}
// Returns: {"exists": {"id1": true, "id2": false, ...}}
// Max 500 task_ids per request。
// 供 taskCloudService 孤儿 CSC 对账按任务存在性打标，替代跨库直连（OPT-20260816-027）。
func handleInternalTasksExist(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, 200, map[string]interface{}{"exists": map[string]interface{}{}})
		return
	}

	// 去重并限制 500。
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

	exists := map[string]interface{}{}
	for _, id := range ids {
		exists[id] = false
	}
	if len(ids) > 0 {
		placeholders := make([]string, len(ids))
		args := make([]interface{}, len(ids))
		for i, id := range ids {
			placeholders[i] = "?"
			args[i] = id
		}
		rows, err := db.Query(`SELECT id FROM task_tasks WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
		if err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				continue
			}
			exists[id] = true
		}
		if err := rows.Err(); err != nil {
			writeError(w, r, 500, err.Error())
			return
		}
	}
	writeJSON(w, 200, map[string]interface{}{"exists": exists})
}
