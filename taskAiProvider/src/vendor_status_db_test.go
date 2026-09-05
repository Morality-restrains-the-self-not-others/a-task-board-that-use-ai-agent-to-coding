package main

// TestSSO 前缀命名 — taskAiProvider/.githooks/pre-commit 门禁以 `go test -run 'SSO'`
// 必跑本文件（厂商门户资格判断为 SSO vendor_bridge 链路的前置展示条件）。
// 依赖真实 MySQL（dbload.OpenTestMySQL 每测试独立建库），MySQL 不可用时 t.Skip。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAiProvider/infrastructure"
)

// seedVendorDocKeys 写入测试证照并返回申请 JSON body（含手机号）。
func seedVendorDocKeys(t *testing.T, app *App, uid int64, company, contact string) string {
	t.Helper()
	idKey, _, err := infrastructure.SaveVendorDocument(app.Cfg.VendorDocsDir, uid, infrastructure.VendorDocKindIDCard, "id.jpg", bytes.NewReader([]byte("idimg")), 5)
	if err != nil {
		t.Fatalf("seed id card: %v", err)
	}
	licKey, _, err := infrastructure.SaveVendorDocument(app.Cfg.VendorDocsDir, uid, infrastructure.VendorDocKindBusinessLicense, "lic.pdf", bytes.NewReader([]byte("%PDF-1")), 6)
	if err != nil {
		t.Fatalf("seed license: %v", err)
	}
	body, _ := json.Marshal(map[string]string{
		"company_name":              company,
		"contact_name":              contact,
		"id_card_file_key":          idKey,
		"business_license_file_key": licKey,
		"contact_phone":             "+8613800138000",
	})
	return string(body)
}

