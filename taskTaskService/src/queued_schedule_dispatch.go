package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"autorunstartvm"
	"tracelog"
)

func runQueuedScheduleDispatchOnce() {
	// 工作空间级调度：按 workspace 分组分发。工作空间未配置节奏时
	// 回退任务级（legacy）逐 top_task 分发（生产既有配置不中断）。
	rows, err := db.Query(`SELECT DISTINCT workspace_id FROM task_queued_auto_run_memberships`)
	if err != nil {
		return
	}
	defer rows.Close()
	var workspaces []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			workspaces = append(workspaces, id)
		}
	}
	for _, wsID := range workspaces {
		dispatchWorkspaceQueue(wsID)
	}
}

func dispatchWorkspaceQueue(workspaceID string) {
	dispatchWorkspaceQueueWith(workspaceID, startQueuedMembership)
}

// dispatchWorkspaceQueueWith 工作空间级分发；startFn 可在测试中替换。
func dispatchWorkspaceQueueWith(workspaceID string, startFn func(queuedMembership) error) {
	wsRhythm, wsScoped := loadEffectiveWorkspaceRhythm(workspaceID)
	if !wsScoped {
		// legacy 回退：按 top_task 逐任务级节奏分发
		tops, err := topTaskIDsForWorkspace(workspaceID)
		if err != nil {
			return
		}
		for _, topID := range tops {
			dispatchTopQueueWith(topID, startFn)
		}
		return
	}
	members, err := listMembershipsForWorkspace(workspaceID)
	if err != nil {
		return
	}
	now := time.Now()
	if !wsRhythm.Enabled {
		recordWorkspaceWindowTransition(wsRhythm, false)
		deferAllMembers(members, "自动调度未启用", false)
		return
	}
	inWin := workspaceRhythmInWindow(wsRhythm, now)
	recordWorkspaceWindowTransition(wsRhythm, inWin)
	if !inWin {
		deferAllMembers(members, deferredReasonWorkspace(wsRhythm), true)
		return
	}
	windows, _ := loadWorkspaceScheduleRhythmWindows(workspaceID)
	maxN := 0
	nowLocal := now.In(resolveScheduleLocation(wsRhythm.Timezone))
	for _, w := range windows {
		if inDailyWindow(nowLocal, w.DailyStart, w.DailyEnd) {
			maxN += w.MaxQueuedMachines
		}
	}
	if maxN <= 0 {
		return
	}
	reclaimStaleStartingMemberships(members)
	slots := countWorkspaceQueuedSlots(workspaceID)
	for _, m := range members {
		if slots >= maxN {
			break
		}
		if m.Status == "starting" && hasQueuedSlot(m.TaskID) {
			continue
		}
		if err := startFn(m); err != nil {
			log.Printf("[taskTaskService] queued start failed task_id=%s workspace_id=%s: %v", m.TaskID, workspaceID, err)
			setMembershipStatus(m.TaskID, "queued")
			continue
		}
		slots++
	}
}

// reclaimStaleStartingMemberships 把「starting 但无槽位」收回 queued，允许下一轮 startVM。
// 旧逻辑无条件 skip starting，startVM 超时/失败后 UI 永久卡在「调度启服中」且占用 0。
func reclaimStaleStartingMemberships(members []queuedMembership) {
	for i := range members {
		m := &members[i]
		if m.Status != "starting" || hasQueuedSlot(m.TaskID) {
			continue
		}
		log.Printf("[taskTaskService] event=queued_starting_reclaim task_id=%s workspace_id=%s top_task_id=%s",
			m.TaskID, m.WorkspaceID, m.TopTaskID)
		setMembershipStatus(m.TaskID, "queued")
		m.Status = "queued"
	}
}

