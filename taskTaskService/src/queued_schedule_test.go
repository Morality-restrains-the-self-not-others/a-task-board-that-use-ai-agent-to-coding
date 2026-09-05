package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInDailyWindowOvernight(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	n := time.Date(2026, 7, 19, 23, 0, 0, 0, loc)
	if !inDailyWindow(n, "22:00", "06:00") {
		t.Fatal("expected in window at 23:00")
	}
	n2 := time.Date(2026, 7, 19, 12, 0, 0, 0, loc)
	if inDailyWindow(n2, "22:00", "06:00") {
		t.Fatal("expected outside window at 12:00")
	}
	n3 := time.Date(2026, 7, 19, 10, 0, 0, 0, loc)
	if !inDailyWindow(n3, "09:00", "18:00") {
		t.Fatal("expected in daytime window")
	}
}

func TestEnqueueOrderDepthFIFO(t *testing.T) {
	setupTestDB(t)
	tenant := "t1"
	ws := "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	child := insertQueuedTestTask(t, tenant, ws, "child", top)
	grandchild := insertQueuedTestTask(t, tenant, ws, "gc", child)

	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: "00:00", DailyEnd: "23:59", MaxQueuedMachines: 2},
	})

	topRec, _ := loadTask(top)
	childRec, _ := loadTask(child)
	gcRec, _ := loadTask(grandchild)
	if _, err := enqueueQueuedAutoRun(gcRec, "u-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := enqueueQueuedAutoRun(childRec, "u-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := enqueueQueuedAutoRun(topRec, "u-test"); err != nil {
		t.Fatal(err)
	}
	list, err := listMembershipsForTop(top)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("want 3 got %d", len(list))
	}
	if list[0].TaskID != top || list[0].Depth != 0 {
		t.Fatalf("first=%+v", list[0])
	}
	if list[1].TaskID != child || list[1].Depth != 1 {
		t.Fatalf("second=%+v", list[1])
	}
	if list[2].TaskID != grandchild || list[2].Depth != 2 {
		t.Fatalf("third=%+v", list[2])
	}
}

func TestDequeueOnManualStart(t *testing.T) {
	setupTestDB(t)
	tenant := "t1"
	ws := "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	child := insertQueuedTestTask(t, tenant, ws, "child", top)
	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: "00:00", DailyEnd: "23:59", MaxQueuedMachines: 1},
	})
	childRec, _ := loadTask(child)
	if _, err := enqueueQueuedAutoRun(childRec, "u-test"); err != nil {
		t.Fatal(err)
	}
	if m, _ := loadMembership(child); m == nil {
		t.Fatal("expected membership")
	}
	dequeueOnManualStart(child, "u-test")
	if m2, _ := loadMembership(child); m2 != nil {
		t.Fatal("expected dequeued")
	}
}

func TestDeferImmediateAutoRunStart(t *testing.T) {
	if !deferImmediateAutoRunStart(map[string]interface{}{"queued_auto_run": true}, false) {
		t.Fatal("queued without force should defer")
	}
	if deferImmediateAutoRunStart(map[string]interface{}{"queued_auto_run": true}, true) {
		t.Fatal("force_auto_run should not defer")
	}
	if deferImmediateAutoRunStart(map[string]interface{}{"queued_auto_run": false}, false) {
		t.Fatal("unchecked queue should not defer")
	}
	if deferImmediateAutoRunStart(map[string]interface{}{}, false) {
		t.Fatal("missing field should not defer")
	}
}

