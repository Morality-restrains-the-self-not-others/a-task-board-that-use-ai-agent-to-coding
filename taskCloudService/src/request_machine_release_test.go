package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSoleOrEmptyBusy(t *testing.T) {
	self := "task-a"
	if !soleOrEmptyBusy(nil, self) {
		t.Fatal("nil bindings => sole")
	}
	if !soleOrEmptyBusy([]instanceBindingRow{
		{TaskID: "task-a", ServerURL: "http://x"},
	}, self) {
		t.Fatal("only self busy => sole")
	}
	if !soleOrEmptyBusy([]instanceBindingRow{
		{TaskID: "task-b", ServerURL: "", TerminalReleased: 0},
		{TaskID: "task-a", ServerURL: "http://x"},
	}, self) {
		t.Fatal("other idle => sole")
	}
	if soleOrEmptyBusy([]instanceBindingRow{
		{TaskID: "task-b", ServerURL: "http://y"},
		{TaskID: "task-a", ServerURL: "http://x"},
	}, self) {
		t.Fatal("other busy => not sole")
	}
	if !soleOrEmptyBusy([]instanceBindingRow{
		{TaskID: "task-b", ServerURL: "http://y", TerminalReleased: 1},
		{TaskID: "task-a", ServerURL: "http://x"},
	}, self) {
		t.Fatal("other released busy url ignored => sole")
	}
}

func TestBusyBindingCount(t *testing.T) {
	n := busyBindingCount([]instanceBindingRow{
		{ServerURL: "http://a", TerminalReleased: 0},
		{ServerURL: "http://b", TerminalReleased: 1},
		{ServerURL: "", TerminalReleased: 0},
	}, true)
	if n != 1 {
		t.Fatalf("busyCount=%d want 1", n)
	}
}

func TestReleaseMachineForTerminalPersistsStopTriggerLog(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskRelStop")
	if _, err := db.Exec(`UPDATE cloud_server_configs SET instance_id='i-rel-stop' WHERE id='cfg1-cmt'`); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskRelStop/comment-container-bindings/",
		strings.NewReader(`{"comment_id":"cmt-rt","execution_mode":"independent"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskRelStop", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	cfg, err := loadCloudServerConfigByID("cfg1-cmt")
	if err != nil || cfg == nil {
		t.Fatalf("load cfg: %v", err)
	}
	if _, err := releaseMachineForTerminal(context.Background(), cfg, "t1", "ws1", "taskRelStop", "cancelled", "instruction_idle"); err != nil {
		t.Fatal(err)
	}
	rows, err := listCommentContainerBindingLogsIn("ws1", "t1", "taskRelStop")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, l := range rows {
		if strings.Contains(l.Message, "正在调用aliyunAPI停止服务器") &&
			strings.Contains(l.Message, "触发：容器指令空闲超时回收") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing annotated stop log; logs=%+v", rows)
	}
}
