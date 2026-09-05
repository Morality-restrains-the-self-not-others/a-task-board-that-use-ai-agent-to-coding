package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"taskTaskService/domain"
	"tracelog"
)

func handleUpdateTask(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	if !canMutateTaskFromRequest(r, tenantID, userID, t) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	if !requirePostActive(w, t) {
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, errMsgInvalidJSON)
		return
	}
	progressColumnName, err := validateTaskFieldsFromRequest(r, tenantID, t.WorkspaceID, body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	title := strField(body, "title")
	if title == "" {
		title = t.Title
	}
	desc := strField(body, "description")
	if _, ok := body["description"]; !ok {
		desc = t.Description
	}
	priority := strField(body, "priority")
	if priority == "" {
		priority = t.Priority
	}
	prevCompleted := t.Completed
	prevProgressColumnID := t.ProgressColumnID
	prevAutoRun := t.AutoRun
	completed := t.Completed
	if _, ok := body["completed"]; ok {
		completed = boolField(body, "completed")
	}
	autoRun := t.AutoRun
	if _, ok := body["auto_run"]; ok {
		autoRun = boolField(body, "auto_run")
	}
	autoCommit := t.AutoCommitAfterAgentComplete
	if _, ok := body["auto_commit_after_agent_complete"]; ok {
		autoCommit = boolField(body, "auto_commit_after_agent_complete")
	}
	forceAutoRun := boolField(body, "force_auto_run")
	if forceAutoRun && !autoRun {
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
			"error":  "强制重新启动需要启用自动运行",
			"detail": "强制重新启动需要启用自动运行",
			"code":   "AUTO_RUN_FORCE_REQUIRES_ENABLED",
		})
		return
	}
	newProgressColumnID := coalesceStr(body, "progress_column_id", t.ProgressColumnID)
	enteringTerminal := updateEntersTerminal(prevProgressColumnID, newProgressColumnID, progressColumnName, prevCompleted, completed)
	imageID := coalesceStr(body, "container_image_id", t.InstalledImageID)
	var autoRunTpl map[string]interface{}
	if shouldValidateAutoRunOnTaskUpdate(autoRun, enteringTerminal) {
		tpl, gateErr := validateAutoRunPrerequisites(tenantID, imageID, linkedProjectIDsFromBodyOrTask(body, taskID, tenantID))
		if gateErr != nil {
			if writeAutoRunGateError(w, gateErr) {
				return
			}
			writeError(w, r, http.StatusBadRequest, gateErr.Error())
			return
		}
		autoRunTpl = tpl
	} else if autoRun && enteringTerminal {
		logSkippedAutoRunPrereqOnTerminal(r.Context(), taskID, progressColumnName)
	}
	shouldTriggerAutoRun := autoRun && (!prevAutoRun || forceAutoRun) && !enteringTerminal
	var autoRunIdents []RepoIdentitySelection
	if shouldTriggerAutoRun {
		idents, identErr := resolveAutoRunRepoIdentities(r.Context(), true, userID, tenantID, taskID, body)
		if identErr != nil {
			writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
				"error":  identErr.Error(),
				"detail": identErr.Error(),
				"code":   "repo_identities_required",
			})
			return
		}
		autoRunIdents = idents
	}
	ownerID := t.OwnerID
	if v := ownerFromBody(body); v != "" {
		ownerID = v
	}
	operatorID := t.OperatorID
	if _, ok := body["operator"]; ok {
		operatorID = operatorFromBody(body)
	} else if _, ok := body["operator_id"]; ok {
		operatorID = operatorFromBody(body)
	}
	fps := t.FeatureParamsSource
	if v := strField(body, "feature_params_source"); v != "" {
		fps = v
	}
	pfpc := t.PersonalFeatureParamsConfigID
	if _, ok := body["personal_feature_params_config_id"]; ok {
		pfpc = strField(body, "personal_feature_params_config_id")
	}
	if hasFeatureParamsKeys(body) {
		if err := validateFeatureParamsSourceRequired(fps, pfpc); err != nil {
			writeFeatureParamsRequiredError(w, err.Error())
			return
		}
	}
	orderNum := t.OrderNum
	if v := strField(body, "order"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		orderNum = n
	}
	if enteringTerminal {
		if enforceDescendantTerminalGate(w, tenantID, t.WorkspaceID, taskID, progressColumnName, prevCompleted, completed) {
			return
		}
	}
	now := time.Now().UTC()
	oldVersioned := domain.VersionedTaskContent{Title: t.Title, Description: t.Description}
	newVersioned := domain.VersionedTaskContent{Title: title, Description: desc}
	taskKind := t.TaskKind
	if _, ok := body["task_kind"]; ok {
		taskKind = strField(body, "task_kind")
	}
	codeLang := t.CodeLang
	if _, ok := body["code_lang"]; ok {
		codeLang = strField(body, "code_lang")
	}
	// D4 契约：镜像技能绑定校验 + 快照更新（与创建同规则）。未触碰镜像/技能字段时
	// 按既有值重校验，幂等；镜像解绑（body 显式传空 container_image_id）→ 技能与
	// 快照一并清空（此时不得回退既有 skill_id，否则会误触 skill_id_requires_image）。
	// 宽容路径：body 未显式改镜像（沿用既有绑定）且镜像已从目录卸载 → 保留既有
	// 快照不阻断更新（既有契约：任务不因镜像目录失效而无法更新/完成，hydrate 侧
	// 以 container_image_removed 展示）。仅「显式请求绑定」才 fail-closed。
	skillID := coalesceStr(body, "container_image_skill_id", t.ImageSkillID)
	explicitImage := false
	if _, ok := body["container_image_id"]; ok {
		explicitImage = true
		if imageID == "" {
			skillID = ""
		}
	}
	skillSnapshot, serr := resolveTaskImageSkill(tenantID, imageID, skillID, desc)
	imageSkillID, snapshotJSON := "", ""
	if serr != nil {
		if !explicitImage && serr.code == "image_not_found" {
			imageSkillID, snapshotJSON = t.ImageSkillID, t.ContainerImageSnapshot
		} else {
			writeErrorMap(w, r, serr.status, map[string]interface{}{
				"error": serr.msg, "detail": serr.msg, "code": serr.code,
			})
			return
		}
	} else if skillSnapshot != nil {
		imageSkillID = skillSnapshot.SkillID
		snapshotJSON, _ = skillSnapshot.encode()
	}
	// JSON 列不能存空字符串（非法 JSON），未绑定技能时写入 NULL。
	var snapshotVal interface{}
	if snapshotJSON != "" {
		snapshotVal = snapshotJSON
	}
	tx, txErr := db.Begin()
	if txErr != nil {
		writeError(w, r, http.StatusInternalServerError, txErr.Error())
		return
	}
	_, err = tx.Exec(`UPDATE task_tasks SET title=?,description=?,completed=?,priority=?,order_num=?,owner_id=?,operator_id=?,deliverable_obj_id=?,progress_column_id=?,parent_task_id=?,fork_from_id=?,installed_image_id=?,image_skill_id=?,container_image_snapshot=?,auto_run=?,auto_commit_after_agent_complete=?,feature_params_source=?,personal_feature_params_config_id=?,task_kind=?,code_lang=?,updated_at=? WHERE id=?`,
		title, desc, boolToInt(completed), priority, orderNum, ownerID, operatorID,
		coalesceStr(body, "deliverable_obj_id", t.DeliverableObjID),
		newProgressColumnID,
		coalesceStr(body, "parent_task", t.ParentTaskID),
		coalesceStr(body, "fork_from", t.ForkFromID),
		imageID,
		imageSkillID, snapshotVal,
		boolToInt(autoRun), boolToInt(autoCommit), fps, pfpc, taskKind, codeLang, now, taskID)
	if err != nil {
		_ = tx.Rollback()
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	revID, revNum, revChanged, revRecorded, revErr := persistTaskRevisionIfChanged(
		tx, tenantID, t.WorkspaceID, taskID, userID, oldVersioned, newVersioned, now)
	if revErr != nil {
		_ = tx.Rollback()
		log.Printf("[taskTaskService] event=task_revision_persist_failed task_id=%s err=%v", taskID, revErr)
		writeError(w, r, http.StatusInternalServerError, revErr.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	tAfterParent, _ := loadTask(taskID)
	if tAfterParent == nil {
		tAfterParent = t
	}
	if err := applyScheduleRhythmFromBody(tAfterParent, body); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := applyQueuedAutoRunFromBody(tAfterParent, body, userID); err != nil {
		writeError(w, r, httpStatusOf(err, http.StatusBadRequest), err.Error())
		return
	}
	if assignees := memberIDsFromBody(body, "assignees"); assignees != nil {
		setAssignees(taskID, assignees)
	}
	if projs, ok := body["projects"].([]interface{}); ok {
		db.Exec(`DELETE FROM task_projects WHERE task_id=?`, taskID)
		if err := saveTaskProjects(r.Context(), taskID, tenantID, projs); err != nil {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}
	saveBranchStrategy(taskID, body)
	t, _ = loadTask(taskID)
	payload := taskJSONWithCreatedBy(t, tenantID)
	autoRunSkipReason := ""
	if shouldTriggerAutoRun {
		autoRunSkipReason = probeGitAccessForAutoRun(r.Context(), userID, tenantID, linkedProjectIDsFromBodyOrTask(body, taskID, tenantID))
		if autoRunSkipReason != "" {
			applyAutoRunStartSkip(payload, autoRunSkipReason)
			persistAutoRunStartSkipReason(taskID, autoRunSkipReason)
			log.Printf("[taskTaskService] auto_run start skipped task_id=%s reason=%s", taskID, autoRunSkipReason)
		} else {
			clearAutoRunStartSkipReason(taskID)
		}
	}
	writeJSON(w, http.StatusOK, payload)
	if revRecorded {
		if pubErr := publishTaskRevisionRecordedFn(r.Context(), tenantID, t.WorkspaceID, taskID, revID, revNum, userID, domain.JoinChangedFields(revChanged)); pubErr != nil {
			log.Printf("[taskTaskService] publish TASK_REVISION_RECORDED failed task_id=%s revision_id=%s: %v", taskID, revID, pubErr)
		}
	}
	if prevProgressColumnID != newProgressColumnID || prevCompleted != completed {
		if pubErr := publishTaskStatusChangedFn(r.Context(), tenantID, t.WorkspaceID, taskID, prevProgressColumnID, newProgressColumnID, prevCompleted, completed, progressColumnName); pubErr != nil {
			log.Printf("[taskTaskService] TASK_STATUS_CHANGED publish failed task_id=%s: %v", taskID, pubErr)
		}
	}
	// Soft-skip still schedules comment-only auto_run (StartSkipReason set → no start-vm).
	if shouldTriggerAutoRun {
		if deferImmediateAutoRunStart(body, forceAutoRun) {
			log.Printf("[taskTaskService] event=task_auto_run_deferred_to_queue task_id=%s tenant_id=%s workspace_id=%s",
				taskID, tenantID, t.WorkspaceID)
			tracelog.LogForwardStage(r.Context(), "task_auto_run_deferred_to_queue", map[string]any{
				"task_id": taskID, "tenant_id": tenantID, "workspace_id": t.WorkspaceID,
			})
		} else {
			scheduleTaskAutoRunFn(autoRunTriggerParams{
				TenantID:        tenantID,
				WorkspaceID:     t.WorkspaceID,
				TaskID:          taskID,
				UserID:          userID,
				ImageID:         imageID,
				RunTemplate:     autoRunTpl,
				ClientPublicIP:  resolveAutoRunClientPublicIP(r, body),
				StartSkipReason: autoRunSkipReason,
				RepoIdentities:  autoRunIdents,
				GrantTicket:     grantTicketFromBody(body),
			})
		}
	}
}

func coalesceStr(body map[string]interface{}, key, fallback string) string {
	if _, ok := body[key]; ok {
		return strField(body, key)
	}
	return fallback
}

// handleRenewTask — 规范路径 POST /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/{taskId}/renew/
// （suffix 分发，兼容存量 /api/tasks/{taskId}/renew/ 与 /api/tenant/.../todos/.../renew/ 位置参数形式）：
// 消耗 1 个创建帖次数续存 12 个月，返回新的 post_expires_at。
func handleRenewTask(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, errMsgMethodNotAllowed)
		return
	}
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	// 传给 taskBill 的 current_expires_at 须为 UTC 墙钟字符串（diskExpiresAtFromMonths 按 UTC 解析）
	currentExpiresAt := ""
	if t.PostExpiresAt.Valid {
		currentExpiresAt = t.PostExpiresAt.Time.UTC().Format("2006-01-02 15:04:05.000000")
	}
	// consumeTaskPostRenewal 在未配置计费服务时本地兜底（now 或当前到期日的较大者 + 12 个月）
	projectID := ""
	if projects := loadProjects(taskID, tenantID); len(projects) > 0 {
		projectID = strField(projects[0], "project_id")
	}
	newExpiresAt, err := consumeTaskPostRenewal(tenantID, taskID, t.WorkspaceID, userID, projectID, currentExpiresAt)
	if err != nil {
		if ibe, ok := err.(*insufficientBalanceError); ok {
			writeErrorMap(w, r, http.StatusPaymentRequired, map[string]interface{}{
				"detail":          ibe.Message,
				"code":            "INSUFFICIENT_TASK_POST_QUOTA",
				"balance_points":  ibe.BalancePoints,
				"required_points": ibe.RequiredPoints,
			})
			return
		}
		writeError(w, r, http.StatusBadGateway, err.Error())
		return
	}
	if newExpiresAt == "" {
		newExpiresAt = time.Now().UTC().AddDate(0, 12, 0).Format("2006-01-02 15:04:05.000000")
	}
	if _, err := db.Exec(`UPDATE task_tasks SET post_expires_at=?, updated_at=? WHERE id=?`, expiresAtDBValue(newExpiresAt), time.Now().UTC(), taskID); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	t, _ = loadTask(taskID)
	writeJSON(w, http.StatusOK, taskJSONWithCreatedBy(t, tenantID))
	// 续存成功 → 发布 TASK_POST_RENEWED（taskEvents 通知用户）
	newExpiresAtRFC := ""
	if t != nil && t.PostExpiresAt.Valid {
		newExpiresAtRFC = t.PostExpiresAt.Time.UTC().Format(time.RFC3339Nano)
	}
	if pubErr := publishTaskPostRenewedFn(r.Context(), tenantID, t.WorkspaceID, taskID, userID, newExpiresAtRFC); pubErr != nil {
		log.Printf("[taskTaskService] publish TASK_POST_RENEWED failed task_id=%s: %v", taskID, pubErr)
	}
}

func handleSwitchTask(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	if !requirePostActive(w, t) {
		return
	}
	var body map[string]interface{}
	json.NewDecoder(r.Body).Decode(&body)
	prevCompleted := t.Completed
	prevProgressColumnID := t.ProgressColumnID
	completed := !t.Completed
	if _, ok := body["completed"]; ok {
		completed = boolField(body, "completed")
	}
	if !prevCompleted && completed {
		if enforceDescendantTerminalGate(w, tenantID, t.WorkspaceID, taskID, "", prevCompleted, completed) {
			return
		}
	}
	db.Exec(`UPDATE task_tasks SET completed=?, updated_at=? WHERE id=?`, boolToInt(completed), time.Now().UTC(), taskID)
	t, _ = loadTask(taskID)
	writeJSON(w, http.StatusOK, taskJSONWithCreatedBy(t, tenantID))
	if prevCompleted != completed {
		// progress_column unchanged on /switch; column name empty — consumer uses completed false→true for terminal.
		if pubErr := publishTaskStatusChangedFn(r.Context(), tenantID, t.WorkspaceID, taskID, prevProgressColumnID, t.ProgressColumnID, prevCompleted, completed, ""); pubErr != nil {
			log.Printf("[taskTaskService] TASK_STATUS_CHANGED publish failed on switch task_id=%s: %v", taskID, pubErr)
		}
	}
}

func handleAssociateTask(w http.ResponseWriter, r *http.Request, tenantID, userID, taskID string) {
	t, err := loadTask(taskID)
	if err == sql.ErrNoRows || t.TenantID != tenantID {
		writeError(w, r, http.StatusNotFound, errMsgNotFound)
		return
	}
	if !hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID) {
		writeError(w, r, http.StatusForbidden, errMsgForbidden)
		return
	}
	if !requirePostActive(w, t) {
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, errMsgInvalidJSON)
		return
	}
	projs, ok := body["projects"].([]interface{})
	if !ok || len(projs) == 0 {
		writeError(w, r, http.StatusBadRequest, errMsgProjectsRequired)
		return
	}
	db.Exec(`DELETE FROM task_projects WHERE task_id=?`, taskID)
	if err := saveTaskProjects(r.Context(), taskID, tenantID, projs); err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	t, _ = loadTask(taskID)
	writeJSON(w, http.StatusOK, taskJSONWithCreatedBy(t, tenantID))
}