// seedVendor 直接插入 ai_provider_vendor 行（运营建档路径），返回厂商 id。
// saasID 传 0 表示 NULL（仅 email 建档、账号未绑定）。
func seedVendor(t *testing.T, app *App, saasID int64, email string, active bool) int64 {
	t.Helper()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	activeInt := 0
	if active {
		activeInt = 1
	}
	var saasVal any
	if saasID != 0 {
		saasVal = saasID
	}
	res, err := app.DB.SQL.Exec(
		`INSERT INTO ai_provider_vendor (saas_user_id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		saasVal, email, "test-hash", "Test Vendor Co", "tester", activeInt, now, now)
	if err != nil {
		t.Fatalf("seed vendor: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func vendorStatusRequest(app *App, mux *http.ServeMux, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/ai-provider/vendor-status/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeVendorStatus(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	return body
}

func TestSSOVendorStatusUnauthenticated(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rec := vendorStatusRequest(app, mux, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without X-User-Id, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSSOVendorStatusBoundBySaasUserID(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(10001), "bound@example.com", true)

	rec := vendorStatusRequest(app, mux, map[string]string{"X-User-Id": "10001"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeVendorStatus(t, rec)
	if body["is_vendor"] != true {
		t.Fatalf("expected is_vendor=true for bound vendor, got %v", body["is_vendor"])
	}
	vendor, _ := body["vendor"].(map[string]any)
	if vendor == nil || vendor["email"] != "bound@example.com" {
		t.Fatalf("expected vendor payload with email, got %v", body["vendor"])
	}
	if vendor["saas_user_id"] != "10001" {
		t.Fatalf("expected saas_user_id=10001 in payload, got %v", vendor["saas_user_id"])
	}
}

func TestSSOVendorStatusInactiveVendor(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(10002), "disabled@example.com", false)

	rec := vendorStatusRequest(app, mux, map[string]string{"X-User-Id": "10002"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeVendorStatus(t, rec)
	if body["is_vendor"] != false {
		t.Fatalf("expected is_vendor=false for inactive vendor, got %v", body["is_vendor"])
	}
}

func TestSSOVendorStatusEmailMatchUnbound(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	// 运营建档：仅有 email，账号尚未绑定（saas_user_id NULL）——按钮应显示，
	// 首次点击 SSO 桥完成绑定，避免死锁。
	seedVendor(t, app, 0, "ops-created@example.com", true)

	rec := vendorStatusRequest(app, mux, map[string]string{
		"X-User-Id": "10003", "X-User-Email": "OPS-CREATED@example.com",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeVendorStatus(t, rec)
	if body["is_vendor"] != true {
		t.Fatalf("expected is_vendor=true for email-matched unbound vendor, got %v", body["is_vendor"])
	}
}

func TestSSOVendorStatusNoVendor(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(20001), "other@example.com", true)

	rec := vendorStatusRequest(app, mux, map[string]string{
		"X-User-Id": "20002", "X-User-Email": "unrelated@example.com",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeVendorStatus(t, rec)
	if body["is_vendor"] != false {
		t.Fatalf("expected is_vendor=false for unrelated user, got %v", body["is_vendor"])
	}
}

func TestSSOVendorStatusSyntheticEmailNoMatch(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	// 微信扫码用户：X-User-Email 为确定性合成邮箱（sso-<id>@sso.invalid，
	// RFC 2606 .invalid 保留域，永不可投递），不可能匹配任何真实厂商记录 → 不显示按钮。
	seedVendor(t, app, int64(30001), "real-vendor@example.com", true)

	rec := vendorStatusRequest(app, mux, map[string]string{
		"X-User-Id": "30002", "X-User-Email": "sso-30002@sso.invalid",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeVendorStatus(t, rec)
	if body["is_vendor"] != false {
		t.Fatalf("expected is_vendor=false for synthetic email (no real vendor), got %v", body["is_vendor"])
	}
}

func TestSSOVendorStatusBoundBeatsEmail(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	// saas_user_id 绑定的厂商优先于 email 匹配（另一厂商档案同邮箱不干扰）
	seedVendor(t, app, int64(40001), "shared@example.com", true)
	seedVendor(t, app, int64(40002), "shared@example.com", false)

	rec := vendorStatusRequest(app, mux, map[string]string{
		"X-User-Id": "40001", "X-User-Email": "shared@example.com",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := decodeVendorStatus(t, rec)
	if body["is_vendor"] != true {
		t.Fatalf("expected is_vendor=true via bound vendor, got %v", body["is_vendor"])
	}
	vendor, _ := body["vendor"].(map[string]any)
	if vendor == nil || vendor["saas_user_id"] != "40001" {
		t.Fatalf("expected bound vendor 40001 payload, got %v", body["vendor"])
	}
}

// ── OPT-20260806-065 审核流：申请提交 / 运营审核 / 四态 / 自动建号移除 ──

func vendorApplicationRequest(app *App, mux *http.ServeMux, headers map[string]string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/ai-provider/vendor-application/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestSSOVendorApplicationSubmitCreatesPending(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	rec := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Id": "50001", "X-User-Email": "applicant@example.com"},
		seedVendorDocKeys(t, app, 50001, "New Vendor Co", "张三"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeVendorStatus(t, rec)
	if body["status"] != "pending" {
		t.Fatalf("expected status=pending, got %v", body["status"])
	}
	// 落库 is_active=0 且 review_note 空
	var active int
	var note string
	var phone string
	if err := app.DB.SQL.QueryRow(`SELECT is_active, COALESCE(review_note,''), COALESCE(contact_phone,'') FROM ai_provider_vendor WHERE saas_user_id=50001`).Scan(&active, &note, &phone); err != nil {
		t.Fatalf("query vendor: %v", err)
	}
	if active != 0 || note != "" {
		t.Fatalf("expected pending row (active=0, note=''), got active=%d note=%q", active, note)
	}
	if phone != "+8613800138000" {
		t.Fatalf("expected contact_phone persisted, got %q", phone)
	}
}

func TestSSOVendorApplicationDuplicatePendingConflict(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(50002), "pending2@example.com", false) // 待审核（无 note）

	rec := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Id": "50002", "X-User-Email": "pending2@example.com"},
		seedVendorDocKeys(t, app, 50002, "X", "Y"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for pending application, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := decodeVendorStatus(t, rec); body["detail"] != "申请审核中，请耐心等待" {
		t.Fatalf("unexpected detail: %v", body["detail"])
	}
}

func TestSSOVendorApplicationAlreadyQualifiedConflict(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(50003), "qualified3@example.com", true)

	rec := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Id": "50003", "X-User-Email": "qualified3@example.com"},
		seedVendorDocKeys(t, app, 50003, "X", "Y"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 for qualified user, got %d body=%s", rec.Code, rec.Body.String())
	}
	if body := decodeVendorStatus(t, rec); body["detail"] != "已是厂商门户成员" {
		t.Fatalf("unexpected detail: %v", body["detail"])
	}
}

func TestSSOVendorApplicationRejectedResubmit(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(50004), "rejected4@example.com", false)
	// 标记为已驳回
	if _, err := app.DB.SQL.Exec(`UPDATE ai_provider_vendor SET review_note='资料不全', reviewed_at=NOW(), reviewed_by=1 WHERE saas_user_id=50004`); err != nil {
		t.Fatalf("mark rejected: %v", err)
	}

	rec := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Id": "50004", "X-User-Email": "rejected4@example.com"},
		seedVendorDocKeys(t, app, 50004, "Resubmit Co", "李四"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 resubmit, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeVendorStatus(t, rec)
	if body["status"] != "pending" {
		t.Fatalf("expected status=pending after resubmit, got %v", body["status"])
	}
	var note string
	if err := app.DB.SQL.QueryRow(`SELECT COALESCE(review_note,'') FROM ai_provider_vendor WHERE saas_user_id=50004`).Scan(&note); err != nil || note != "" {
		t.Fatalf("expected review_note cleared, got %q err=%v", note, err)
	}
}

func TestSSOVendorApplicationRequiresRealEmail(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// 合成邮箱（微信用户未绑定邮箱）→ 400
	rec := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Id": "50005", "X-User-Email": "sso-50005@sso.invalid"},
		`{"company_name":"X","contact_name":"Y"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for synthetic email, got %d body=%s", rec.Code, rec.Body.String())
	}
	// 无 X-User-Id → 401
	rec2 := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Email": "a@b.com"},
		`{"company_name":"X","contact_name":"Y"}`)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without X-User-Id, got %d", rec2.Code)
	}
	// 缺公司名 → 400
	rec3 := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Id": "50006", "X-User-Email": "c@d.com"},
		`{"company_name":"  ","contact_name":"Y"}`)
	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing company_name, got %d", rec3.Code)
	}
}

