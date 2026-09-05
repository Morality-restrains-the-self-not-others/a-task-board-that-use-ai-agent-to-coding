package main

// 回归测试（OPT-20260807-013 修复）：运营审核 URL 前缀解析。
// 生产事实：AdminPortal.vue 以 PATCH /api/admin/vendors/{id}/ 调用审核；
// 旧实现先剥 /api/ai-provider/admin-vendors/ 前缀、仅当前缀剥空才回退，
// 导致 /api/admin/vendors/{id}/ 路径 ParseInt 整串失败 → 400 "vendor id 无效"
// （Loki trace 7381532a-6587-4ff6-bae5-21b983518c29, 2026-08-07T13:33:08）。
// 两条前缀路径都必须能完成审核（200），缺一即回归。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"taskAiProvider/infrastructure"
)

func TestAdminVendorReviewPATCHOnBothPrefixes(t *testing.T) {
	app := testApp(t)
	staffID := infrastructure.NextID()
	vendorAID := infrastructure.NextID()
	vendorBID := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_platformstaff
		(id, username, password_hash, display_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,1,?,?)`,
		staffID, "test_staff_vendor_review", "x", "Test Staff", now, now)
	if err != nil {
		t.Fatal(err)
	}
	// 待审核厂商申请（is_active=0、review_note 空）— 与 ApplyVendorApplication 建档形态一致
	for _, v := range []struct {
		id    int64
		email string
	}{
		{vendorAID, "vendor_review_path_a@example.com"},
		{vendorBID, "vendor_review_path_b@example.com"},
	} {
		_, err = app.DB.SQL.Exec(`INSERT INTO ai_provider_vendor
			(id, email, password_hash, company_name, contact_name, is_active, created_at, updated_at)
			VALUES (?,?,?,?,?,0,?,?)`,
			v.id, v.email, "x", "ReviewPathCo", "Contact", now, now)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_vendor WHERE id IN (?,?)`, vendorAID, vendorBID)
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_platformstaff WHERE id=?`, staffID)
	})

	staffTok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(staffID), "staff", 3600)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]string{"action": "approve"})
	paths := []string{
		"/api/admin/vendors/" + infrastructure.IDStr(vendorAID) + "/",             // 生产路径（AdminPortal.vue）
		"/api/ai-provider/admin-vendors/" + infrastructure.IDStr(vendorBID) + "/", // 兼容路径
	}
	for _, p := range paths {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPatch, p, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+staffTok)
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rr, req)
		if rr.Code != 200 {
			t.Fatalf("PATCH %s status %d body %s", p, rr.Code, rr.Body.String())
		}
	}
}

// 非法 id（非数字/缺 id）仍须 400，防止修复把校验一起放掉。
func TestAdminVendorReviewRejectsInvalidID(t *testing.T) {
	app := testApp(t)
	staffID := infrastructure.NextID()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")

	_, err := app.DB.SQL.Exec(`INSERT INTO ai_provider_platformstaff
		(id, username, password_hash, display_name, is_active, created_at, updated_at)
		VALUES (?,?,?,?,1,?,?)`,
		staffID, "test_staff_vendor_review_bad", "x", "Test Staff", now, now)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = app.DB.SQL.Exec(`DELETE FROM ai_provider_platformstaff WHERE id=?`, staffID)
	})

	staffTok, err := infrastructure.IssueToken(app.Cfg.SecretKey, infrastructure.IDStr(staffID), "staff", 3600)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]string{"action": "approve"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/vendors/not-a-number/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+staffTok)
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != 400 {
		t.Fatalf("invalid id: status %d want 400, body %s", rr.Code, rr.Body.String())
	}
}