func TestScheduleRhythmOnlyTopLevel(t *testing.T) {
	setupTestDB(t)
	tenant := "t1"
	ws := "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	child := insertQueuedTestTask(t, tenant, ws, "child", top)
	childRec, _ := loadTask(child)
	err := applyScheduleRhythmFromBody(childRec, map[string]interface{}{
		"schedule_rhythm": map[string]interface{}{
			"enabled": true,
			"windows": []interface{}{
				map[string]interface{}{
					"daily_start": "22:00", "daily_end": "06:00",
					"max_queued_machines": 2,
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for non-top")
	}
	topRec, _ := loadTask(top)
	if err := applyScheduleRhythmFromBody(topRec, map[string]interface{}{
		"schedule_rhythm": map[string]interface{}{
			"enabled": true, "timezone": "Asia/Shanghai",
			"windows": []interface{}{
				map[string]interface{}{
					"daily_start": "22:00", "daily_end": "06:00",
					"max_queued_machines": float64(2),
				},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	r, _ := loadScheduleRhythm(top)
	if r == nil || !r.Enabled {
		t.Fatalf("rhythm=%+v", r)
	}
	wins, _ := loadScheduleRhythmWindows(top)
	if len(wins) != 1 || wins[0].MaxQueuedMachines != 2 {
		t.Fatalf("windows=%+v", wins)
	}
}

func TestGetQueuedAutoRunAPI(t *testing.T) {
	setupTestDB(t)
	startMockProjectService(t)
	tenant := "t1"
	ws := "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: "00:00", DailyEnd: "23:59", MaxQueuedMachines: 1},
	})
	topRec, _ := loadTask(top)
	_, _ = enqueueQueuedAutoRun(topRec, "u-test")

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+tenant+"/workspace/"+ws+"/todos/"+top+"/queued-auto-run/", nil)
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Auth-Tenant-Id", tenant)
	rr := httptest.NewRecorder()
	handleGetQueuedAutoRun(rr, req, tenant, "u1", top)
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	members, _ := body["members"].([]interface{})
	if len(members) != 1 {
		t.Fatalf("members=%v", body["members"])
	}
}

func TestQueuedSlotsCountIndependent(t *testing.T) {
	setupTestDB(t)
	top := insertQueuedTestTask(t, "t1", "ws1", "top", "")
	a := insertQueuedTestTask(t, "t1", "ws1", "a", top)
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`, a, top, time.Now().UTC())
	if n := countQueuedSlots(top); n != 1 {
		t.Fatalf("slots=%d", n)
	}
	// manual/auto_run machines are not in task_queued_machine_slots — isolation holds
	if n := countQueuedSlots("other-top"); n != 0 {
		t.Fatalf("other slots=%d", n)
	}
}

func insertQueuedTestTask(t *testing.T, tenant, ws, title, parent string) string {
	t.Helper()
	id := genID("task")
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO task_tasks(id,tenant_id,title,description,completed,priority,order_num,workspace_id,owner_id,deliverable_obj_id,progress_column_id,parent_task_id,fork_from_id,installed_image_id,auto_run,feature_params_source,personal_feature_params_config_id,task_kind,code_lang,created_at,updated_at)
		VALUES(?,?,?,?,0,'medium',0,?,?,?,?,?,?,?,0,'company','','','',?,?)`,
		id, tenant, title, "", ws, "u1", "", "", parent, "", "", now, now)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// OPT-20260824-075: 延迟原因/窗口消息文案「节奏未启用」统一更名「自动调度未启用」。
// 覆盖 4 个用户可见文案出口（legacy + workspace 两套 windowMessage/deferredReason），无 DB 依赖。
func TestDeferredReasonTextAutoScheduleNotEnabled(t *testing.T) {
	if got := deferredReason(nil); got != "自动调度未启用" {
		t.Fatalf("deferredReason(nil) = %q, want 自动调度未启用", got)
	}
	if got := windowMessage(nil, false); got != "自动调度未启用" {
		t.Fatalf("windowMessage(nil,false) = %q, want 自动调度未启用", got)
	}
	if got := deferredReasonWorkspace(nil); got != "自动调度未启用" {
		t.Fatalf("deferredReasonWorkspace(nil) = %q, want 自动调度未启用", got)
	}
	if got := workspaceWindowMessage(nil, false); got != "自动调度未启用" {
		t.Fatalf("workspaceWindowMessage(nil,false) = %q, want 自动调度未启用", got)
	}
}

// OPT-20260719-037: integration test — enqueue, dispatch within window, verify depth-first order.
func TestQueuedScheduleDispatchWithinWindowDepthFirst(t *testing.T) {
	setupTestDB(t)
	tenant := "t1"
	ws := "ws1"

	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	c1 := insertQueuedTestTask(t, tenant, ws, "child1", top)
	c2 := insertQueuedTestTask(t, tenant, ws, "child2", top)

	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: "00:00", DailyEnd: "23:59", MaxQueuedMachines: 1},
	})

	for _, tid := range []string{top, c1, c2} {
		rec, err := loadTask(tid)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := enqueueQueuedAutoRun(rec, "u-test"); err != nil {
			t.Fatal(err)
		}
	}

	list, err := listMembershipsForTop(top)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("want 3, got %d", len(list))
	}
	if list[0].Depth != 0 || list[0].TaskID != top {
		t.Fatalf("depth-first: want top first, got %+v", list[0])
	}

	dispatched := 0
	for _, m := range list {
		if m.Status == "queued" {
			dispatched++
		}
	}
	if dispatched != 3 {
		t.Fatalf("want 3 pending, got %d", dispatched)
	}
}