func TestSSOVendorApplicationRequiresDocuments(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	rec := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Id": "50007", "X-User-Email": "docs@example.com"},
		`{"company_name":"Co","contact_name":"人","contact_phone":"+8613800138000"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 missing docs, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSSOVendorApplicationRequiresPhoneVerified(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	prev := vendorContactVerifiedFn
	vendorContactVerifiedFn = func(_ *App, _ context.Context, _ string) (bool, error) {
		return false, nil
	}
	t.Cleanup(func() { vendorContactVerifiedFn = prev })
	rec := vendorApplicationRequest(app, mux,
		map[string]string{"X-User-Id": "50008", "X-User-Email": "phone@example.com"},
		seedVendorDocKeys(t, app, 50008, "Co", "人"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 unverified phone, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSSOAdminVendorReviewApprove(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	vendorID := seedVendor(t, app, int64(60001), "review@example.com", false)
	seedStaff(t, app, 77701)

	staffTok, _ := infrastructure.IssueToken(app.Cfg.SecretKey, "77701", "staff", 3600)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/ai-provider/admin-vendors/%d/", vendorID),
		strings.NewReader(`{"action":"approve","note":"资料齐全"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+staffTok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 approve, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeVendorStatus(t, rec)
	if body["status"] != "qualified" {
		t.Fatalf("expected qualified after approve, got %v", body["status"])
	}
	var active int
	if err := app.DB.SQL.QueryRow(`SELECT is_active FROM ai_provider_vendor WHERE id=?`, vendorID).Scan(&active); err != nil || active != 1 {
		t.Fatalf("expected is_active=1, got %d err=%v", active, err)
	}
}

func TestSSOAdminVendorReviewReject(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	vendorID := seedVendor(t, app, int64(60002), "reject@example.com", false)
	seedStaff(t, app, 77702)

	staffTok, _ := infrastructure.IssueToken(app.Cfg.SecretKey, "77702", "staff", 3600)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/ai-provider/admin-vendors/%d/", vendorID),
		strings.NewReader(`{"action":"reject","note":"资料不全，请补充营业执照"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+staffTok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 reject, got %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeVendorStatus(t, rec)
	if body["status"] != "rejected" {
		t.Fatalf("expected rejected, got %v", body["status"])
	}
	vendor, _ := body["vendor"].(map[string]any)
	if vendor == nil || vendor["review_note"] != "资料不全，请补充营业执照" {
		t.Fatalf("expected review_note in payload, got %v", body["vendor"])
	}
}

func TestSSOAdminVendorReviewRequiresStaff(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	vendorID := seedVendor(t, app, int64(60003), "noperm@example.com", false)

	// 无 token → 401
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/ai-provider/admin-vendors/%d/", vendorID),
		strings.NewReader(`{"action":"approve"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without staff token, got %d", rec.Code)
	}
	// 非 staff（vendor token）→ 401
	vendorTok, _ := infrastructure.IssueToken(app.Cfg.SecretKey, "60003", "vendor", 3600)
	req2 := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/ai-provider/admin-vendors/%d/", vendorID),
		strings.NewReader(`{"action":"approve"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+vendorTok)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for vendor token, got %d", rec2.Code)
	}
}

// TestSSOAdminVendorReviewIdempotentSameValue OPT-20260807-014 回归：
// 同秒内以相同 action/note 重复审核，reviewed_at/updated_at（DATETIME 秒精度）
// 截断后行值不变 → MySQL RowsAffected=0，原实现误判 ErrNoRows → 404「厂商不存在」。
// 修复后 UPDATE 完成即按 id 复查存在性，重复审核幂等返回 200（运营二次修正允许覆盖）。
// 固定审核时钟保证两次写入相同时间戳，确定性复现 RowsAffected=0 路径。
func TestSSOAdminVendorReviewIdempotentSameValue(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	vendorID := seedVendor(t, app, int64(60004), "idem@example.com", false)
	seedStaff(t, app, 77704)

	orig := infrastructure.ReviewNowFunc
	infrastructure.ReviewNowFunc = func() string { return "2026-08-08 00:00:00.000000" }
	defer func() { infrastructure.ReviewNowFunc = orig }()

	staffTok, _ := infrastructure.IssueToken(app.Cfg.SecretKey, "77704", "staff", 3600)
	review := func() (int, string) {
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/ai-provider/admin-vendors/%d/", vendorID),
			strings.NewReader(`{"action":"approve","note":"资料齐全"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+staffTok)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	if code, body := review(); code != http.StatusOK {
		t.Fatalf("first review: expected 200, got %d body=%s", code, body)
	}
	if code, body := review(); code != http.StatusOK {
		t.Fatalf("same-value repeat review: expected 200 (idempotent overwrite), got %d body=%s", code, body)
	}
}

// TestSSOAdminVendorReviewMissingVendor404 OPT-20260807-014 契约保持：
// 真不存在的厂商仍须 404（UPDATE 后 GetVendorByID 自然返回 sql.ErrNoRows）。
func TestSSOAdminVendorReviewMissingVendor404(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedStaff(t, app, 77705)

	staffTok, _ := infrastructure.IssueToken(app.Cfg.SecretKey, "77705", "staff", 3600)
	req := httptest.NewRequest(http.MethodPatch, "/api/ai-provider/admin-vendors/999999/",
		strings.NewReader(`{"action":"approve","note":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+staffTok)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing vendor, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSSOVendorStatusFourStates(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(61001), "p@example.com", false) // pending
	seedVendor(t, app, int64(61002), "r@example.com", false) // 待驳回
	seedVendor(t, app, int64(61003), "q@example.com", true)  // qualified
	if _, err := app.DB.SQL.Exec(`UPDATE ai_provider_vendor SET review_note='no' WHERE saas_user_id=61002`); err != nil {
		t.Fatalf("mark rejected: %v", err)
	}

	// pending
	rec := vendorStatusRequest(app, mux, map[string]string{"X-User-Id": "61001"})
	body := decodeVendorStatus(t, rec)
	if body["status"] != "pending" || body["is_vendor"] != false {
		t.Fatalf("expected pending/is_vendor=false, got %v", body)
	}
	// rejected
	rec = vendorStatusRequest(app, mux, map[string]string{"X-User-Id": "61002"})
	body = decodeVendorStatus(t, rec)
	if body["status"] != "rejected" {
		t.Fatalf("expected rejected, got %v", body)
	}
	if vendor, _ := body["vendor"].(map[string]any); vendor == nil || vendor["review_note"] != "no" {
		t.Fatalf("expected review_note, got %v", body["vendor"])
	}
	// qualified + has_email
	rec = vendorStatusRequest(app, mux, map[string]string{"X-User-Id": "61003", "X-User-Email": "q@example.com"})
	body = decodeVendorStatus(t, rec)
	if body["status"] != "qualified" || body["is_vendor"] != true || body["has_email"] != true {
		t.Fatalf("expected qualified/is_vendor=true/has_email=true, got %v", body)
	}
	// none + 无邮箱（合成）→ has_email=false
	rec = vendorStatusRequest(app, mux, map[string]string{"X-User-Id": "61004", "X-User-Email": "sso-61004@sso.invalid"})
	body = decodeVendorStatus(t, rec)
	if body["status"] != "none" || body["has_email"] != false {
		t.Fatalf("expected none/has_email=false, got %v", body)
	}
	// none + 空邮箱头 → has_email=false（不能把空串当成已绑邮箱）
	rec = vendorStatusRequest(app, mux, map[string]string{"X-User-Id": "61005"})
	body = decodeVendorStatus(t, rec)
	if body["status"] != "none" || body["has_email"] != false {
		t.Fatalf("expected none/has_email=false for empty email, got %v", body)
	}
}

func TestSSOExchangeNoVendorNoAutoCreate(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	// 自动建号移除回归：未申请用户构造 bridge → 拒绝且不落库
	bridge := signSSOBridge(t, app, taskAuthStyleClaims(app, "70001", "vendor_bridge",
		map[string]any{"email": "not-applied@example.com"}))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/",
		strings.NewReader(`{"bridge":"`+bridge+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unapplied user, got %d body=%s", rec.Code, rec.Body.String())
	}
	var n int
	if err := app.DB.SQL.QueryRow(`SELECT COUNT(1) FROM ai_provider_vendor WHERE saas_user_id=70001`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("expected no auto-created vendor (n=%d err=%v)", n, err)
	}
}

func TestSSOExchangePendingVendorRejected(t *testing.T) {
	app := testApp(t)
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)
	seedVendor(t, app, int64(70002), "pending-x@example.com", false)

	bridge := signSSOBridge(t, app, taskAuthStyleClaims(app, "70002", "vendor_bridge",
		map[string]any{"email": "pending-x@example.com"}))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sso/exchange/",
		strings.NewReader(`{"bridge":"`+bridge+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for pending vendor, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if detail, _ := out["detail"].(string); !strings.Contains(detail, "审核中") {
		t.Fatalf("expected 审核中 detail, got %q", detail)
	}
}

// seedStaff 插入运营 staff 记录（requireStaff 校验存在且活跃）。
func seedStaff(t *testing.T, app *App, staffID int64) {
	t.Helper()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if _, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_platformstaff (id, username, password_hash, display_name, is_active, created_at, updated_at) VALUES (?,?,?,?,1,?,?)`,
		staffID, fmt.Sprintf("staff-%d", staffID), "x", "Ops", now, now); err != nil {
		t.Fatalf("seed staff: %v", err)
	}
}
