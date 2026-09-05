package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// authAsUser 构造带鉴权头的请求（X-User-Id 为网关 forward-auth 注入头）。
func authAsUser(method, path, userID string, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-User-Id", userID)
	return req
}

// overrideExportFetchers 覆盖远端 personal-data fetch（测试隔离内部 HTTP 依赖）。
func overrideExportFetchers(t *testing.T, bill, tenant, cloud func(context.Context, string) (interface{}, error)) {
	t.Helper()
	fetchPersonalDataBillFn = bill
	fetchPersonalDataTenantFn = tenant
	fetchPersonalDataCloudFn = cloud
	t.Cleanup(func() {
		fetchPersonalDataBillFn = fetchBillPersonalData
		fetchPersonalDataTenantFn = fetchTenantPersonalData
		fetchPersonalDataCloudFn = fetchCloudPersonalData
	})
}

func okRemoteSection(name string) func(context.Context, string) (interface{}, error) {
	return func(_ context.Context, _ string) (interface{}, error) {
		return map[string]interface{}{"ok": true, "source": name}, nil
	}
}

func TestPersonalDataExportLifecycle(t *testing.T) {
	setupAuthTestDB(t)
	overrideExportFetchers(t, okRemoteSection("bill"), okRemoteSection("tenant"), okRemoteSection("cloud"))
	userID := insertTestUser(t, "export-life@test.com", "secret123")

	// request → ready + 完整 sections
	req := authAsUser(http.MethodPost, "/api/accounts/users/me/personal-data-export/request/", userID, "{}")
	rec := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("request: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("request body: %v", err)
	}
	if resp["status"] != "ready" {
		t.Fatalf("expected status ready, got %v", resp["status"])
	}
	exportID := fmt.Sprintf("%v", resp["export_id"])
	if exportID == "" || exportID == "<nil>" {
		t.Fatalf("missing export_id: %v", resp)
	}

	// status → ready + 有效期
	rec2 := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec2, authAsUser(http.MethodGet, "/api/accounts/users/me/personal-data-export/status/", userID, ""))
	if rec2.Code != http.StatusOK {
		t.Fatalf("status: expected 200, got %d", rec2.Code)
	}
	var st map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &st)
	if st["status"] != "ready" || st["export_id"] != exportID {
		t.Fatalf("unexpected status payload: %v", st)
	}

	// download → 200 + Content-Disposition + 内容含本地与远端 section
	rec3 := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec3, authAsUser(http.MethodGet, "/api/accounts/users/me/personal-data-export/download/", userID, ""))
	if rec3.Code != http.StatusOK {
		t.Fatalf("download: expected 200, got %d", rec3.Code)
	}
	if ct := rec3.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type=%s", ct)
	}
	if cd := rec3.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Fatalf("Content-Disposition=%s", cd)
	}
	if cc := rec3.Header().Get("Cache-Control"); cc != "private, no-store" {
		t.Fatalf("Cache-Control=%s", cc)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec3.Body.Bytes(), &payload); err != nil {
		t.Fatalf("download content: %v", err)
	}
	sections, _ := payload["sections"].(map[string]interface{})
	for _, key := range []string{"account", "profile", "login_methods", "access_tokens",
		"wechat_identity", "login_history", "kyc", "deletion_history", "billing", "tenant_memberships", "cloud_servers"} {
		if _, ok := sections[key]; !ok {
			t.Fatalf("missing section %q in %v", key, payload)
		}
	}

	// 凭证不泄漏：内容不得包含 password 字段/hash
	raw := rec3.Body.String()
	for _, forbidden := range []string{"password_hash", "activation_token", "token_hash", "customtoken"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("export leaks credential field %q", forbidden)
		}
	}
	// 登录方式 identifier 应包含测试邮箱（个人信息可导出）
	if !strings.Contains(raw, "export-life@test.com") {
		t.Fatalf("export missing login identifier")
	}
}

