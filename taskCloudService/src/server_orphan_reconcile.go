package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// reconcileOrphanTaskCSCs scans cloud_server_configs for rows where instance_id is
// non-empty but the corresponding task_id no longer exists in the task service. These
// "orphan" rows occur when a task is deleted without cascading the CSC cleanup.
//
// Orphan rows are marked terminal_released=1; the existing reconcileLeakedServerURLs
// ticker then handles the sidecar stop + server_url clear.
//
// OPT-20260816-027：任务存在性改走 taskTaskService 批量 HTTP API，禁止直连 task-task 库。
func reconcileOrphanTaskCSCs(now time.Time) (cleaned int, err error) {
	// Find CSC rows with active instances that are not already terminal_released.
	rows, err := db.Query(
		`SELECT id, company_id, workspace_id, task_id, instance_id
		 FROM cloud_server_configs
		 WHERE COALESCE(instance_id,'') != ''
		   AND COALESCE(terminal_released,0) != 1
		   AND COALESCE(last_runtime_status,'') != 'Released'`,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type orphanCandidate struct {
		ID          string
		CompanyID   string
		WorkspaceID string
		TaskID      string
		InstanceID  string
	}

	var candidates []orphanCandidate
	for rows.Next() {
		var c orphanCandidate
		if err := rows.Scan(&c.ID, &c.CompanyID, &c.WorkspaceID, &c.TaskID, &c.InstanceID); err != nil {
			continue
		}
		if strings.TrimSpace(c.TaskID) == "" {
			continue
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(candidates) == 0 {
		return 0, nil
	}

	// 批量向 taskTaskService 查一次任务存在性（替代逐行直连 task-task 库）。
	taskIDs := make([]string, 0, len(candidates))
	seen := map[string]bool{}
	for _, c := range candidates {
		tid := strings.TrimSpace(c.TaskID)
		if tid == "" || seen[tid] {
			continue
		}
		seen[tid] = true
		taskIDs = append(taskIDs, tid)
	}
	exists, err := taskIDsExistHTTP(taskIDs)
	if err != nil {
		return 0, err
	}

	for _, c := range candidates {
		taskID := strings.TrimSpace(c.TaskID)
		if exists[taskID] {
			continue // Task still exists — not an orphan.
		}

		// Task deleted — mark CSC terminal_released so reconcileLeakedServerURLs cleans it.
		if _, err := db.Exec(
			`UPDATE cloud_server_configs SET terminal_released=1, updated_at=? WHERE id=?`,
			now.UTC(), c.ID,
		); err != nil {
			logInfo(fmt.Sprintf("orphan reconcile: mark terminal_released failed id=%s task_id=%s: %v",
				c.ID, taskID, err), "")
			continue
		}

		logInfo(fmt.Sprintf("orphan reconcile: marked terminal_released=1 id=%s task_id=%s instance_id=%s (task deleted)",
			c.ID, taskID, c.InstanceID), "")
		cleaned++
	}

	return cleaned, nil
}

// taskIDsExistHTTP asks taskTaskService which of the given task_ids still exist.
// 单库所有权：孤儿对账不再跨库直连 task-task，改走 /api/internal/tasks/exists/。
func taskIDsExistHTTP(taskIDs []string) (map[string]bool, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if base == "" {
		return nil, fmt.Errorf("task service url not configured")
	}
	clean := make([]string, 0, len(taskIDs))
	for _, id := range taskIDs {
		if tid := strings.TrimSpace(id); tid != "" {
			clean = append(clean, tid)
		}
	}
	if len(clean) == 0 {
		return map[string]bool{}, nil
	}
	payload, _ := json.Marshal(map[string]interface{}{"task_ids": clean})
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/tasks/exists/", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if v := strings.TrimSpace(cfg.InternalSecret); v != "" {
		req.Header.Set("X-Internal-Secret", v)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("task exists status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var body struct {
		Exists map[string]bool `json:"exists"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if body.Exists == nil {
		return map[string]bool{}, nil
	}
	return body.Exists, nil
}

// handleInternalReconcileOrphanCSC 供 taskEvents cloud_csc_reconcile timer 一次性触发
// （OPT-20260816-027）。OPT-20260816-028 已移除进程内 ticker。
func handleInternalReconcileOrphanCSC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	cleaned, err := reconcileOrphanTaskCSCs(time.Now().UTC())
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	terminalCleaned, terr := reconcileTerminalTaskCSCs(time.Now().UTC())
	if terr != nil {
		logWarn(fmt.Sprintf("event=reconcile_terminal_csc_failed err=%v orphan_cleaned=%d", terr, cleaned), "")
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":           "success",
		"cleaned":          cleaned,
		"terminal_cleaned": terminalCleaned,
	})
}
