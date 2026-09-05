package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// instanceBindingRow is one CSC row sharing an instance_id.
type instanceBindingRow struct {
	TaskID              string `json:"task_id"`
	InstanceID          string `json:"instance_id"`
	ServerURL           string `json:"server_url"`
	BusinessAPIEndpoint string `json:"business_api_endpoint"`
	LastRuntimeStatus   string `json:"last_runtime_status"`
	TerminalReleased    int    `json:"terminal_released"`
	PublicIP            string `json:"public_ip"`
}

func listInstanceBindings(companyID, workspaceID, instanceID string) ([]instanceBindingRow, error) {
	companyID = strings.TrimSpace(companyID)
	workspaceID = strings.TrimSpace(workspaceID)
	instanceID = strings.TrimSpace(instanceID)
	if companyID == "" || workspaceID == "" || instanceID == "" {
		return nil, fmt.Errorf("company_id, workspace_id, instance_id required")
	}
	rows, err := db.Query(
		`SELECT task_id, COALESCE(instance_id,''), COALESCE(server_url,''),
			COALESCE(business_api_endpoint,''), COALESCE(last_runtime_status,''),
			COALESCE(terminal_released,0), COALESCE(public_ip,'')
		 FROM cloud_server_configs
		 WHERE company_id=? AND workspace_id=? AND TRIM(COALESCE(instance_id,''))=?`,
		companyID, workspaceID, instanceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []instanceBindingRow
	for rows.Next() {
		var r instanceBindingRow
		if err := rows.Scan(&r.TaskID, &r.InstanceID, &r.ServerURL, &r.BusinessAPIEndpoint,
			&r.LastRuntimeStatus, &r.TerminalReleased, &r.PublicIP); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func markTerminalReleased(companyID, workspaceID, taskID string) error {
	companyID = strings.TrimSpace(companyID)
	workspaceID = strings.TrimSpace(workspaceID)
	taskID = strings.TrimSpace(taskID)
	if companyID == "" || taskID == "" {
		return fmt.Errorf("company_id and task_id required")
	}
	if workspaceID == "" {
		return fmt.Errorf("workspace_id required")
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	// Comment-scoped CSCs hold instance_id/server_url; the empty-comment template
	// row is not a running instance. Clear every row for this task so work-panel
	// indicators (instance_id + last_runtime_status) go dark after 已取消/已完成.
	_, err := db.Exec(
		`UPDATE cloud_server_configs
		 SET terminal_released=1,
		     instance_id='',
		     public_ip='',
		     server_url='',
		     business_api_endpoint='',
		     container_vscode_url='',
		     last_runtime_status='Released',
		     error_reason='',
		     instruction_idle_since=NULL,
		     updated_at=?
		 WHERE company_id=? AND workspace_id=? AND task_id=?`,
		now, companyID, workspaceID, taskID,
	)
	if err != nil {
		return err
	}
	// OPT-20260822-001: 任务终态时把该任务仍处于调度/运行态的评论绑定同步为 released，
	// 避免 CSC 已 Released 而 cloud_comment_container_bindings.status 仍为 running，
	// 详情页继续显示「容器 运行中」并误点「提交并创建PR」。已终态（completed/failed/
	// cancelled/released）行保持不变，保留各自语义。
	if _, bindErr := db.Exec(
		`UPDATE cloud_comment_container_bindings
		 SET status=?, updated_at=?
		 WHERE company_id=? AND workspace_id=? AND task_id=?
		   AND status IN (?,?,?,?)`,
		ccbStatusReleased, now, companyID, workspaceID, taskID,
		ccbStatusPending, ccbStatusWaitingPrevious, ccbStatusStarting, ccbStatusRunning,
	); bindErr != nil {
		logWarn(fmt.Sprintf("event=terminal_binding_sync_failed task_id=%s err=%v", taskID, bindErr), taskID)
	}
	_ = clearConfigIdleSince(companyID, workspaceID, taskID)
	logInfo("event=terminal_hard_release_marked tenant="+companyID+" workspace="+workspaceID+" task_id="+taskID, taskID)
	return nil
}

func setTerminalReleasedFlag(companyID, workspaceID, taskID string) error {
	companyID = strings.TrimSpace(companyID)
	workspaceID = strings.TrimSpace(workspaceID)
	taskID = strings.TrimSpace(taskID)
	if companyID == "" || workspaceID == "" || taskID == "" {
		return fmt.Errorf("company_id, workspace_id, task_id required")
	}
	_, err := db.Exec(
		`UPDATE cloud_server_configs SET terminal_released=1, updated_at=?
		 WHERE company_id=? AND workspace_id=? AND task_id=?`,
		time.Now().UTC().Format("2006-01-02 15:04:05"), companyID, workspaceID, taskID,
	)
	return err
}

// clearIdleBindingsOnInstance clears CSC rows that share instanceID but have no container (idle),
// excluding excludeTaskID. Prevents reuse of a machine about to be hard-stopped.
func clearIdleBindingsOnInstance(companyID, workspaceID, instanceID, excludeTaskID string) (int, error) {
	rows, err := listInstanceBindings(companyID, workspaceID, instanceID)
	if err != nil {
		return 0, err
	}
	cleared := 0
	for _, b := range rows {
		if b.TaskID == excludeTaskID {
			continue
		}
		if strings.TrimSpace(b.ServerURL) != "" {
			continue
		}
		if b.TerminalReleased != 0 {
			continue
		}
		cfg, err := loadCloudServerConfig(companyID, workspaceID, b.TaskID)
		if err != nil || cfg == nil {
			continue
		}
		cfg.InstanceID = ""
		cfg.PublicIP = ""
		cfg.ServerURL = ""
		cfg.BusinessAPIEndpoint = ""
		cfg.ContainerVscodeURL = ""
		cfg.LastRuntimeStatus = ""
		if err := upsertCloudServerConfig(*cfg); err != nil {
			return cleared, err
		}
		_ = clearConfigIdleSince(companyID, workspaceID, b.TaskID)
		cleared++
		logInfo("event=clear_idle_sibling_before_hard_release tenant="+companyID+
			" workspace="+workspaceID+" task_id="+b.TaskID+" instance_id="+instanceID, b.TaskID)
	}
	return cleared, nil
}

func handleInternalClearIdleBindingsOnInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	companyID := strings.TrimSpace(firstNonEmpty(strField(body, "company_id"), strField(body, "tenant_id")))
	workspaceID := strings.TrimSpace(strField(body, "workspace_id"))
	instanceID := strings.TrimSpace(strField(body, "instance_id"))
	excludeTaskID := strings.TrimSpace(strField(body, "exclude_task_id"))
	if companyID == "" || workspaceID == "" || instanceID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "tenant_id/company_id, workspace_id, instance_id required",
		})
		return
	}
	n, err := clearIdleBindingsOnInstance(companyID, workspaceID, instanceID, excludeTaskID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "cleared": n})
}

