package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// setupWorkspaceRhythmAllDay 配置工作空间全天窗口节奏（00:00–00:00 = 24h 窗口）。
func setupWorkspaceRhythmAllDay(t *testing.T, tenant, ws string, enabled bool, maxN int) {
	t.Helper()
	_ = upsertWorkspaceScheduleRhythm(workspaceScheduleRhythm{
		WorkspaceID: ws, TenantID: tenant, Enabled: enabled, Timezone: "UTC",
	})
	_ = upsertWorkspaceScheduleRhythmWindows(ws, tenant, []workspaceScheduleRhythmWindow{
		{ID: "ws_win0", WorkspaceID: ws, TenantID: tenant,
			DailyStart: "00:00", DailyEnd: "00:00", MaxQueuedMachines: maxN},
	})
}

func TestWorkspaceScheduleRhythmRoundtrip(t *testing.T) {
	setupTestDB(t)
	ws := "ws1"
	_ = upsertWorkspaceScheduleRhythm(workspaceScheduleRhythm{WorkspaceID: ws, TenantID: "t1", Enabled: true, Timezone: "Asia/Shanghai"})
	_ = upsertWorkspaceScheduleRhythmWindows(ws, "t1", []workspaceScheduleRhythmWindow{
		{ID: "w0", WorkspaceID: ws, TenantID: "t1", DailyStart: "09:00", DailyEnd: "11:00", MaxQueuedMachines: 2, AutoClose: true, AutoCloseWarnMinutes: 10},
		{ID: "w1", WorkspaceID: ws, TenantID: "t1", DailyStart: "14:00", DailyEnd: "16:00", MaxQueuedMachines: 1},
	})

	r, err := loadWorkspaceScheduleRhythm(ws)
	if err != nil {
		t.Fatal(err)
	}
	if r == nil || !r.Enabled || r.Timezone != "Asia/Shanghai" {
		t.Fatalf("rhythm=%+v", r)
	}
	wins, err := loadWorkspaceScheduleRhythmWindows(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(wins) != 2 {
		t.Fatalf("windows=%d", len(wins))
	}
	if wins[0].SortOrder != 0 || wins[1].SortOrder != 1 {
		t.Fatalf("sort_order wrong: %+v", wins)
	}
	if !wins[0].AutoClose || wins[0].AutoCloseWarnMinutes != 10 || wins[1].AutoClose {
		t.Fatalf("auto_close wrong: %+v", wins)
	}

	// 重写窗口（delete+reinsert）后旧窗口消失
	_ = upsertWorkspaceScheduleRhythmWindows(ws, "t1", []workspaceScheduleRhythmWindow{
		{ID: "w2", WorkspaceID: ws, TenantID: "t1", DailyStart: "20:00", DailyEnd: "21:00", MaxQueuedMachines: 1},
	})
	wins, _ = loadWorkspaceScheduleRhythmWindows(ws)
	if len(wins) != 1 || wins[0].ID != "w2" {
		t.Fatalf("rewrite windows: %+v", wins)
	}
}

func TestApplyWorkspaceScheduleRhythmFromBodyValidation(t *testing.T) {
	setupTestDB(t)
	base := func() map[string]interface{} {
		return map[string]interface{}{
			"enabled":  true,
			"timezone": "UTC",
			"windows": []interface{}{
				map[string]interface{}{"daily_start": "09:00", "daily_end": "11:00", "max_queued_machines": 1},
			},
		}
	}
	// 缺 windows
	b := base()
	delete(b, "windows")
	if err := applyWorkspaceScheduleRhythmFromBody("t1", "ws1", "", b); err == nil {
		t.Fatal("want error for missing windows")
	}
	// 非法 HH:MM
	b = base()
	b["windows"] = []interface{}{map[string]interface{}{"daily_start": "25:00", "daily_end": "11:00", "max_queued_machines": 1}}
	if err := applyWorkspaceScheduleRhythmFromBody("t1", "ws1", "", b); err == nil || !strings.Contains(err.Error(), "HH:MM") {
		t.Fatalf("want HH:MM error, got %v", err)
	}
	// 交叠窗口
	b = base()
	b["windows"] = []interface{}{
		map[string]interface{}{"daily_start": "09:00", "daily_end": "11:00", "max_queued_machines": 1},
		map[string]interface{}{"daily_start": "10:00", "daily_end": "12:00", "max_queued_machines": 1},
	}
	if err := applyWorkspaceScheduleRhythmFromBody("t1", "ws1", "", b); err == nil {
		t.Fatal("want overlap error")
	}
	// 合法：保存成功 + 无 ID 窗口自动生成 ID + 空 start/end 也可存（全天语义由前端控制）
	b = base()
	if err := applyWorkspaceScheduleRhythmFromBody("t1", "ws1", "", b); err != nil {
		t.Fatal(err)
	}
	r, _ := loadWorkspaceScheduleRhythm("ws1")
	if r == nil || !r.Enabled || r.Timezone != "UTC" {
		t.Fatalf("saved rhythm=%+v", r)
	}
	wins, _ := loadWorkspaceScheduleRhythmWindows("ws1")
	if len(wins) != 1 || wins[0].ID == "" {
		t.Fatalf("windows=%+v", wins)
	}
}

func TestWorkspaceRhythmInWindow(t *testing.T) {
	setupTestDB(t)
	if workspaceRhythmInWindow(nil, time.Now()) {
		t.Fatal("nil rhythm must not be in window")
	}
	// 未启用
	setupWorkspaceRhythmAllDay(t, "t1", "ws1", false, 1)
	r, _ := loadWorkspaceScheduleRhythm("ws1")
	if workspaceRhythmInWindow(r, time.Now()) {
		t.Fatal("disabled rhythm must not be in window")
	}
	// 全天窗口
	setupWorkspaceRhythmAllDay(t, "t1", "ws1", true, 1)
	r, _ = loadWorkspaceScheduleRhythm("ws1")
	if !workspaceRhythmInWindow(r, time.Now()) {
		t.Fatal("all-day window should be in window")
	}
	// 错位窗口：当前时刻必然不在 [M+2, M+3)
	start, end := shiftedWindowMinutes(time.Now())
	_ = upsertWorkspaceScheduleRhythmWindows("ws1", "t1", []workspaceScheduleRhythmWindow{
		{ID: "w0", WorkspaceID: "ws1", TenantID: "t1", DailyStart: start, DailyEnd: end, MaxQueuedMachines: 1},
	})
	r, _ = loadWorkspaceScheduleRhythm("ws1")
	if workspaceRhythmInWindow(r, time.Now()) {
		t.Fatalf("window %s-%s should NOT contain now", start, end)
	}
}

func TestLoadEffectiveWorkspaceRhythmFallback(t *testing.T) {
	setupTestDB(t)
	if _, wsScoped := loadEffectiveWorkspaceRhythm("nows"); wsScoped {
		t.Fatal("unconfigured workspace must fall back")
	}
	setupWorkspaceRhythmAllDay(t, "t1", "ws1", true, 1)
	r, wsScoped := loadEffectiveWorkspaceRhythm("ws1")
	if !wsScoped || r == nil {
		t.Fatal("configured workspace must be scoped")
	}
}

// shiftedWindowMinutes 返回一个当前时刻必然不落在其中的窗口（HH:MM）。
func shiftedWindowMinutes(now time.Time) (string, string) {
	mins := now.UTC().Hour()*60 + now.UTC().Minute()
	start, end := mins+2, mins+3
	if end >= 1440 {
		start, end = mins-59, mins-58
	}
	return fmt.Sprintf("%02d:%02d", start/60, start%60), fmt.Sprintf("%02d:%02d", end/60, end%60)
}

func enqueueTestMembers(t *testing.T, tenant, ws string, taskIDs ...string) {
	t.Helper()
	for _, tid := range taskIDs {
		rec, err := loadTask(tid)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := enqueueQueuedAutoRun(rec, "u-test"); err != nil {
			t.Fatal(err)
		}
	}
}

func membershipStatus(t *testing.T, taskID string) string {
	t.Helper()
	m, err := loadMembership(taskID)
	if err != nil || m == nil {
		t.Fatalf("membership %s: %v", taskID, err)
	}
	return m.Status
}

func TestDispatchWorkspaceQueueLegacyFallback(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	// 工作空间未配置节奏 → 回退任务级（legacy）
	_ = upsertScheduleRhythm(scheduleRhythm{TaskID: top, TenantID: tenant, WorkspaceID: ws, Enabled: true, Timezone: "UTC"})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws, DailyStart: "00:00", DailyEnd: "00:00", MaxQueuedMachines: 1},
	})
	enqueueTestMembers(t, tenant, ws, top)

	var started []string
	stub := func(m queuedMembership) error { started = append(started, m.TaskID); return nil }
	dispatchWorkspaceQueueWith(ws, stub)
	if len(started) != 1 || started[0] != top {
		t.Fatalf("legacy fallback: started=%v", started)
	}
}

