package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// workspaceScheduleRhythm — 工作空间级排队调度节奏。
// 与任务级 task_top_deliverable_schedule_rhythms 并存；工作空间未配置节奏时
// 调度分发回退任务级（legacy）逻辑，生产既有配置不中断（见 dispatchWorkspaceQueue）。
type workspaceScheduleRhythm struct {
	WorkspaceID string
	TenantID    string
	Enabled     bool
	Timezone    string
	UpdatedAt   time.Time
}

type workspaceScheduleRhythmWindow struct {
	ID                   string
	WorkspaceID          string
	TenantID             string
	DailyStart           string // HH:MM
	DailyEnd             string
	MaxQueuedMachines    int
	AutoClose            bool
	AutoCloseWarnMinutes int // 关闭前预告分钟数，0 表示使用默认值 5
	AutoCloseWarnKey     string
	AutoCloseReleaseKey  string
	SortOrder            int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func loadWorkspaceScheduleRhythm(workspaceID string) (*workspaceScheduleRhythm, error) {
	row := db.QueryRow(`SELECT workspace_id,tenant_id,enabled,timezone,updated_at
		FROM workspace_schedule_rhythms WHERE workspace_id=?`, workspaceID)
	var r workspaceScheduleRhythm
	var en int
	var ua time.Time
	err := row.Scan(&r.WorkspaceID, &r.TenantID, &en, &r.Timezone, &ua)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled = en != 0
	r.UpdatedAt = ua
	return &r, nil
}

func loadWorkspaceScheduleRhythmWindows(workspaceID string) ([]workspaceScheduleRhythmWindow, error) {
	rows, err := db.Query(`SELECT id,workspace_id,tenant_id,daily_start,daily_end,
		max_queued_machines,COALESCE(auto_close,0),COALESCE(auto_close_warn_minutes,5),
		COALESCE(auto_close_warn_key,''),COALESCE(auto_close_release_key,''),sort_order,created_at,updated_at
		FROM workspace_schedule_rhythm_windows WHERE workspace_id=? ORDER BY sort_order ASC, created_at ASC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []workspaceScheduleRhythmWindow
	for rows.Next() {
		var w workspaceScheduleRhythmWindow
		var ac int
		if err := rows.Scan(&w.ID, &w.WorkspaceID, &w.TenantID, &w.DailyStart, &w.DailyEnd,
			&w.MaxQueuedMachines, &ac, &w.AutoCloseWarnMinutes,
			&w.AutoCloseWarnKey, &w.AutoCloseReleaseKey, &w.SortOrder, &w.CreatedAt, &w.UpdatedAt); err != nil {
			continue
		}
		w.AutoClose = ac != 0
		out = append(out, w)
	}
	return out, rows.Err()
}

func upsertWorkspaceScheduleRhythm(r workspaceScheduleRhythm) error {
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO workspace_schedule_rhythms(
		workspace_id,tenant_id,enabled,timezone,updated_at)
		VALUES(?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			enabled=VALUES(enabled),
			timezone=VALUES(timezone),
			updated_at=VALUES(updated_at)`,
		r.WorkspaceID, r.TenantID, boolToInt(r.Enabled), r.Timezone, now)
	return err
}

func upsertWorkspaceScheduleRhythmWindows(workspaceID, tenantID string, windows []workspaceScheduleRhythmWindow) error {
	// Delete existing windows for this workspace and re-insert
	if _, err := db.Exec(`DELETE FROM workspace_schedule_rhythm_windows WHERE workspace_id=?`, workspaceID); err != nil {
		return err
	}
	now := time.Now().UTC()
	for i, w := range windows {
		sortOrder := w.SortOrder
		if sortOrder == 0 && i > 0 {
			sortOrder = i
		}
		_, err := db.Exec(`INSERT INTO workspace_schedule_rhythm_windows(
			id,workspace_id,tenant_id,daily_start,daily_end,max_queued_machines,auto_close,
			auto_close_warn_minutes,auto_close_warn_key,auto_close_release_key,sort_order,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			w.ID, workspaceID, tenantID, w.DailyStart, w.DailyEnd, w.MaxQueuedMachines, boolToInt(w.AutoClose),
			w.AutoCloseWarnMinutes, w.AutoCloseWarnKey, w.AutoCloseReleaseKey, sortOrder, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

// resolveScheduleLocation 解析节奏时区；空值/非法回退 Asia/Shanghai。
func resolveScheduleLocation(timezone string) *time.Location {
	tz := strings.TrimSpace(timezone)
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// workspaceRhythmInWindow 报告当前时间是否处于工作空间节奏的任一窗口内。
func workspaceRhythmInWindow(r *workspaceScheduleRhythm, now time.Time) bool {
	if r == nil || !r.Enabled {
		return false
	}
	windows, _ := loadWorkspaceScheduleRhythmWindows(r.WorkspaceID)
	nowLocal := now.In(resolveScheduleLocation(r.Timezone))
	for _, w := range windows {
		if inDailyWindow(nowLocal, w.DailyStart, w.DailyEnd) {
			return true
		}
	}
	return false
}

// loadEffectiveWorkspaceRhythm 返回工作空间的有效节奏。
// isWorkspace=false 表示工作空间尚未配置节奏 → 调度回退任务级（legacy）。
func loadEffectiveWorkspaceRhythm(workspaceID string) (*workspaceScheduleRhythm, bool) {
	r, err := loadWorkspaceScheduleRhythm(workspaceID)
	if err != nil || r == nil {
		return nil, false
	}
	return r, true
}

func deferredReasonWorkspace(r *workspaceScheduleRhythm) string {
	if r == nil || !r.Enabled {
		return "自动调度未启用"
	}
	windows, _ := loadWorkspaceScheduleRhythmWindows(r.WorkspaceID)
	if len(windows) == 0 {
		return "未配置调度时段"
	}
	parts := make([]string, 0, len(windows))
	for _, w := range windows {
		parts = append(parts, fmt.Sprintf("%s–%s", w.DailyStart, w.DailyEnd))
	}
	return fmt.Sprintf("等待时段 %s (%s)", strings.Join(parts, ", "), r.Timezone)
}

// workspaceRhythmToJSON 序列化工作空间节奏（含窗口与窗口态）。
func workspaceRhythmToJSON(r *workspaceScheduleRhythm) map[string]interface{} {
	if r == nil {
		return nil
	}
	windows, _ := loadWorkspaceScheduleRhythmWindows(r.WorkspaceID)
	winJSON := make([]map[string]interface{}, 0, len(windows))
	maxTotal := 0
	autoCloseAny := false
	for _, w := range windows {
		maxTotal += w.MaxQueuedMachines
		if w.AutoClose {
			autoCloseAny = true
		}
		winJSON = append(winJSON, map[string]interface{}{
			"id":                      w.ID,
			"daily_start":             w.DailyStart,
			"daily_end":               w.DailyEnd,
			"max_queued_machines":     w.MaxQueuedMachines,
			"auto_close":              w.AutoClose,
			"auto_close_warn_minutes": w.AutoCloseWarnMinutes,
		})
	}
	return map[string]interface{}{
		"enabled":                 r.Enabled,
		"timezone":                r.Timezone,
		"windows":                 winJSON,
		"max_queued_machines":     maxTotal,
		"auto_close":              autoCloseAny,
		"auto_close_warn_minutes": 5,
		"in_window":               workspaceRhythmInWindow(r, time.Now()),
	}
}

// workspaceWindowMessage 供 GET 快照展示当前调度窗口状态。
func workspaceWindowMessage(r *workspaceScheduleRhythm, inWindow bool) string {
	if r == nil || !r.Enabled {
		return "自动调度未启用"
	}
	if inWindow {
		return "当前处于允许运行时段"
	}
	return deferredReasonWorkspace(r)
}

// applyWorkspaceScheduleRhythmFromBody 解析并保存工作空间节奏（PUT queue-schedule）。
// body 即节奏对象：{ enabled, timezone, windows:[{id,daily_start,daily_end,max_queued_machines,auto_close,auto_close_warn_minutes}] }
func applyWorkspaceScheduleRhythmFromBody(tenantID, workspaceID, actorUserID string, body map[string]interface{}) error {
	enabled := boolField(body, "enabled")
	tz := strField(body, "timezone")
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	rawWindows, hasWindows := body["windows"]
	if !hasWindows {
		return fmt.Errorf("schedule_rhythm.windows is required")
	}
	winList, ok := rawWindows.([]interface{})
	if !ok {
		return fmt.Errorf("schedule_rhythm.windows must be an array")
	}
	var windows []workspaceScheduleRhythmWindow
	validate := make([]scheduleRhythmWindow, 0, len(winList))
	for i, rawW := range winList {
		wObj, ok := rawW.(map[string]interface{})
		if !ok {
			return fmt.Errorf("schedule_rhythm.windows[%d] must be an object", i)
		}
		start := strField(wObj, "daily_start")
		end := strField(wObj, "daily_end")
		if start != "" {
			if _, _, ok := parseHHMM(start); !ok {
				return fmt.Errorf("windows[%d].daily_start 格式须为 HH:MM", i)
			}
		}
		if end != "" {
			if _, _, ok := parseHHMM(end); !ok {
				return fmt.Errorf("windows[%d].daily_end 格式须为 HH:MM", i)
			}
		}
		maxN := intFromField(wObj, "max_queued_machines")
		if maxN < 0 {
			maxN = 0
		}
		autoClose := boolField(wObj, "auto_close")
		autoCloseWarnMinutes := intFromField(wObj, "auto_close_warn_minutes")
		if autoCloseWarnMinutes <= 0 {
			autoCloseWarnMinutes = 5
		}
		// Preserve existing warn/release keys if matching window ID
		warnKey, releaseKey := "", ""
		if wid := strField(wObj, "id"); wid != "" {
			prevWindows, _ := loadWorkspaceScheduleRhythmWindows(workspaceID)
			for _, pw := range prevWindows {
				if pw.ID == wid {
					warnKey, releaseKey = pw.AutoCloseWarnKey, pw.AutoCloseReleaseKey
					break
				}
			}
		}
		windows = append(windows, workspaceScheduleRhythmWindow{
			ID:                   strField(wObj, "id"),
			WorkspaceID:          workspaceID,
			TenantID:             tenantID,
			DailyStart:           start,
			DailyEnd:             end,
			MaxQueuedMachines:    maxN,
			AutoClose:            autoClose,
			AutoCloseWarnMinutes: autoCloseWarnMinutes,
			AutoCloseWarnKey:     warnKey,
			AutoCloseReleaseKey:  releaseKey,
			SortOrder:            i,
		})
		validate = append(validate, scheduleRhythmWindow{DailyStart: start, DailyEnd: end})
	}

	// Validate no overlap between windows
	if len(windows) > 0 {
		if err := validateWindowsNoOverlap(validate); err != nil {
			return err
		}
	}

	// Assign IDs to windows that don't have one
	for i := range windows {
		if windows[i].ID == "" {
			windows[i].ID = generateWindowID(workspaceID + "_" + strconv.Itoa(i))
		}
	}

	if err := upsertWorkspaceScheduleRhythm(workspaceScheduleRhythm{
		WorkspaceID: workspaceID,
		TenantID:    tenantID,
		Enabled:     enabled,
		Timezone:    tz,
	}); err != nil {
		return err
	}
	if err := upsertWorkspaceScheduleRhythmWindows(workspaceID, tenantID, windows); err != nil {
		return err
	}
	_ = publishDomainEvent(context.Background(), "WorkspaceScheduleSaved", map[string]interface{}{
		"workspace_id": workspaceID,
		"tenant_id":    tenantID,
		"enabled":      enabled,
		"window_count": len(windows),
	}, workspaceID)
	saveMsg := "已保存调度设置"
	if enabled {
		saveMsg += "（已启用）"
	} else {
		saveMsg += "（未启用）"
	}
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: tenantID, WorkspaceID: workspaceID, EventType: scheduleHistoryEventRhythmSaved,
		Message: saveMsg, ActorUserID: actorUserID,
	})
	return nil
}

// listWorkspaceQueueItems 返回工作空间队列成员（含任务标题），按 depth/enqueued 排序。
func listWorkspaceQueueItems(workspaceID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT m.task_id,m.tenant_id,m.workspace_id,m.top_task_id,m.depth,m.status,m.enqueued_at,COALESCE(t.title,'')
		FROM task_queued_auto_run_memberships m
		LEFT JOIN task_tasks t ON t.id = m.task_id
		WHERE m.workspace_id=? ORDER BY m.depth ASC, m.enqueued_at ASC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var m queuedMembership
		var title string
		if err := rows.Scan(&m.TaskID, &m.TenantID, &m.WorkspaceID, &m.TopTaskID,
			&m.Depth, &m.Status, &m.EnqueuedAt, &title); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"task_id":      m.TaskID,
			"title":        title,
			"top_task_id":  m.TopTaskID,
			"depth":        m.Depth,
			"status":       m.Status,
			"enqueued_at":  m.EnqueuedAt.UTC().Format(time.RFC3339Nano),
			"workspace_id": m.WorkspaceID,
		})
	}
	return out, rows.Err()
}

// countWorkspaceQueuedSlots 统计工作空间当前占用的机器槽位（按成员归属聚合）。
func countWorkspaceQueuedSlots(workspaceID string) int {
	var n int
	_ = db.QueryRow(`SELECT COUNT(1) FROM task_queued_machine_slots s
		INNER JOIN task_queued_auto_run_memberships m ON m.task_id = s.task_id
		WHERE m.workspace_id=?`, workspaceID).Scan(&n)
	return n
}

// clearWorkspaceQueuedSlots 清空工作空间全部机器槽位（auto_close 释放用）。
func clearWorkspaceQueuedSlots(workspaceID string) {
	_, _ = db.Exec(`DELETE s FROM task_queued_machine_slots s
		INNER JOIN task_queued_auto_run_memberships m ON m.task_id = s.task_id
		WHERE m.workspace_id=?`, workspaceID)
}
