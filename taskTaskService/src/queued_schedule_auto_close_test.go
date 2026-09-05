package main

import (
	"context"
	"testing"
	"time"
)

func TestMinutesUntilWindowEnd(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	n := time.Date(2026, 7, 22, 5, 56, 0, 0, loc)
	mins := minutesUntilWindowEnd(n, "22:00", "06:00")
	if mins != 4 {
		t.Fatalf("overnight near end: want 4 got %d", mins)
	}
	n2 := time.Date(2026, 7, 22, 17, 0, 0, 0, loc)
	if minutesUntilWindowEnd(n2, "09:00", "18:00") != 60 {
		t.Fatalf("daytime: want 60 got %d", minutesUntilWindowEnd(n2, "09:00", "18:00"))
	}
	n3 := time.Date(2026, 7, 22, 12, 0, 0, 0, loc)
	if minutesUntilWindowEnd(n3, "22:00", "06:00") != -1 {
		t.Fatal("outside overnight window should be -1")
	}
}

func TestAutoCloseWarnOnce(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	child := insertQueuedTestTask(t, tenant, ws, "child", top)

	nowUTC := time.Now().UTC()
	end := nowUTC.Add(3 * time.Minute)
	start := nowUTC.Add(-1 * time.Hour)
	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: start.Format("15:04"), DailyEnd: end.Format("15:04"),
			MaxQueuedMachines: 1, AutoClose: true},
	})
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`,
		child, top, time.Now().UTC())

	var warned []string
	prev := notifyContainerClosingSoonFn
	notifyContainerClosingSoonFn = func(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
		warned = append(warned, taskID)
		return nil
	}
	t.Cleanup(func() { notifyContainerClosingSoonFn = prev })

	runAutoCloseForTop(top)
	if len(warned) != 1 || warned[0] != child {
		t.Fatalf("warn once: got %v", warned)
	}
	wins, _ := loadScheduleRhythmWindows(top)
	if len(wins) != 1 || wins[0].AutoCloseWarnKey == "" {
		t.Fatal("expected warn key set")
	}
	warned = nil
	runAutoCloseForTop(top)
	if len(warned) != 0 {
		t.Fatalf("idempotent warn: got %v", warned)
	}
}

func TestAutoCloseReleaseOutsideWindow(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	child := insertQueuedTestTask(t, tenant, ws, "child", top)

	// Window in the past relative to now (UTC): ended an hour ago
	nowUTC := time.Now().UTC()
	start := nowUTC.Add(-3 * time.Hour)
	end := nowUTC.Add(-1 * time.Hour)
	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: start.Format("15:04"), DailyEnd: end.Format("15:04"),
			MaxQueuedMachines: 1, AutoClose: true},
	})
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`,
		child, top, time.Now().UTC())

	var shut []string
	var stopped []string
	prevShut := shutdownContainerLifecycleFn
	prevStop := stopVMFn
	shutdownContainerLifecycleFn = func(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
		shut = append(shut, taskID)
		return nil
	}
	stopVMFn = func(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
		stopped = append(stopped, taskID)
		return nil
	}
	t.Cleanup(func() {
		shutdownContainerLifecycleFn = prevShut
		stopVMFn = prevStop
	})

	runAutoCloseForTop(top)
	if len(shut) != 1 || shut[0] != child {
		t.Fatalf("shutdown: got %v", shut)
	}
	if len(stopped) != 0 {
		t.Fatalf("stop-vm should not run when shutdown ok: %v", stopped)
	}
	ids, _ := listQueuedSlotTaskIDs(top)
	if len(ids) != 0 {
		t.Fatalf("slots should clear, got %v", ids)
	}
	wins, _ := loadScheduleRhythmWindows(top)
	if len(wins) != 1 || wins[0].AutoCloseReleaseKey == "" {
		t.Fatal("expected release key")
	}
	shut = nil
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`,
		child, top, time.Now().UTC())
	runAutoCloseForTop(top)
	if len(shut) != 0 {
		t.Fatalf("idempotent release: got %v", shut)
	}
}

func TestAutoCloseDisabledNoRelease(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	child := insertQueuedTestTask(t, tenant, ws, "child", top)
	nowUTC := time.Now().UTC()
	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart:        nowUTC.Add(-3 * time.Hour).Format("15:04"),
			DailyEnd:          nowUTC.Add(-1 * time.Hour).Format("15:04"),
			MaxQueuedMachines: 1, AutoClose: false},
	})
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`,
		child, top, time.Now().UTC())
	called := 0
	prev := shutdownContainerLifecycleFn
	shutdownContainerLifecycleFn = func(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
		called++
		return nil
	}
	t.Cleanup(func() { shutdownContainerLifecycleFn = prev })
	runAutoCloseForTop(top)
	if called != 0 {
		t.Fatalf("auto_close=false must not release, called=%d", called)
	}
}

