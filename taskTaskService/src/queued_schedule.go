package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type scheduleRhythm struct {
	TaskID      string
	TenantID    string
	WorkspaceID string
	Enabled     bool
	Timezone    string
	UpdatedAt   time.Time
}

type scheduleRhythmWindow struct {
	ID                   string
	TaskID               string
	TenantID             string
	WorkspaceID          string
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

type queuedMembership struct {
	TaskID        string
	TenantID      string
	WorkspaceID   string
	TopTaskID     string
	Depth         int
	Status        string
	EnablerUserID string
	EnqueuedAt    time.Time
	UpdatedAt     time.Time
}

func parseHHMM(s string) (hour, min int, ok bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

// inDailyWindow reports whether nowLocal is inside [start,end) daily window (supports overnight).
func inDailyWindow(nowLocal time.Time, dailyStart, dailyEnd string) bool {
	sh, sm, ok1 := parseHHMM(dailyStart)
	eh, em, ok2 := parseHHMM(dailyEnd)
	if !ok1 || !ok2 {
		return false
	}
	mins := nowLocal.Hour()*60 + nowLocal.Minute()
	start := sh*60 + sm
	end := eh*60 + em
	if start == end {
		return true // 24h window
	}
	if start < end {
		return mins >= start && mins < end
	}
	// overnight e.g. 22:00–06:00
	return mins >= start || mins < end
}

func loadScheduleRhythm(taskID string) (*scheduleRhythm, error) {
	row := db.QueryRow(`SELECT task_id,tenant_id,workspace_id,enabled,timezone,updated_at
		FROM task_top_deliverable_schedule_rhythms WHERE task_id=?`, taskID)
	var r scheduleRhythm
	var en int
	var ua time.Time
	err := row.Scan(&r.TaskID, &r.TenantID, &r.WorkspaceID, &en, &r.Timezone, &ua)
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

func loadScheduleRhythmWindows(taskID string) ([]scheduleRhythmWindow, error) {
	rows, err := db.Query(`SELECT id,task_id,tenant_id,workspace_id,daily_start,daily_end,
		max_queued_machines,COALESCE(auto_close,0),COALESCE(auto_close_warn_minutes,5),
		COALESCE(auto_close_warn_key,''),COALESCE(auto_close_release_key,''),sort_order,created_at,updated_at
		FROM task_top_deliverable_schedule_rhythm_windows WHERE task_id=? ORDER BY sort_order ASC, created_at ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []scheduleRhythmWindow
	for rows.Next() {
		var w scheduleRhythmWindow
		var ac int
		if err := rows.Scan(&w.ID, &w.TaskID, &w.TenantID, &w.WorkspaceID, &w.DailyStart, &w.DailyEnd,
			&w.MaxQueuedMachines, &ac, &w.AutoCloseWarnMinutes,
			&w.AutoCloseWarnKey, &w.AutoCloseReleaseKey, &w.SortOrder, &w.CreatedAt, &w.UpdatedAt); err != nil {
			continue
		}
		w.AutoClose = ac != 0
		out = append(out, w)
	}
	return out, rows.Err()
}

// validateWindowsNoOverlap checks that no two windows' time ranges overlap.
// Returns nil if all windows are disjoint, or an error describing the overlap.
func validateWindowsNoOverlap(windows []scheduleRhythmWindow) error {
	type window struct {
		start, end int // minutes since midnight
		idx        int // 1-indexed for user-facing messages
	}
	ws := make([]window, 0, len(windows))
	for i, w := range windows {
		sh, sm, ok1 := parseHHMM(w.DailyStart)
		eh, em, ok2 := parseHHMM(w.DailyEnd)
		if !ok1 || !ok2 {
			continue // skip invalid, validated elsewhere
		}
		start := sh*60 + sm
		end := eh*60 + em
		if start == end {
			continue // 24h window spans everything
		}
		ws = append(ws, window{start: start, end: end, idx: i + 1})
	}
	for i := 0; i < len(ws); i++ {
		for j := i + 1; j < len(ws); j++ {
			a, b := ws[i], ws[j]
			if intervalsOverlap(a.start, a.end, b.start, b.end) {
				return fmt.Errorf("时间段 %d（%s–%s）与时间段 %d（%s–%s）存在交叠",
					a.idx, windows[a.idx-1].DailyStart, windows[a.idx-1].DailyEnd,
					b.idx, windows[b.idx-1].DailyStart, windows[b.idx-1].DailyEnd)
			}
		}
	}
	return nil
}

// intervalsOverlap checks if [aStart, aEnd) and [bStart, bEnd) overlap.
// Both are minutes-since-midnight, with aStart < aEnd (non-overnight) XOR aStart > aEnd (overnight).
func intervalsOverlap(aStart, aEnd, bStart, bEnd int) bool {
	aOvernight := aStart > aEnd
	bOvernight := bStart > bEnd
	if !aOvernight && !bOvernight {
		return aStart < bEnd && bStart < aEnd
	}
	if aOvernight && bOvernight {
		// Both overnight — they overlap unless one is entirely before the other (impossible for overnight)
		return true
	}
	// One is overnight, one is not
	if aOvernight {
		// a: e.g. 22:00–06:00, b: e.g. 08:00–12:00
		// b is inside a's overnight gap? i.e. bStart < aEnd OR bEnd > aStart
		return bStart < aEnd || bEnd > aStart
	}
	// b is overnight
	return aStart < bEnd || aEnd > bStart
}

func upsertScheduleRhythm(r scheduleRhythm) error {
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO task_top_deliverable_schedule_rhythms(
		task_id,tenant_id,workspace_id,enabled,timezone,updated_at)
		VALUES(?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
			enabled=VALUES(enabled),
			timezone=VALUES(timezone),
			updated_at=VALUES(updated_at)`,
		r.TaskID, r.TenantID, r.WorkspaceID, boolToInt(r.Enabled), r.Timezone, now)
	return err
}

func upsertScheduleRhythmWindows(taskID, tenantID, workspaceID string, windows []scheduleRhythmWindow) error {
	// Delete existing windows for this task and re-insert
	if _, err := db.Exec(`DELETE FROM task_top_deliverable_schedule_rhythm_windows WHERE task_id=?`, taskID); err != nil {
		return err
	}
	now := time.Now().UTC()
	for i, w := range windows {
		sortOrder := w.SortOrder
		if sortOrder == 0 && i > 0 {
			sortOrder = i
		}
		_, err := db.Exec(`INSERT INTO task_top_deliverable_schedule_rhythm_windows(
			id,task_id,tenant_id,workspace_id,daily_start,daily_end,max_queued_machines,auto_close,
			auto_close_warn_minutes,auto_close_warn_key,auto_close_release_key,sort_order,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			w.ID, taskID, tenantID, workspaceID, w.DailyStart, w.DailyEnd, w.MaxQueuedMachines, boolToInt(w.AutoClose),
			w.AutoCloseWarnMinutes, w.AutoCloseWarnKey, w.AutoCloseReleaseKey, sortOrder, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}

func rhythmToJSON(r *scheduleRhythm) map[string]interface{} {
	if r == nil {
		return nil
	}
	windows, _ := loadScheduleRhythmWindows(r.TaskID)
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
		"max_queued_machines":     maxTotal, // backward compat: sum of all windows
		"auto_close":              autoCloseAny,
		"auto_close_warn_minutes": 5,
		"in_window":               rhythmInWindow(r, time.Now()),
	}
}

func rhythmInWindow(r *scheduleRhythm, now time.Time) bool {
	if r == nil || !r.Enabled {
		return false
	}
	tz := strings.TrimSpace(r.Timezone)
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	windows, _ := loadScheduleRhythmWindows(r.TaskID)
	nowLocal := now.In(loc)
	for _, w := range windows {
		if inDailyWindow(nowLocal, w.DailyStart, w.DailyEnd) {
			return true
		}
	}
	return false
}

func deferredReason(r *scheduleRhythm) string {
	if r == nil || !r.Enabled {
		return "自动调度未启用"
	}
	windows, _ := loadScheduleRhythmWindows(r.TaskID)
	if len(windows) == 0 {
		return "未配置调度时段"
	}
	parts := make([]string, 0, len(windows))
	for _, w := range windows {
		parts = append(parts, fmt.Sprintf("%s–%s", w.DailyStart, w.DailyEnd))
	}
	return fmt.Sprintf("等待时段 %s (%s)", strings.Join(parts, ", "), r.Timezone)
}

func applyScheduleRhythmFromBody(t *taskRecord, body map[string]interface{}) error {
	raw, ok := body["schedule_rhythm"]
	if !ok {
		return nil
	}
	if strings.TrimSpace(t.ParentTaskID) != "" {
		return fmt.Errorf("仅顶层交付物任务可配置调度节奏")
	}
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("schedule_rhythm must be object")
	}
	tz := strField(obj, "timezone")
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	enabled := boolField(obj, "enabled")

	// Build windows from required "windows" array
	var windows []scheduleRhythmWindow
	rawWindows, hasWindows := obj["windows"]
	if !hasWindows {
		return fmt.Errorf("schedule_rhythm.windows is required")
	}
	winList, ok := rawWindows.([]interface{})
	if !ok {
		return fmt.Errorf("schedule_rhythm.windows must be an array")
	}
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
			prevWindows, _ := loadScheduleRhythmWindows(t.ID)
			for _, pw := range prevWindows {
				if pw.ID == wid {
					warnKey, releaseKey = pw.AutoCloseWarnKey, pw.AutoCloseReleaseKey
					break
				}
			}
		}
		windows = append(windows, scheduleRhythmWindow{
			ID:                   strField(wObj, "id"),
			TaskID:               t.ID,
			TenantID:             t.TenantID,
			WorkspaceID:          t.WorkspaceID,
			DailyStart:           start,
			DailyEnd:             end,
			MaxQueuedMachines:    maxN,
			AutoClose:            autoClose,
			AutoCloseWarnMinutes: autoCloseWarnMinutes,
			AutoCloseWarnKey:     warnKey,
			AutoCloseReleaseKey:  releaseKey,
			SortOrder:            i,
		})
	}

	// Validate no overlap between windows
	if len(windows) > 0 {
		if err := validateWindowsNoOverlap(windows); err != nil {
			return err
		}
	}

	// Assign IDs to windows that don't have one
	for i := range windows {
		if windows[i].ID == "" {
			windows[i].ID = generateWindowID(t.ID + "_" + strconv.Itoa(i))
		}
	}

	// Upsert task-level rhythm
	r := scheduleRhythm{
		TaskID:      t.ID,
		TenantID:    t.TenantID,
		WorkspaceID: t.WorkspaceID,
		Enabled:     enabled,
		Timezone:    tz,
	}
	if err := upsertScheduleRhythm(r); err != nil {
		return err
	}
	if err := upsertScheduleRhythmWindows(t.ID, t.TenantID, t.WorkspaceID, windows); err != nil {
		return err
	}
	_ = publishDomainEvent(context.Background(), "ScheduleRhythmUpdated", map[string]interface{}{
		"task_id":      t.ID,
		"tenant_id":    t.TenantID,
		"workspace_id": t.WorkspaceID,
		"enabled":      enabled,
		"window_count": len(windows),
	}, t.ID)
	return nil
}

// generateWindowID creates a unique ID for a schedule rhythm window.
func generateWindowID(taskID string) string {
	return taskID + "_win_" + time.Now().UTC().Format("20060102150405")
}
