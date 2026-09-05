package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func countScheduleHistory(t *testing.T, workspaceID, eventType string) int {
	t.Helper()
	q := `SELECT COUNT(1) FROM task_queued_schedule_history WHERE workspace_id=?`
	args := []interface{}{workspaceID}
	if eventType != "" {
		q += ` AND event_type=?`
		args = append(args, eventType)
	}
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAppendAndListScheduleHistoryIsolated(t *testing.T) {
	setupTestDB(t)
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: "t1", WorkspaceID: "ws1", EventType: scheduleHistoryEventMemberEnqueued,
		TaskID: "task_a", Message: "加入自动调度队列：A",
	})
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: "t1", WorkspaceID: "ws1", EventType: scheduleHistoryEventRhythmSaved,
		Message: "已保存调度设置",
	})
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: "t1", WorkspaceID: "ws2", EventType: scheduleHistoryEventMemberEnqueued,
		TaskID: "task_b", Message: "other ws",
	})
	items, next, hasMore, err := listScheduleHistory("t1", "ws1", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if hasMore || next != "" {
		t.Fatalf("hasMore=%v next=%q", hasMore, next)
	}
	if len(items) != 2 {
		t.Fatalf("ws1 items=%d %+v", len(items), items)
	}
	if items[0]["event_type"] != scheduleHistoryEventRhythmSaved {
		t.Fatalf("newest first: %+v", items[0])
	}
	other, _, _, err := listScheduleHistory("t1", "ws2", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 || other[0]["task_id"] != "task_b" {
		t.Fatalf("ws2=%+v", other)
	}
}

func TestListScheduleHistoryPagination(t *testing.T) {
	setupTestDB(t)
	for i := 0; i < 3; i++ {
		appendScheduleHistory(scheduleHistoryInput{
			TenantID: "t1", WorkspaceID: "ws1", EventType: scheduleHistoryEventMemberEnqueued,
			Message: "row", TaskID: "t" + string(rune('a'+i)),
		})
	}
	page1, cur, more, err := listScheduleHistory("t1", "ws1", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !more || cur == "" || len(page1) != 2 {
		t.Fatalf("page1 len=%d more=%v cur=%q", len(page1), more, cur)
	}
	page2, _, more2, err := listScheduleHistory("t1", "ws1", cur, 2)
	if err != nil {
		t.Fatal(err)
	}
	if more2 || len(page2) != 1 {
		t.Fatalf("page2 len=%d more=%v", len(page2), more2)
	}
	if page1[0]["id"] == page2[0]["id"] || page1[1]["id"] == page2[0]["id"] {
		t.Fatalf("overlap p1=%+v p2=%+v", page1, page2)
	}
}

func TestEnqueueWritesScheduleHistory(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	setupWorkspaceRhythmAllDay(t, tenant, ws, true, 1)
	top := insertQueuedTestTask(t, tenant, ws, "hist-task", "")
	enqueueTestMembers(t, tenant, ws, top)
	if n := countScheduleHistory(t, ws, scheduleHistoryEventMemberEnqueued); n != 1 {
		t.Fatalf("enqueued history=%d", n)
	}
	var actor string
	if err := db.QueryRow(`SELECT actor_user_id FROM task_queued_schedule_history
		WHERE workspace_id=? AND event_type=?`, ws, scheduleHistoryEventMemberEnqueued).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if actor == "" {
		t.Fatalf("enqueue history actor_user_id is empty")
	}
}

func TestWorkspaceQueueScheduleSnapshotHasMore(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	for i := 0; i < 9; i++ {
		appendScheduleHistory(scheduleHistoryInput{
			TenantID: tenant, WorkspaceID: ws, EventType: scheduleHistoryEventMemberEnqueued,
			TaskID: "t" + strconv.Itoa(i), Message: "row",
		})
	}
	snap, err := buildWorkspaceQueueScheduleSnapshot(tenant, ws)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(snap["recent_history"].([]map[string]interface{})); n != 8 {
		t.Fatalf("recent_history len=%d want 8", n)
	}
	if snap["recent_history_has_more"] != true {
		t.Fatalf("recent_history_has_more=%v want true", snap["recent_history_has_more"])
	}
}

func TestSaveRhythmWritesScheduleHistory(t *testing.T) {
	setupTestDB(t)
	err := applyWorkspaceScheduleRhythmFromBody("t1", "ws1", "u1", map[string]interface{}{
		"enabled":  true,
		"timezone": "UTC",
		"windows": []interface{}{
			map[string]interface{}{"daily_start": "09:00", "daily_end": "11:00", "max_queued_machines": 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := countScheduleHistory(t, "ws1", scheduleHistoryEventRhythmSaved); n != 1 {
		t.Fatalf("saved history=%d", n)
	}
}

func TestWindowTransitionWritesOnce(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	setupWorkspaceRhythmAllDay(t, tenant, ws, true, 1)
	top := insertQueuedTestTask(t, tenant, ws, "top", "")
	enqueueTestMembers(t, tenant, ws, top)

	stub := func(m queuedMembership) error { return nil }
	dispatchWorkspaceQueueWith(ws, stub)
	nEnter := countScheduleHistory(t, ws, scheduleHistoryEventWindowEntered)
	if nEnter != 1 {
		t.Fatalf("first dispatch window_entered=%d", nEnter)
	}
	dispatchWorkspaceQueueWith(ws, stub)
	if n := countScheduleHistory(t, ws, scheduleHistoryEventWindowEntered); n != 1 {
		t.Fatalf("second dispatch should not rewrite window_entered, got %d", n)
	}

	// disable → one window_exited
	setupWorkspaceRhythmAllDay(t, tenant, ws, false, 1)
	dispatchWorkspaceQueueWith(ws, stub)
	if n := countScheduleHistory(t, ws, scheduleHistoryEventWindowExited); n != 1 {
		t.Fatalf("disable window_exited=%d", n)
	}
	dispatchWorkspaceQueueWith(ws, stub)
	if n := countScheduleHistory(t, ws, scheduleHistoryEventWindowExited); n != 1 {
		t.Fatalf("repeat disable should not rewrite, got %d", n)
	}
}

func TestWorkspaceQueueScheduleHistoryHTTP(t *testing.T) {
	setupTestDB(t)
	tenant, ws := "t1", "ws1"
	appendScheduleHistory(scheduleHistoryInput{
		TenantID: tenant, WorkspaceID: ws, EventType: scheduleHistoryEventRhythmSaved, Message: "已保存调度设置",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+tenant+"/workspace/"+ws+"/queue-schedule/", nil)
	req.Header.Set("X-Auth-User-Id", "internal")
	rr := httptest.NewRecorder()
	handleWorkspaceQueueScheduleRoutes(rr, req, tenant, ws)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET snapshot status=%d body=%s", rr.Code, rr.Body.String())
	}
	var snap map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &snap); err != nil {
		t.Fatal(err)
	}
	recent, _ := snap["recent_history"].([]interface{})
	if len(recent) != 1 {
		t.Fatalf("recent_history=%v", snap["recent_history"])
	}

	req = httptest.NewRequest(http.MethodGet, "/api/tenant/"+tenant+"/workspace/"+ws+"/queue-schedule/history/?limit=20", nil)
	req.Header.Set("X-Auth-User-Id", "internal")
	rr = httptest.NewRecorder()
	handleWorkspaceQueueScheduleRoutes(rr, req, tenant, ws)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET history status=%d body=%s", rr.Code, rr.Body.String())
	}
	var hist map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &hist); err != nil {
		t.Fatal(err)
	}
	items, _ := hist["items"].([]interface{})
	if len(items) != 1 || hist["has_more"] != false {
		t.Fatalf("history=%v", hist)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/tenant/"+tenant+"/workspace/"+ws+"/queue-schedule/history/", nil)
	req.Header.Set("X-Auth-User-Id", "u-no-access")
	rr = httptest.NewRecorder()
	handleWorkspaceQueueScheduleRoutes(rr, req, tenant, ws)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}
