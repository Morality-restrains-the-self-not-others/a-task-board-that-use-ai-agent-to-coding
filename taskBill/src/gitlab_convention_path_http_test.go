package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleGitlabRegionsList_ConventionKVPath(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-regions/tenant_id/877397588196749312/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convention gitlab-regions path: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Regions []map[string]interface{} `json:"regions"`
		Total   int                      `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Total < 1 || len(out.Regions) < 1 {
		t.Fatalf("expected active regions, got %+v", out)
	}
	for _, r := range out.Regions {
		if _, has := r["admin_private_token"]; has {
			t.Fatalf("public region list must not leak admin_private_token: %+v", r)
		}
		if slug, _ := r["slug"].(string); slug == "" {
			t.Fatalf("region missing slug: %+v", r)
		}
	}
}

func TestHandleGitlabRegionsList_StripsStoredAdminToken(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	if _, err := db.Exec(`UPDATE billing_gitlab_region SET admin_private_token = 'glpat-secret-must-not-leak' WHERE is_active = 1`); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-regions/tenant_id/877397588196749312/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "glpat-secret-must-not-leak") {
		t.Fatalf("tenant region list leaked admin_private_token: %s", rec.Body.String())
	}
}

func TestHandleGitlabRegionsList_LegacyPathStillWorks(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/877397588196749312/billing/gitlab-regions/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy gitlab-regions path: code=%d body=%s", rec.Code, rec.Body.String())
	}
}

// Navbar「代码仓库」下拉：无 region 时返回租户已购/获赠区域列表（resources[]），
// 不再 400 — 与带 ?region= 的单区详情契约并存。
func TestHandleGitlabResources_WithoutRegionReturnsResourcesList(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tenantID int64 = 9300000011
	if _, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 1, Region: "tencent-sh-1", Reason: "navbar-list"},
	}, "admin-1", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-resources/tenant_id/9300000011/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("without region: code=%d body=%s want 200", rec.Code, rec.Body.String())
	}
	var out struct {
		TenantID  string                   `json:"tenant_id"`
		Resources []map[string]interface{} `json:"resources"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.TenantID != "9300000011" {
		t.Fatalf("tenant_id=%q", out.TenantID)
	}
	if len(out.Resources) < 1 {
		t.Fatalf("expected >=1 resources, body=%s", rec.Body.String())
	}
	found := false
	for _, r := range out.Resources {
		if r["region"] == "tencent-sh-1" {
			found = true
			if web, _ := r["gitlab_web_url"].(string); strings.TrimSpace(web) == "" {
				t.Fatalf("missing gitlab_web_url: %+v", r)
			}
			if name, _ := r["region_name"].(string); strings.TrimSpace(name) == "" {
				t.Fatalf("missing region_name: %+v", r)
			}
		}
	}
	if !found {
		t.Fatalf("tencent-sh-1 not in resources: %+v", out.Resources)
	}
}

func TestHandleGitlabResources_WithoutRegionEmptyTenantReturnsEmptyList(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/9300000012/billing/gitlab-resources/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("empty tenant list: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Resources []map[string]interface{} `json:"resources"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Resources == nil {
		t.Fatal("resources must be [] not null")
	}
	if len(out.Resources) != 0 {
		t.Fatalf("want empty resources, got %+v", out.Resources)
	}
}

func TestHandleGitlabResources_ConventionKVWithRegionReturnsGranted(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tenantID int64 = 9300000001
	if _, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 1, Region: "tencent-sh-1", Reason: "gift"},
	}, "admin-1", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-resources/tenant_id/9300000001/?region=tencent-sh-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("convention gitlab-resources with region: code=%d body=%s", rec.Code, rec.Body.String())
	}
	var view map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view["region"] != "tencent-sh-1" {
		t.Fatalf("region=%v want tencent-sh-1 body=%s", view["region"], rec.Body.String())
	}
	if view["provisioning_status"] != "active" {
		t.Fatalf("provisioning_status=%v want active", view["provisioning_status"])
	}
	diskGB, ok := view["disk_gb"].(float64)
	if !ok || diskGB != 1 {
		t.Fatalf("disk_gb=%v want 1 body=%s", view["disk_gb"], rec.Body.String())
	}
	web, _ := view["gitlab_web_url"].(string)
	if web == "" {
		t.Fatalf("missing gitlab_web_url: %+v", view)
	}
}

func TestHandleGitlabResources_ConventionKVEmptyTenantNotPurchased(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-resources/tenant_id/9300000002/?region=tencent-sh-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var view map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if view["provisioning_status"] != "not_purchased" {
		t.Fatalf("empty tenant should be not_purchased, got %v body=%s", view["provisioning_status"], rec.Body.String())
	}
}

func TestHandleGitlabResources_ConventionKVWithRegionQueryNot404(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodGet, "/api/billing/gitlab-resources/tenant_id/9300000002/?region=tencent-sh-1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Fatalf("convention gitlab-resources+region 404: body=%s", rec.Body.String())
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGitlabResourcesPurchase_ConventionKVNot404(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/api/billing/gitlab-resources/tenant_id/9300000003/purchase/", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Fatalf("convention purchase path 404: body=%s", rec.Body.String())
	}
}