func TestPersonalDataExportUpstreamTimeoutDegradesToPartial(t *testing.T) {
	setupAuthTestDB(t)
	// 覆盖超时阈值为极小值：慢上游应触发上下文超时 → 按 section 降级为 partial，
	// 而不是让整次导出请求挂起直至外层超时。
	old := personalDataExportUpstreamTimeout
	personalDataExportUpstreamTimeout = 30 * time.Millisecond
	t.Cleanup(func() { personalDataExportUpstreamTimeout = old })

	// 慢函数直接走真实 internalPersonalDataGet 路径验证超时生效：
	// 用慢 httptest 服务器替换 bill fetch，观察超时后 bill 进入 sections_unavailable。
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer slowServer.Close()
	overrideExportFetchers(t,
		func(_ context.Context, _ string) (interface{}, error) {
			return internalPersonalDataGet(context.Background(), slowServer.URL, "X-TaskBill-Internal-Secret", "secret")
		},
		okRemoteSection("tenant"),
		okRemoteSection("cloud"),
	)

	userID := insertTestUser(t, "export-timeout@test.com", "secret123")
	rec := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec, authAsUser(http.MethodPost, "/api/accounts/users/me/personal-data-export/request/", userID, "{}"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 (timeout must degrade to partial), got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "partial" {
		t.Fatalf("expected status partial after upstream timeout, got %v", resp["status"])
	}
	unavailable, _ := resp["sections_unavailable"].([]interface{})
	if len(unavailable) != 1 || unavailable[0] != "billing" {
		t.Fatalf("expected billing unavailable, got %v", unavailable)
	}
}

func TestPersonalDataExportPartialOnRemoteFailure(t *testing.T) {
	setupAuthTestDB(t)
	fail := func(_ context.Context, _ string) (interface{}, error) {
		return nil, fmt.Errorf("simulated outage")
	}
	overrideExportFetchers(t, fail, okRemoteSection("tenant"), okRemoteSection("cloud"))
	userID := insertTestUser(t, "export-partial@test.com", "secret123")

	rec := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec, authAsUser(http.MethodPost, "/api/accounts/users/me/personal-data-export/request/", userID, "{}"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 (partial must not fail), got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["status"] != "partial" {
		t.Fatalf("expected status partial, got %v", resp["status"])
	}
	unavailable, _ := resp["sections_unavailable"].([]interface{})
	if len(unavailable) != 1 || unavailable[0] != "billing" {
		t.Fatalf("expected billing unavailable, got %v", unavailable)
	}

	rec3 := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec3, authAsUser(http.MethodGet, "/api/accounts/users/me/personal-data-export/download/", userID, ""))
	if rec3.Code != http.StatusOK {
		t.Fatalf("partial download: expected 200, got %d", rec3.Code)
	}
}

func TestPersonalDataExportExpiry(t *testing.T) {
	setupAuthTestDB(t)
	overrideExportFetchers(t, okRemoteSection("bill"), okRemoteSection("tenant"), okRemoteSection("cloud"))
	userID := insertTestUser(t, "export-exp@test.com", "secret123")

	req := authAsUser(http.MethodPost, "/api/accounts/users/me/personal-data-export/request/", userID, "{}")
	rec := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("request: %d", rec.Code)
	}

	// 回拨 expires_at 至过去 → download 410 + status expired
	_, _ = db.Exec(`UPDATE auth_personal_data_export SET expires_at = ? WHERE user_id = ?`,
		time.Now().UTC().Add(-time.Minute), userID)

	recD := httptest.NewRecorder()
	handlePersonalDataExportRouter(recD, authAsUser(http.MethodGet, "/api/accounts/users/me/personal-data-export/download/", userID, ""))
	if recD.Code != http.StatusGone {
		t.Fatalf("expired download: expected 410, got %d", recD.Code)
	}

	recS := httptest.NewRecorder()
	handlePersonalDataExportRouter(recS, authAsUser(http.MethodGet, "/api/accounts/users/me/personal-data-export/status/", userID, ""))
	var st map[string]interface{}
	_ = json.Unmarshal(recS.Body.Bytes(), &st)
	if st["status"] != "expired" {
		t.Fatalf("expected status expired, got %v", st["status"])
	}

	// 过期后重新生成 → 恢复 ready（旧行被覆盖）
	recR := httptest.NewRecorder()
	handlePersonalDataExportRouter(recR, authAsUser(http.MethodPost, "/api/accounts/users/me/personal-data-export/request/", userID, "{}"))
	if recR.Code != http.StatusCreated {
		t.Fatalf("regenerate: %d", recR.Code)
	}
	recD2 := httptest.NewRecorder()
	handlePersonalDataExportRouter(recD2, authAsUser(http.MethodGet, "/api/accounts/users/me/personal-data-export/download/", userID, ""))
	if recD2.Code != http.StatusOK {
		t.Fatalf("download after regenerate: expected 200, got %d", recD2.Code)
	}
	// 每用户仅保留一行
	var cnt int
	_ = db.QueryRow(`SELECT COUNT(*) FROM auth_personal_data_export WHERE user_id = ?`, userID).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("expected 1 row per user, got %d", cnt)
	}
}

func TestPersonalDataExportAuthRequired(t *testing.T) {
	setupAuthTestDB(t)
	// 无鉴权 → 401
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/me/personal-data-export/status/", nil)
	rec := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/accounts/users/me/personal-data-export/download/", nil)
	rec2 := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("download without auth: expected 401, got %d", rec2.Code)
	}
}

func TestPersonalDataExportDownloadWithoutRequest(t *testing.T) {
	setupAuthTestDB(t)
	userID := insertTestUser(t, "export-none@test.com", "secret123")
	rec := httptest.NewRecorder()
	handlePersonalDataExportRouter(rec, authAsUser(http.MethodGet, "/api/accounts/users/me/personal-data-export/download/", userID, ""))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	recS := httptest.NewRecorder()
	handlePersonalDataExportRouter(recS, authAsUser(http.MethodGet, "/api/accounts/users/me/personal-data-export/status/", userID, ""))
	var st map[string]interface{}
	_ = json.Unmarshal(recS.Body.Bytes(), &st)
	if st["status"] != "none" {
		t.Fatalf("expected status none, got %v", st["status"])
	}
}

func TestPersonalDataExportRoutePatterns(t *testing.T) {
	mux := http.NewServeMux()
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("mountRoutes panicked: %v", rec)
		}
	}()
	mountRoutes(mux)

	getReq := httptest.NewRequest(http.MethodGet, "/api/accounts/users/me/personal-data-export/status", nil)
	_, getPattern := mux.Handler(getReq)
	if getPattern != "GET /api/accounts/users/me/personal-data-export/" {
		t.Fatalf("status matched %q, want export GET subtree", getPattern)
	}
	postReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/me/personal-data-export/request", nil)
	_, postPattern := mux.Handler(postReq)
	if postPattern != "POST /api/accounts/users/me/personal-data-export/" {
		t.Fatalf("request matched %q, want export POST subtree", postPattern)
	}
}
