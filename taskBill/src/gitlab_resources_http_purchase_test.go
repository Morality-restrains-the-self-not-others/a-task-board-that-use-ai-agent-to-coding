package main

import (
	"authz"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 产品模型：无用户留存现金。POST /purchase/ 不得扣 billing_account.balance，须引导资源订单。
func TestHandleGitlabResourcesPurchase_DoesNotDebitWallet(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()

	const tenantID int64 = 9300000099
	const startBalance int64 = 99999
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, generateSnowflakeID(), tenantID, startBalance, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_gitlab_region (
			id, name, slug, description, gitlab_api_base, gitlab_web_url,
			is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
		) VALUES (?, ?, ?, 'desc', 'https://api.daydaymoney.com', 'https://example.com', 1, 99, 50, 500, 0, 0, 'test-token', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		generateSnowflakeID(), "腾讯上海一区", "tencent-sh-1", now, now,
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}

	tid := formatID(tenantID)
	body := `{"region":"tencent-sh-1","disk_gb":1,"disk_months":1,"traffic_prepaid_gb":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/billing/gitlab-resources/tenant_id/"+tid+"/purchase/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "10001")
	req.Header.Set(authz.HeaderTenantPerms, tid+":billing:manage")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("wallet purchase must not succeed, body=%s", rec.Body.String())
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("code=%d want 409 body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	errMsg, _ := out["error"].(string)
	if !strings.Contains(errMsg, "资源订单") {
		t.Fatalf("error=%q want 资源订单引导", errMsg)
	}
	if out["code"] != "USE_RESOURCE_ORDER" {
		t.Fatalf("code=%v want USE_RESOURCE_ORDER", out["code"])
	}

	var balance int64
	if err := db.QueryRow(`SELECT balance FROM billing_account WHERE tenant_id = ?`, tenantID).Scan(&balance); err != nil {
		t.Fatalf("query balance: %v", err)
	}
	if balance != startBalance {
		t.Fatalf("balance changed %d → %d; purchase must not debit wallet", startBalance, balance)
	}
	var n int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM billing_tenant_gitlab_resource
		WHERE tenant_id = ? AND (disk_gb > 0 OR traffic_prepaid_gb > 0)`, tenantID).Scan(&n); err != nil {
		t.Fatalf("query resource: %v", err)
	}
	if n != 0 {
		t.Fatalf("quota rows=%d; wallet purchase must not grant GitLab quota", n)
	}
}

func TestRegionResourceView_OmitsSpendableCash(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000100
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 12345, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_gitlab_region (
			id, name, slug, description, gitlab_api_base, gitlab_web_url,
			is_active, sort_order, total_disk_gb, total_traffic_gb,
			allocated_disk_gb, allocated_traffic_gb, admin_private_token, cloud_provider, created_at, updated_at
		) VALUES (?, ?, ?, 'desc', 'https://api.daydaymoney.com', 'https://example.com', 1, 99, 50, 500, 0, 0, 'test-token', 'tencent', ?, ?)
		ON DUPLICATE KEY UPDATE name = VALUES(name)`,
		generateSnowflakeID(), "腾讯上海一区", "tencent-sh-1", now, now,
	); err != nil {
		t.Fatalf("seed region: %v", err)
	}
	view, err := regionResourceView(tenantID, "tencent-sh-1")
	if err != nil {
		t.Fatalf("regionResourceView: %v", err)
	}
	if _, ok := view["balance_points"]; ok {
		t.Fatalf("tenant GitLab view must not include balance_points: %+v", view)
	}
}

func TestHandleResourceQuotas_OmitsSpendableCash(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000101
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, frozen_balance, created_at, updated_at)
		VALUES (?, ?, 888, 100, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+formatID(tenantID)+"/billing/quotas/", nil)
	rec := httptest.NewRecorder()
	handleResourceQuotas(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := body["balance_points"]; ok {
		t.Fatalf("quotas must not include balance_points: %+v", body)
	}
	if _, ok := body["frozen_balance"]; ok {
		t.Fatalf("quotas must not include frozen_balance: %+v", body)
	}
}