func TestDispatchWorkspaceQueueDisabled(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	c1 := insertQueuedTestTask(t, tenant, ws, "c1", top)
	// 先启用并入队（status=queued），再改为禁用
	setupWorkspaceRhythmAllDay(t, tenant, ws, true, 5)
	enqueueTestMembers(t, tenant, ws, top, c1)
	if membershipStatus(t, top) != "queued" {
		t.Fatalf("precondition: top=%s", membershipStatus(t, top))
	}
	_ = upsertWorkspaceScheduleRhythm(workspaceScheduleRhythm{WorkspaceID: ws, TenantID: tenant, Enabled: false, Timezone: "UTC"})

	var started []string
	stub := func(m queuedMembership) error { started = append(started, m.TaskID); return nil }
	dispatchWorkspaceQueueWith(ws, stub)
	if len(started) != 0 {
		t.Fatalf("disabled must not start: %v", started)
	}
	if s := membershipStatus(t, top); s != "deferred" {
		t.Fatalf("top status=%s", s)
	}
	if s := membershipStatus(t, c1); s != "deferred" {
		t.Fatalf("c1 status=%s", s)
	}
}

func TestDispatchWorkspaceQueueOutOfWindow(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	c1 := insertQueuedTestTask(t, tenant, ws, "c1", top)
	// 先全天窗口入队，再把窗口改成当前时刻之外
	setupWorkspaceRhythmAllDay(t, tenant, ws, true, 5)
	enqueueTestMembers(t, tenant, ws, top, c1)
	start, end := shiftedWindowMinutes(time.Now())
	_ = upsertWorkspaceScheduleRhythmWindows(ws, tenant, []workspaceScheduleRhythmWindow{
		{ID: "w0", WorkspaceID: ws, TenantID: tenant, DailyStart: start, DailyEnd: end, MaxQueuedMachines: 5},
	})

	var started []string
	stub := func(m queuedMembership) error { started = append(started, m.TaskID); return nil }
	dispatchWorkspaceQueueWith(ws, stub)
	if len(started) != 0 {
		t.Fatalf("out-of-window must not start: %v", started)
	}
	if s := membershipStatus(t, top); s != "deferred" {
		t.Fatalf("top status=%s", s)
	}
}

