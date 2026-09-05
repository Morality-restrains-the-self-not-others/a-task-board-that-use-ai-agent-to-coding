package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestUIHomePage_QueueStatusBarShowsBulkProgressDetail 回归：
// queue-status-bar 曾只显示「正在执行: 精准编译重启」而无进度细节，且中断按钮
// 未调用 /api/precise-restart/cancel，导致 UI 看起来卡住且无法真正中断。
func TestUIHomePage_QueueStatusBarShowsBulkProgressDetail(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})

	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, snippet := range []string{
		`id="queue-status-bar"`,
		`applyProgress:`,
		`adoptExternalOp:`,
		`syncBulkQueueFromStatus`,
		`lastStatusEnvelope`,
		`resumeActiveBulkProgress(lastStatusEnvelope)`,
		`hydrateExecQueueFromStatus`,
		`restorePending:`,
		`self.processNext()`,
		`await this.syncPendingToServer()`,
		`adoptInProgressBulkOp`,
		`/api/exec-queue`,
		`execution_queue`,
		`queue-progress-detail`,
		`buildAllBtn.dataset.confirming`,
		`delete buildAllBtn.dataset.confirming`,
		`/api/precise-restart/cancel`,
		`else if (op === 'precise-restart') url = '/api/precise-restart/cancel'`,
		`cancelOpFromQueueType`,
		`disconnectPreciseRestartSSE()`,
		`waitingProgressHint:`,
		`启动/健康检查中…`,
		`关闭中…`,
		`编译中…`,
		`重启/健康检查中…`,
		`初始化中…`,
		`清空中…`,
		`执行中…`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status page missing snippet %q", snippet)
		}
	}
	if strings.Contains(body, `编译/健康检查中…`) {
		t.Fatal("status page still uses generic waiting copy 编译/健康检查中…")
	}
}
