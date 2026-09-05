package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// reconcileLeakedServerURLs scans cloud_server_configs for rows where
// terminal_released=1 but server_url is still populated — a "leaked" server
// that was never released due to the CSC-not-found race condition in the
// task_status_changed handler.
//
// For each leaked row it clears the server_url and stops the sidecars
// (relay), then marks the history row as closed.
func reconcileLeakedServerURLs(now time.Time) (cleaned int, err error) {
	rows, qerr := db.Query(
		`SELECT company_id, workspace_id, task_id, server_url, instance_id
		 FROM cloud_server_configs
		 WHERE COALESCE(terminal_released,0) = 1
		   AND COALESCE(server_url,'') != ''
		   AND COALESCE(last_runtime_status,'') != 'Released'`,
	)
	if qerr != nil {
		return 0, qerr
	}
	defer rows.Close()

	type leakedRow struct {
		CompanyID   string
		WorkspaceID string
		TaskID      string
		ServerURL   string
		InstanceID  string
	}

	var leaked []leakedRow
	for rows.Next() {
		var lr leakedRow
		if err := rows.Scan(&lr.CompanyID, &lr.WorkspaceID, &lr.TaskID, &lr.ServerURL, &lr.InstanceID); err != nil {
			continue
		}
		leaked = append(leaked, lr)
	}

	for _, lr := range leaked {
		taskID := strings.TrimSpace(lr.TaskID)
		serverURL := strings.TrimSpace(lr.ServerURL)
		companyID := strings.TrimSpace(lr.CompanyID)
		workspaceID := strings.TrimSpace(lr.WorkspaceID)

		log.Printf("[server-release-reconcile] found leaked server task_id=%s server_url=%s",
			taskID, serverURL)

		// Stop sidecars (relay) that may still be running for this task.
		// Soft-fail: if sidecar stop fails, still clear the CSC to prevent
		// repeated reconciliation attempts.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := stopRelaySidecarForReconcile(ctx, taskID); err != nil {
			log.Printf("[server-release-reconcile] relay stop failed task_id=%s: %v (continue)", taskID, err)
		}
		cancel()

		// Clear the CSC: remove server_url, instance_id, close history sessions.
		// 必须带 instanceID 精确命中泄漏行；空串会退化为「任务最新 CSC」，
		// 任务级泄漏行会被静默跳过（OPT-20260816-028 修复）。
		if _, err := clearContainerReachabilityNative(companyID, workspaceID, taskID, "reconcile_leaked", lr.InstanceID); err != nil {
			log.Printf("[server-release-reconcile] clear CSC failed task_id=%s: %v", taskID, err)
			continue
		}

		// Assert terminal_released stays set to prevent rediscovery.
		_, _ = db.Exec(
			`UPDATE cloud_server_configs SET terminal_released=1, updated_at=? WHERE company_id=? AND workspace_id=? AND task_id=?`,
			now.UTC(), companyID, workspaceID, taskID,
		)

		log.Printf("[server-release-reconcile] cleaned leaked server task_id=%s server_url=%s",
			taskID, serverURL)
		cleaned++
	}

	return cleaned, nil
}

// stopRelaySidecarForReconcile POSTs /v1/stop to go_relayToTrae.
// Soft-fail: errors are logged but not returned as fatal.
func stopRelaySidecarForReconcile(ctx context.Context, taskID string) error {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_URL")), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("TASK_RELAY_TO_TRAE_URL")), "/")
	}
	if base == "" {
		base = "http://127.0.0.1:8797"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/stop", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if secret := strings.TrimSpace(os.Getenv("RELAY_TO_TRAE_SECRET")); secret != "" {
		req.Header.Set("X-Relay-To-Trae-Secret", secret)
	}
	if strings.TrimSpace(taskID) != "" {
		req.Header.Set("X-Task-Id", taskID)
	}
	resp, err := djangoHTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("relay sidecar stop HTTP %d", resp.StatusCode)
	}
	return nil
}

// handleInternalReconcileLeakedServers 供 taskEvents cloud_csc_reconcile timer 一次性触发
// （OPT-20260816-028）。业务进程不再自带 ticker；由 timer worker 批量 POST 驱动。
func handleInternalReconcileLeakedServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	cleaned, err := reconcileLeakedServerURLs(time.Now().UTC())
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{
			"status": "error", "message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"cleaned": cleaned,
	})
}