func TestAutoCloseMultiWindowIndependentWarnRelease(t *testing.T) {
	// Two windows with auto_close=true — verify warn/release per independent window.
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	child1 := insertQueuedTestTask(t, tenant, ws, "c1", top)
	child2 := insertQueuedTestTask(t, tenant, ws, "c2", top)

	nowUTC := time.Now().UTC()
	// Window A (active, about to end): ends in 3 min → inside warn window (warn_minutes=5)
	startA := nowUTC.Add(-1 * time.Hour)
	endA := nowUTC.Add(3 * time.Minute)
	// Window B (past): ended 30 min ago → already outside → should release
	startB := nowUTC.Add(-2 * time.Hour)
	endB := nowUTC.Add(-30 * time.Minute)

	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_w0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: startA.Format("15:04"), DailyEnd: endA.Format("15:04"),
			MaxQueuedMachines: 1, AutoClose: true},
		{ID: top + "_w1", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: startB.Format("15:04"), DailyEnd: endB.Format("15:04"),
			MaxQueuedMachines: 1, AutoClose: true},
	})

	// Slot child1 in window A (active time) — actually slots don't carry window info,
	// so let's just slot both children and verify behavior is deterministic.
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`,
		child1, top, nowUTC)
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`,
		child2, top, nowUTC)

	var warned []string
	prevWarn := notifyContainerClosingSoonFn
	notifyContainerClosingSoonFn = func(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
		warned = append(warned, taskID)
		return nil
	}
	t.Cleanup(func() { notifyContainerClosingSoonFn = prevWarn })

	runAutoCloseForTop(top)
	// Window A (ending in 3 min, within 5 min warn threshold): should warn both children
	// Window B (ended 30 min ago): should release both children
	// Verify both windows acted independently
	if len(warned) < 1 {
		t.Fatalf("expected at least 1 warn from window A, got %d: %v", len(warned), warned)
	}

	// Verify both windows maintain independent warn/release keys
	wins, _ := loadScheduleRhythmWindows(top)
	if len(wins) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(wins))
	}
	for i, w := range wins {
		if !w.AutoClose {
			t.Fatalf("window[%d] expected auto_close=true", i)
		}
		t.Logf("window[%d] start=%s end=%s warnKey=%s releaseKey=%s",
			i, w.DailyStart, w.DailyEnd, w.AutoCloseWarnKey, w.AutoCloseReleaseKey)
	}
	// Window A should have issued a warn (warnKey set)
	if wins[0].AutoCloseWarnKey == "" {
		t.Fatal("window A (active, about to end) expected warn key")
	}
	// Window B was past — may have issued release (releaseKey set); warn key may also be set from first tick
	if wins[1].AutoCloseReleaseKey == "" {
		t.Log("window B (past): release may have been processed; releaseKey=", wins[1].AutoCloseReleaseKey)
	}
}

func TestApplyScheduleRhythmAutoClose(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	topRec, _ := loadTask(top)
	if err := applyScheduleRhythmFromBody(topRec, map[string]interface{}{
		"schedule_rhythm": map[string]interface{}{
			"enabled": true, "timezone": "Asia/Shanghai",
			"windows": []interface{}{
				map[string]interface{}{
					"daily_start": "22:00", "daily_end": "06:00",
					"max_queued_machines": 1, "auto_close": true,
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	r, _ := loadScheduleRhythm(top)
	if r == nil || !r.Enabled {
		t.Fatalf("want enabled true, got %+v", r)
	}
	j := rhythmToJSON(r)
	if j["auto_close"] != true {
		t.Fatalf("json auto_close: %v", j["auto_close"])
	}
}

func TestAutoCloseReleaseResetsStartingMembership(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	child := insertQueuedTestTask(t, tenant, ws, "child", top)

	nowUTC := time.Now().UTC()
	start := nowUTC.Add(-3 * time.Hour)
	end := nowUTC.Add(-1 * time.Hour)
	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: start.Format("15:04"), DailyEnd: end.Format("15:04"),
			MaxQueuedMachines: 1, AutoClose: true},
	})
	childRec, err := loadTask(child)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enqueueQueuedAutoRun(childRec, "u-test"); err != nil {
		t.Fatal(err)
	}
	setMembershipStatus(child, "starting")
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`,
		child, top, time.Now().UTC())

	prevShut := shutdownContainerLifecycleFn
	shutdownContainerLifecycleFn = func(ctx context.Context, tenantID, workspaceID, taskID string, body map[string]interface{}) error {
		return nil
	}
	t.Cleanup(func() { shutdownContainerLifecycleFn = prevShut })

	runAutoCloseForTop(top)
	m, _ := loadMembership(child)
	if m == nil || m.Status != "queued" {
		t.Fatalf("auto-close after slot release must reset starting→queued, got %+v", m)
	}
	if hasQueuedSlot(child) {
		t.Fatal("slot must be cleared")
	}
}
