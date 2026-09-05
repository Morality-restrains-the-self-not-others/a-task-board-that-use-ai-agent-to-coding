package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublishTaskSSEStopSuccessDoesNotMarkServerStarted(t *testing.T) {
	setupCommentContainerBindingTest(t, "taskSrvStop")

	body := `{"comment_id":"cStop","execution_mode":"independent"}`
	req := httptest.NewRequest(http.MethodPost,
		"/api/tenant/t1/task/taskSrvStop/comment-container-bindings/",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCommentContainerBindingsRoutes(rec, req, "t1", "taskSrvStop", "", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	if err := publishTaskSSE(context.Background(), "taskSrvStop", "cStop", map[string]interface{}{
		"status":     "success",
		"message":    "停止虚拟机成功",
		"progress":   100,
		"event_name": "server_status_update",
	}); err != nil {
		t.Logf("publishTaskSSE err (ok if kafka down): %v", err)
	}

	rows, err := listCommentContainerBindingLogs("t1", "taskSrvStop")
	if err != nil {
		t.Fatal(err)
	}
	foundStop := false
	for i := range rows {
		if rows[i].CommentID != "cStop" {
			continue
		}
		if rows[i].Stage == ccbStageServerStarted {
			t.Fatalf("stop-vm success must not use server_started: %+v", rows[i])
		}
		if rows[i].Stage == ccbStageServerScheduling && strings.Contains(rows[i].Message, "停止虚拟机成功") {
			foundStop = true
		}
	}
	if !foundStop {
		t.Fatalf("missing stop scheduling log; logs=%+v", rows)
	}
}

func TestIsCloudServerStopSchedulingMessage(t *testing.T) {
	if !isCloudServerStopSchedulingMessage("正在准备停止aliyun服务器...") {
		t.Fatal("prepare-stop copy")
	}
	if !isCloudServerStopSchedulingMessage("正在调用aliyunAPI停止服务器...") {
		t.Fatal("calling-stop copy")
	}
	if !isCloudServerStopSchedulingMessage("正在调用aliyunAPI停止服务器...（触发：容器指令空闲超时回收）") {
		t.Fatal("annotated calling-stop copy")
	}
	if isCloudServerStopSchedulingMessage("正在准备启动aliyun服务器...") {
		t.Fatal("start copy must not match")
	}
	if isCloudServerStopSchedulingMessage("正在停止 Mock 实例...") {
		// ok
	} else {
		t.Fatal("mock stop copy")
	}
	if !isCloudServerStopSchedulingMessage("停止虚拟机成功") {
		t.Fatal("stop success copy")
	}
}