func TestDispatchWorkspaceQueueWithinWindowOrdered(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	c1 := insertQueuedTestTask(t, tenant, ws, "c1", top)
	c2 := insertQueuedTestTask(t, tenant, ws, "c2", top)
	setupWorkspaceRhythmAllDay(t, tenant, ws, true, 2)
	enqueueTestMembers(t, tenant, ws, top, c1, c2)

	var started []string
	stub := func(m queuedMembership) error { started = append(started, m.TaskID); return nil }
	dispatchWorkspaceQueueWith(ws, stub)
	if len(started) != 2 {
		t.Fatalf("maxN=2 want 2 started, got %v", started)
	}
	if started[0] != top {
		t.Fatalf("depth-first: first=%s", started[0])
	}
	// 第 3 个成员保持 queued（槽位不足）
	rest := map[string]bool{top: true, c1: true, c2: true}
	for _, s := range started {
		delete(rest, s)
	}
	for tid := range rest {
		if s := membershipStatus(t, tid); s != "queued" {
			t.Fatalf("remaining %s status=%s", tid, s)
		}
	}
}

func TestDispatchWorkspaceQueueSlotLimit(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	c1 := insertQueuedTestTask(t, tenant, ws, "c1", top)
	setupWorkspaceRhythmAllDay(t, tenant, ws, true, 1)
	enqueueTestMembers(t, tenant, ws, top, c1)
	// 预占槽位：c1 已有机器在跑 → 无空槽
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`, c1, top, time.Now().UTC())
	if n := countWorkspaceQueuedSlots(ws); n != 1 {
		t.Fatalf("slots=%d", n)
	}

	var started []string
	stub := func(m queuedMembership) error { started = append(started, m.TaskID); return nil }
	dispatchWorkspaceQueueWith(ws, stub)
	if len(started) != 0 {
		t.Fatalf("slot limit reached, want 0 started, got %v", started)
	}
}

func TestListWorkspaceQueueItemsAndSlots(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	c1 := insertQueuedTestTask(t, tenant, ws, "child-1", top)
	enqueueTestMembers(t, tenant, ws, top, c1)
	_, _ = db.Exec(`INSERT INTO task_queued_machine_slots(task_id,top_task_id,acquired_at) VALUES(?,?,?)`, c1, top, time.Now().UTC())

	items, err := listWorkspaceQueueItems(ws)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items=%d", len(items))
	}
	byID := map[string]map[string]interface{}{}
	for _, it := range items {
		byID[it["task_id"].(string)] = it
	}
	if byID[c1]["title"] != "child-1" || byID[top]["title"] != "top" {
		t.Fatalf("titles: %v %v", byID[c1]["title"], byID[top]["title"])
	}
	if n := countWorkspaceQueuedSlots(ws); n != 1 {
		t.Fatalf("workspace slots=%d", n)
	}
}

func TestWorkspaceQueueScheduleHandlers(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")

	// GET 未配置 → 200 + schedule_rhythm nil + members 含已入队任务
	enqueueTestMembers(t, tenant, ws, top)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+tenant+"/workspace/"+ws+"/queue-schedule/", nil)
	req.Header.Set("X-Auth-User-Id", "internal")
	rr := httptest.NewRecorder()
	handleWorkspaceQueueScheduleRoutes(rr, req, tenant, ws)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["schedule_rhythm"] != nil {
		t.Fatalf("unconfigured rhythm should be null: %v", body["schedule_rhythm"])
	}
	if len(body["members"].([]interface{})) != 1 {
		t.Fatalf("members=%v", body["members"])
	}
	if _, ok := body["recent_history_has_more"]; !ok {
		t.Fatalf("snapshot missing recent_history_has_more: %v", body)
	}
	if body["recent_history_has_more"] != false {
		t.Fatalf("single-event snapshot has_more=%v want false", body["recent_history_has_more"])
	}

	// PUT 合法 → 200 + 保存生效（再 GET 一致）
	putBody := `{"enabled":true,"timezone":"UTC","windows":[{"id":"","daily_start":"09:00","daily_end":"11:00","max_queued_machines":2,"auto_close":false}]}`
	req = httptest.NewRequest(http.MethodPut, "/api/tenant/"+tenant+"/workspace/"+ws+"/queue-schedule/", strings.NewReader(putBody))
	req.Header.Set("X-Auth-User-Id", "internal")
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handleWorkspaceQueueScheduleRoutes(rr, req, tenant, ws)
	if rr.Code != http.StatusOK {
		t.Fatalf("PUT status=%d body=%s", rr.Code, rr.Body.String())
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	rhythm := body["schedule_rhythm"].(map[string]interface{})
	if rhythm["enabled"] != true {
		t.Fatalf("rhythm=%v", rhythm)
	}
	wins := rhythm["windows"].([]interface{})
	if len(wins) != 1 {
		t.Fatalf("windows=%v", wins)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/tenant/"+tenant+"/workspace/"+ws+"/queue-schedule/", nil)
	req.Header.Set("X-Auth-User-Id", "internal")
	rr = httptest.NewRecorder()
	handleWorkspaceQueueScheduleRoutes(rr, req, tenant, ws)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET2 status=%d", rr.Code)
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["schedule_rhythm"].(map[string]interface{})["enabled"] != true {
		t.Fatal("PUT did not persist")
	}

	// PUT 非法 HH:MM → 400
	badBody := `{"enabled":true,"timezone":"UTC","windows":[{"daily_start":"9am","daily_end":"11:00","max_queued_machines":1}]}`
	req = httptest.NewRequest(http.MethodPut, "/api/tenant/"+tenant+"/workspace/"+ws+"/queue-schedule/", strings.NewReader(badBody))
	req.Header.Set("X-Auth-User-Id", "internal")
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	handleWorkspaceQueueScheduleRoutes(rr, req, tenant, ws)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad PUT status=%d body=%s", rr.Code, rr.Body.String())
	}
}
