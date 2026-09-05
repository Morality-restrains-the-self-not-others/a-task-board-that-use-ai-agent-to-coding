package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260815-015: 模板仍为 mock 时，bootstrap 的 ensureCommentCloudServerConfig
// 必须硬失败（而不是前置 soft-fail 打出误导日志后被忽略）。
func TestBootstrapStartVmTokensFailsOnMockPlatform(t *testing.T) {
	setupCloudTestDB(t)
	// platformType="mock" → 任务级 CSC 落 mock，评论级克隆后 commentCSCCloudMetaIllegal 命中 → 硬失败
	credSrv := mockCredentialTokenServer(t, "should-not-be-reached")
	prev := cfg.CredentialServiceURL
	cfg.CredentialServiceURL = credSrv.URL
	t.Cleanup(func() { cfg.CredentialServiceURL = prev })

	_, err := bootstrapStartVmTokens(
		context.Background(),
		"c1", "w1", "task-mock-tpl", "cmt_mock", "auth-mock", "mock", "cn-hangzhou", "cn-hangzhou-h",
	)
	if err == nil {
		t.Fatal("expected hard failure on mock platform template")
	}
	if !strings.Contains(err.Error(), "ensure comment cloud_server_config") {
		t.Fatalf("err=%v", err)
	}
}

// OPT-20260815-015: 无模板/无授权时，start-vm-auto 在 RunInstances（executeStartVmAutoNative）
// 之前返回错误；凭证解析失败直接 400，不进入 finalize/bootstrap。
func TestHandleStartVmAutoNativeNoAuthFailsBeforeRunInstances(t *testing.T) {
	setupCloudTestDB(t)
	body := map[string]interface{}{
		"task_id":                     "task_noauth",
		"comment_id":                  "cmt_noauth",
		"container_image_id":          "img-noauth",
		"auto_create_security_group":  true,
		"auto_create_vswitch":         true,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/internal/cloud/compute/start-vm-auto/", strings.NewReader(string(raw)))
	req.Header.Set("X-User-Id", "u1")
	rec := httptest.NewRecorder()

	handleStartVmAutoNative(rec, req, "c1", "w1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "未配置云平台授权") {
		t.Fatalf("body=%s", rec.Body.String())
	}
	// 凭证解析失败 → 未写入任何 CSC（finalize/bootstrap 未执行，RunInstances 未调用）
	if row, err := loadCloudServerConfigForComment("c1", "w1", "task_noauth", "cmt_noauth"); err == nil && row != nil {
		t.Fatalf("comment CSC must not exist before auth resolution: %+v", row)
	}
}