// deferAllMembers 将成员置为 deferred 并投递 QueuedAutoRunDeferred 事件。
// skipStarting 仅跳过「starting 且已占槽」的进行中启服；无槽 starting 视为卡死，窗口外应收 deferred。
func deferAllMembers(members []queuedMembership, reason string, skipStarting bool) {
	for _, m := range members {
		if skipStarting && m.Status == "starting" && hasQueuedSlot(m.TaskID) {
			continue
		}
		if m.Status != "deferred" {
			setMembershipStatus(m.TaskID, "deferred")
			_ = publishDomainEvent(context.Background(), "QueuedAutoRunDeferred", map[string]interface{}{
				"task_id": m.TaskID, "tenant_id": m.TenantID, "workspace_id": m.WorkspaceID, "reason": reason,
			}, m.TaskID)
		}
	}
}

func dispatchTopQueue(topID string) {
	dispatchTopQueueWith(topID, startQueuedMembership)
}

// dispatchTopQueueWith 任务级（legacy）分发；startFn 可在测试中替换。
func dispatchTopQueueWith(topID string, startFn func(queuedMembership) error) {
	rhythm, err := loadScheduleRhythm(topID)
	if err != nil {
		return
	}
	members, err := listMembershipsForTop(topID)
	if err != nil {
		return
	}
	now := time.Now()
	if rhythm == nil || !rhythm.Enabled {
		deferAllMembers(members, "自动调度未启用", false)
		return
	}
	if !rhythmInWindow(rhythm, now) {
		deferAllMembers(members, deferredReason(rhythm), true)
		return
	}
	slots := countQueuedSlots(topID)
	windows, _ := loadScheduleRhythmWindows(topID)
	maxN := 0
	for _, w := range windows {
		if inDailyWindow(time.Now(), w.DailyStart, w.DailyEnd) {
			maxN += w.MaxQueuedMachines
		}
	}
	if maxN <= 0 {
		return
	}
	reclaimStaleStartingMemberships(members)
	slots = countQueuedSlots(topID)
	for _, m := range members {
		if slots >= maxN {
			break
		}
		if m.Status == "starting" && hasQueuedSlot(m.TaskID) {
			continue
		}
		if err := startFn(m); err != nil {
			log.Printf("[taskTaskService] queued start failed task_id=%s: %v", m.TaskID, err)
			setMembershipStatus(m.TaskID, "queued")
			continue
		}
		slots++
	}
}