func handleInternalSetTerminalReleasedFlag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	companyID := strings.TrimSpace(firstNonEmpty(strField(body, "company_id"), strField(body, "tenant_id")))
	workspaceID := strings.TrimSpace(strField(body, "workspace_id"))
	taskID := strings.TrimSpace(strField(body, "task_id"))
	if companyID == "" || workspaceID == "" || taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "tenant_id/company_id, workspace_id, task_id required",
		})
		return
	}
	if err := setTerminalReleasedFlag(companyID, workspaceID, taskID); err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "terminal_released": true})
}

// migrateContainerOffInstance moves a busy owner task off fromInstanceID onto a dedicated machine node.
// - Real ECS (non-mock from): full start-vm-auto (UserData boots image). Cross-task idle reuse removed (ADR-0013).
// - Mock/local: allocate mock-migrated-* then restart container via gateway with original image.
func migrateContainerOffInstance(companyID, workspaceID, ownerTaskID, fromInstanceID string) (string, error) {
	res, err := migrateContainerOffInstanceDetailed(companyID, workspaceID, ownerTaskID, fromInstanceID)
	if err != nil {
		return "", err
	}
	return res.InstanceID, nil
}

func migrateContainerOffInstanceDetailed(companyID, workspaceID, ownerTaskID, fromInstanceID string) (migrateOffResult, error) {
	companyID = strings.TrimSpace(companyID)
	workspaceID = strings.TrimSpace(workspaceID)
	ownerTaskID = strings.TrimSpace(ownerTaskID)
	fromInstanceID = strings.TrimSpace(fromInstanceID)
	if companyID == "" || workspaceID == "" || ownerTaskID == "" || fromInstanceID == "" {
		return migrateOffResult{}, fmt.Errorf("company_id, workspace_id, owner_task_id, from_instance_id required")
	}

	owner, err := loadCloudServerConfig(companyID, workspaceID, ownerTaskID)
	if err != nil {
		return migrateOffResult{}, err
	}
	if owner == nil {
		return migrateOffResult{}, fmt.Errorf("owner cloud_server_config not found for task_id=%s", ownerTaskID)
	}
	hadContainer := strings.TrimSpace(owner.ServerURL) != ""
	imageID, imageURL := extractContainerImageHints(companyID, ownerTaskID)
	curInst := strings.TrimSpace(owner.InstanceID)
	if curInst != "" && curInst != fromInstanceID {
		logInfo("event=container_migrate_idempotent tenant="+companyID+" workspace="+workspaceID+
			" owner_task="+ownerTaskID+" instance_id="+curInst, ownerTaskID)
		return migrateOffResult{InstanceID: curInst, Via: "idempotent", ContainerImageID: imageID}, nil
	}

	// 跨任务闲置复用已移除（ADR-0013）；迁机一律冷启动新实例。
	if !isMockMachineInstanceID(fromInstanceID) {
		newID, perr := provisionMigratedRealECSFn(companyID, workspaceID, ownerTaskID, fromInstanceID, imageID, owner)
		if perr != nil {
			return migrateOffResult{}, perr
		}
		logInfo("event=container_migrated tenant="+companyID+" workspace="+workspaceID+
			" owner_task="+ownerTaskID+" from_instance="+fromInstanceID+
			" to_instance="+newID+" via=start_vm_auto had_container="+fmt.Sprint(hadContainer), ownerTaskID)
		publishContainerMigrateAwaitReadyFn(companyID, workspaceID, ownerTaskID, newID, "start_vm_auto", imageID, imageURL)
		return migrateOffResult{
			InstanceID:              newID,
			Via:                     "start_vm_auto",
			ContainerImageID:        imageID,
			ContainerStartTriggered: true, // UserData boots container
		}, nil
	}

	// Mock / local: dedicated mock node + explicit container restart.
	newInstanceID := "mock-migrated-" + ownerTaskID
	owner.InstanceID = newInstanceID
	owner.PublicIP = ""
	owner.ServerURL = ""
	owner.BusinessAPIEndpoint = ""
	owner.ContainerVscodeURL = ""
	owner.LastRuntimeStatus = machineRuntimeRunning
	owner.ErrorReason = ""
	if err := upsertCloudServerConfig(*owner); err != nil {
		return migrateOffResult{}, err
	}
	_ = clearConfigIdleSince(companyID, workspaceID, ownerTaskID)
	_, _ = db.Exec(
		`UPDATE cloud_server_configs SET terminal_released=0 WHERE company_id=? AND workspace_id=? AND task_id=?`,
		companyID, workspaceID, ownerTaskID,
	)
	out := migrateOffResult{InstanceID: newInstanceID, Via: "new_mock", ContainerImageID: imageID}
	if hadContainer {
		if serr := startMigratedContainerFn(companyID, workspaceID, ownerTaskID, imageID, imageURL); serr != nil {
			logInfo("event=container_migrate_start_failed tenant="+companyID+" workspace="+workspaceID+
				" owner_task="+ownerTaskID+" err="+serr.Error(), ownerTaskID)
			return out, fmt.Errorf("mock node ready but container start failed: %w", serr)
		}
		out.ContainerStartTriggered = true
		publishContainerMigrateAwaitReadyFn(companyID, workspaceID, ownerTaskID, newInstanceID, out.Via, imageID, imageURL)
	}
	logInfo("event=container_migrated tenant="+companyID+" workspace="+workspaceID+
		" owner_task="+ownerTaskID+" from_instance="+fromInstanceID+
		" to_instance="+newInstanceID+" via=new_mock had_container="+fmt.Sprint(hadContainer)+
		" container_started="+fmt.Sprint(out.ContainerStartTriggered), ownerTaskID)
	return out, nil
}

