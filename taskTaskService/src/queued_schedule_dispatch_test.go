package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// OPT-20260816-030：排队自动开跑迁 taskEvents queued_auto_run_scan timer，
// taskTaskService 仅暴露一次性 HTTP 端点（不再自带 30s 进程内 ticker）。
func TestHandleInternalQueuedScheduleDispatchOnce(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tasks/queued-schedule/dispatch-once/", strings.NewReader(`{}`))
	req.Header.Set("X-Internal-Secret", "sec")
	rec := httptest.NewRecorder()
	handleInternalQueuedScheduleDispatchOnce(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"success"`) {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestHandleInternalQueuedScheduleDispatchOnceMethodNotAllowed(t *testing.T) {
	setupTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/tasks/queued-schedule/dispatch-once/", nil)
	rec := httptest.NewRecorder()
	handleInternalQueuedScheduleDispatchOnce(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", rec.Code)
	}
}

func TestHandleInternalQueuedScheduleDispatchOnceForbiddenWithoutSecret(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "sec"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	req := httptest.NewRequest(http.MethodPost, "/api/internal/tasks/queued-schedule/dispatch-once/", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handleInternalQueuedScheduleDispatchOnce(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rec.Code)
	}
}

// OPT-20260816-030 回归：移除进程内 ticker 后，文件不得再引用 time.NewTicker/startQueuedScheduleTicker。
func TestQueuedScheduleDispatchNoLegacyTicker(t *testing.T) {
	src, err := os.ReadFile("queued_schedule_dispatch.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"time.NewTicker", "startQueuedScheduleTicker"} {
		if strings.Contains(string(src), needle) {
			t.Fatalf("queued_schedule_dispatch.go must not reference %q (ticker 已迁 timer)", needle)
		}
	}
}

func insertStartingMembership(t *testing.T, tenant, ws, title string) (taskID string) {
	t.Helper()
	taskID = insertQueuedTestTask(t, tenant, ws, title, "")
	rec, err := loadTask(taskID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enqueueQueuedAutoRun(rec, "u-test"); err != nil {
		t.Fatal(err)
	}
	setMembershipStatus(taskID, "starting")
	return taskID
}

// 无槽位的 starting 必须收回 queued 并再次 start，否则 UI 会卡在「调度启服中」且占用 0。
func TestDispatchReclaimsStartingWithoutSlot(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	setupWorkspaceRhythmAllDay(t, tenant, ws, true, 1)
	taskID := insertStartingMembership(t, tenant, ws, "stuck")
	if hasQueuedSlot(taskID) {
		t.Fatal("fixture must have no slot")
	}

	var started []string
	dispatchWorkspaceQueueWith(ws, func(m queuedMembership) error {
		started = append(started, m.TaskID)
		return nil
	})
	if len(started) != 1 || started[0] != taskID {
		t.Fatalf("want reclaim+start of %s, got %v", taskID, started)
	}
}

// 已占槽的 starting 表示启服进行中，不得重复 startVM。
func TestDispatchSkipsStartingWithSlot(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	setupWorkspaceRhythmAllDay(t, tenant, ws, true, 1)
	taskID := insertStartingMembership(t, tenant, ws, "inflight")
	if err := acquireQueuedSlot(taskID, taskID); err != nil {
		t.Fatal(err)
	}

	var started []string
	dispatchWorkspaceQueueWith(ws, func(m queuedMembership) error {
		started = append(started, m.TaskID)
		return nil
	})
	if len(started) != 0 {
		t.Fatalf("must skip starting-with-slot, got %v", started)
	}
}

// 窗口外：无槽 starting 视为卡死应收成 deferred；有槽 starting 保持启服中。
func TestDeferAllMembersOutOfWindowStaleStarting(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	stale := insertQueuedTestTask(t, tenant, ws, "stale", top)
	inflight := insertQueuedTestTask(t, tenant, ws, "inflight", top)
	for _, id := range []string{stale, inflight} {
		rec, err := loadTask(id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := enqueueQueuedAutoRun(rec, "u-test"); err != nil {
			t.Fatal(err)
		}
		setMembershipStatus(id, "starting")
	}
	if err := acquireQueuedSlot(inflight, top); err != nil {
		t.Fatal(err)
	}

	members, err := listMembershipsForTop(top)
	if err != nil {
		t.Fatal(err)
	}
	deferAllMembers(members, "窗口外", true)

	ms, _ := loadMembership(stale)
	if ms == nil || ms.Status != "deferred" {
		t.Fatalf("stale starting without slot want deferred, got %+v", ms)
	}
	mi, _ := loadMembership(inflight)
	if mi == nil || mi.Status != "starting" {
		t.Fatalf("starting with slot want keep starting, got %+v", mi)
	}
}

func TestLegacyDispatchReclaimsStartingWithoutSlot(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws-legacy"
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	_ = upsertScheduleRhythm(scheduleRhythm{
		TaskID: top, TenantID: tenant, WorkspaceID: ws,
		Enabled: true, Timezone: "UTC",
	})
	_ = upsertScheduleRhythmWindows(top, tenant, ws, []scheduleRhythmWindow{
		{ID: top + "_win0", TaskID: top, TenantID: tenant, WorkspaceID: ws,
			DailyStart: "00:00", DailyEnd: "00:00", MaxQueuedMachines: 1},
	})
	rec, err := loadTask(top)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enqueueQueuedAutoRun(rec, "u-test"); err != nil {
		t.Fatal(err)
	}
	setMembershipStatus(top, "starting")

	var started []string
	dispatchTopQueueWith(top, func(m queuedMembership) error {
		started = append(started, m.TaskID)
		return nil
	})
	if len(started) != 1 || started[0] != top {
		t.Fatalf("legacy want reclaim+start of %s, got %v", top, started)
	}
}

func TestQueuedSlotAcquireRelease(t *testing.T) {
	setupTestDB(t)
	id := insertQueuedTestTask(t, "t1", "ws1", "t", "")
	if hasQueuedSlot(id) {
		t.Fatal("new task must have no slot")
	}
	if err := acquireQueuedSlot(id, id); err != nil {
		t.Fatal(err)
	}
	if !hasQueuedSlot(id) {
		t.Fatal("want slot after acquire")
	}
	releaseQueuedSlot(id)
	if hasQueuedSlot(id) {
		t.Fatal("want no slot after release")
	}
}