// topTaskIDsForWorkspace 列出工作空间内出现的 top_task 集合（legacy 回退用）。
func topTaskIDsForWorkspace(workspaceID string) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT top_task_id FROM task_queued_auto_run_memberships WHERE workspace_id=?`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tops []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			tops = append(tops, id)
		}
	}
	return tops, rows.Err()
}

// listMembershipsForWorkspace 列出工作空间全部队列成员（工作空间级调度用）。
func listMembershipsForWorkspace(workspaceID string) ([]queuedMembership, error) {
	rows, err := db.Query(`SELECT task_id,tenant_id,workspace_id,top_task_id,depth,status,COALESCE(enabler_user_id,''),enqueued_at,updated_at
		FROM task_queued_auto_run_memberships WHERE workspace_id=?
		ORDER BY depth ASC, enqueued_at ASC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []queuedMembership
	for rows.Next() {
		var m queuedMembership
		if err := rows.Scan(&m.TaskID, &m.TenantID, &m.WorkspaceID, &m.TopTaskID,
			&m.Depth, &m.Status, &m.EnablerUserID, &m.EnqueuedAt, &m.UpdatedAt); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func setMembershipStatus(taskID, status string) {
	now := time.Now().UTC()
	_, _ = db.Exec(`UPDATE task_queued_auto_run_memberships SET status=?, updated_at=? WHERE task_id=?`, status, now, taskID)
}

func startQueuedMembership(m queuedMembership) error {
	t, err := loadTask(m.TaskID)
	if err != nil {
		return err
	}
	held := false
	defer func() {
		if held {
			return
		}
		releaseQueuedSlot(m.TaskID)
		setMembershipStatus(m.TaskID, "queued")
	}()
	if err := acquireQueuedSlot(m.TaskID, m.TopTaskID); err != nil {
		return err
	}
	setMembershipStatus(m.TaskID, "starting")
	tpl, gateErr := validateAutoRunPrerequisites(m.TenantID, t.InstalledImageID, linkedProjectIDsForTask(m.TaskID, m.TenantID))
	if gateErr != nil {
		return gateErr
	}
	actor := strings.TrimSpace(m.EnablerUserID)
	if actor == "" {
		actor = existingAutoRunCommentAuthor(m.TaskID)
	}
	if actor == "" {
		actor = strings.TrimSpace(t.OwnerID)
	}
	p := autoRunTriggerParams{
		TenantID:    m.TenantID,
		WorkspaceID: m.WorkspaceID,
		TaskID:      m.TaskID,
		UserID:      actor,
		ImageID:     t.InstalledImageID,
		RunTemplate: tpl,
	}
	commentID, err := resolveAutoRunCommentID(p)
	if err != nil {
		return err
	}
	if author := loadCommentCreatedByID(commentID); author != "" {
		p.UserID = author
	}
	req, err := autorunstartvm.BuildAutoRunStartVmRequest(p.TaskID, p.ImageID, p.RunTemplate)
	if err != nil {
		return err
	}
	if req.Body == nil {
		req.Body = map[string]interface{}{}
	}
	attachStartVmCommentID(req.Body, commentID, p.TaskID)
	req.Body["started_via"] = "queued_schedule"
	ctx := context.Background()
	traceID := tracelog.NormalizeTraceID(m.TaskID)
	if traceID == "" {
		traceID = tracelog.NewTraceID()
	}
	ctx = tracelog.ContextWithCorrelation(ctx, tracelog.Correlation{
		TraceID: traceID,
		SpanID:  tracelog.NewSpanID(),
	})
	tracelog.LogForwardStage(ctx, "queued_auto_run_start_vm_begin", map[string]any{
		"task_id":     m.TaskID,
		"top_task_id": m.TopTaskID,
		"comment_id":  commentID,
	})
	if err := startVMFn(ctx, p.TenantID, p.WorkspaceID, req.APIPath, p.UserID, req.Body); err != nil {
		tracelog.LogForwardStage(ctx, "queued_auto_run_start_vm_error", map[string]any{
			"task_id": m.TaskID, "error": err.Error(),
		})
		return err
	}
	held = true
	_ = publishDomainEvent(ctx, "QueuedAutoRunStarted", map[string]interface{}{
		"task_id": m.TaskID, "tenant_id": m.TenantID, "workspace_id": m.WorkspaceID, "top_task_id": m.TopTaskID,
	}, m.TaskID)
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: m.TenantID, WorkspaceID: m.WorkspaceID, EventType: scheduleHistoryEventMemberStarted,
		TaskID: m.TaskID, Message: "已调度启动任务",
	})
	tracelog.LogForwardStage(ctx, "queued_auto_run_start_vm_ok", map[string]any{"task_id": m.TaskID})
	return nil
}

func linkedProjectIDsForTask(taskID, tenantID string) []string {
	return linkedProjectIDsFromBodyOrTask(nil, taskID, tenantID)
}

func loadCommentCreatedByID(commentID string) string {
	commentID = strings.TrimSpace(commentID)
	if commentID == "" || db == nil {
		return ""
	}
	var raw string
	err := db.QueryRow(`SELECT COALESCE(created_by_id,'') FROM task_comments WHERE id=? LIMIT 1`, commentID).Scan(&raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(raw)
}

func existingAutoRunCommentAuthor(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || db == nil {
		return ""
	}
	var raw string
	err := db.QueryRow(`
SELECT COALESCE(created_by_id,'') FROM task_comments
WHERE task_id=? AND content LIKE '【自动运行】%'
ORDER BY created_at ASC LIMIT 1`, taskID).Scan(&raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(raw)
}

// handleInternalQueuedScheduleDispatchOnce 供 taskEvents queued_auto_run_scan timer
// 一次性触发（OPT-20260816-030）：一次 POST 顺序执行「排队调度分发」与「窗口自动关闭」。
// 业务进程不再自带 30s ticker。
func handleInternalQueuedScheduleDispatchOnce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	runQueuedScheduleDispatchOnce()
	runQueuedScheduleAutoCloseOnce()
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "success"})
}