func handleInternalInstanceBindings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	q := r.URL.Query()
	companyID := strings.TrimSpace(firstNonEmpty(q.Get("company_id"), q.Get("tenant_id")))
	workspaceID := strings.TrimSpace(q.Get("workspace_id"))
	instanceID := strings.TrimSpace(q.Get("instance_id"))
	if companyID == "" || workspaceID == "" || instanceID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status": "error", "message": "company_id/tenant_id, workspace_id, instance_id required",
		})
		return
	}
	rows, err := listInstanceBindings(companyID, workspaceID, instanceID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	if rows == nil {
		rows = []instanceBindingRow{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "bindings": rows})
}

func handleInternalMarkTerminalReleased(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	companyID := strings.TrimSpace(firstNonEmpty(strField(body, "company_id"), strField(body, "tenant_id")))
	workspaceID := strings.TrimSpace(strField(body, "workspace_id"))
	taskID := strings.TrimSpace(strField(body, "task_id"))
	if companyID == "" || taskID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "tenant_id/company_id and task_id required"})
		return
	}
	if err := markTerminalReleased(companyID, workspaceID, taskID); err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "terminal_released": true})
}

// markCommentTerminalReleased marks a comment-scoped CSC as terminal_released=1 +
// last_runtime_status='Released'. Used when a task goes terminal while RunInstances is
// still in flight (launch_request_id set, no instance_id yet) so a late-arriving VM is
// not left as an orphan (OPT-20260817-027).
func handleInternalMarkCommentTerminalReleased(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	companyID := strings.TrimSpace(firstNonEmpty(strField(body, "company_id"), strField(body, "tenant_id")))
	workspaceID := strings.TrimSpace(strField(body, "workspace_id"))
	taskID := strings.TrimSpace(strField(body, "task_id"))
	commentID := strings.TrimSpace(strField(body, "comment_id"))
	if companyID == "" || taskID == "" || commentID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{"status": "error", "message": "tenant_id/company_id, task_id and comment_id required"})
		return
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	// OPT-20260821-021: 与 markTerminalReleased 同一套清字段——评论级终态打标时
	// instance_id/public_ip/server_url 一并清空，避免 Starting 晚到 VM 补偿窗口再按 instance 计「已启动」。
	res, err := db.Exec(
		`UPDATE cloud_server_configs SET terminal_released=1,
		     instance_id='', public_ip='', server_url='', business_api_endpoint='',
		     container_vscode_url='', last_runtime_status='Released', error_reason='', updated_at=?
		 WHERE company_id=? AND workspace_id=? AND task_id=? AND comment_id=?`,
		now, companyID, workspaceID, taskID, commentID,
	)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	affected, _ := res.RowsAffected()
	logInfo(fmt.Sprintf("event=mark_comment_terminal_released task_id=%s comment_id=%s affected=%d",
		taskID, commentID, affected), "")
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "terminal_released": true, "affected": affected})
}

func handleInternalMigrateContainerOffInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	companyID := strings.TrimSpace(firstNonEmpty(strField(body, "company_id"), strField(body, "tenant_id")))
	workspaceID := strings.TrimSpace(strField(body, "workspace_id"))
	ownerTaskID := strings.TrimSpace(firstNonEmpty(strField(body, "owner_task_id"), strField(body, "task_id")))
	fromInstanceID := strings.TrimSpace(firstNonEmpty(strField(body, "from_instance_id"), strField(body, "forbid_reuse_instance_id")))
	if companyID == "" || workspaceID == "" || ownerTaskID == "" || fromInstanceID == "" {
		writeErrorMapJSON(w, r, http.StatusBadRequest, map[string]interface{}{
			"status":  "error",
			"message": "tenant_id/company_id, workspace_id, owner_task_id, from_instance_id required",
		})
		return
	}
	detailed, err := migrateContainerOffInstanceDetailed(companyID, workspaceID, ownerTaskID, fromInstanceID)
	if err != nil {
		writeErrorMapJSON(w, r, http.StatusInternalServerError, map[string]interface{}{"status": "error", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":                    "ok",
		"instance_id":               detailed.InstanceID,
		"owner_task_id":             ownerTaskID,
		"from_instance_id":          fromInstanceID,
		"via":                       detailed.Via,
		"container_start_triggered": detailed.ContainerStartTriggered,
		"container_image_id":        detailed.ContainerImageID,
	})
}
