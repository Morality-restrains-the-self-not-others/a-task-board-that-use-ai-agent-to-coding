package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompaniesGetByID(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies?company_id=c1", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["name"] != "Test Co" {
		t.Fatalf("expected name='Test Co', got %v", out)
	}
}

func TestCompaniesGetByIDMissingParam(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCompaniesGetByIDNotFound(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies?company_id=nonexist", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCompaniesByName(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/by-name?name=Test+Co", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["name"] != "Test Co" {
		t.Fatalf("expected name='Test Co', got %v", out)
	}
}

func TestCompaniesByNameNotFound(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/by-name?name=NonExistent", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCompaniesByCreator(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/by-creator?creator_id=admin1", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var list []interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) < 1 {
		t.Fatalf("expected at least 1 company, got %v", string(rec.Body.Bytes()))
	}
}

func TestCompaniesByCreatorEmpty(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/by-creator?creator_id=nonexist", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCompaniesNameTaken(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/name-taken?name=Test+Co", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["taken"] != true {
		t.Fatalf("expected taken=true, got %v", out)
	}

	req2 := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/name-taken?name=FreeName&exclude_id=c1", nil), "admin1")
	rec2 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec2, req2)
	var out2 map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &out2)
	if out2["taken"] != false {
		t.Fatalf("expected taken=false for free name, got %v", out2)
	}
}

func TestCompaniesUnauthenticated(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies?company_id=c1", nil)
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 for unauthenticated, got %d", rec.Code)
	}
}

func TestCompaniesNotFoundRoute(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/invalid-route", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected 404 for invalid route, got %d", rec.Code)
	}
}

func TestCompaniesCreateCompany(t *testing.T) {
	mux := setupTestService(t)

	body := `{"name":"New Test Company"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/_/accounts/companies/", bytes.NewBufferString(body)), "admin1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["company_id"] == nil || out["company_id"] == "" {
		t.Fatalf("expected company_id, got %v", out)
	}
	if out["name"] != "New Test Company" {
		t.Fatalf("expected name='New Test Company', got %v", out["name"])
	}
	// Verify company was persisted
	cid, _ := out["company_id"].(string)
	c, err := getCompanyByID(cid)
	if err != nil || c == nil {
		t.Fatalf("company not persisted: err=%v c=%v", err, c)
	}
	// Verify creator was added as admin member
	var memberExists int
	db.QueryRow(`SELECT COUNT(1) FROM tenant_company_member WHERE company_id=? AND user_id=? AND is_admin=1`, cid, "admin1").Scan(&memberExists)
	if memberExists != 1 {
		t.Fatalf("expected creator admin member, got %d", memberExists)
	}
}

func TestCompaniesCreateCompanyEmptyName(t *testing.T) {
	mux := setupTestService(t)

	body := `{"name":"  "}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/_/accounts/companies/", bytes.NewBufferString(body)), "admin1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for empty name, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCompaniesCreateCompanyUnauthenticated(t *testing.T) {
	mux := setupTestService(t)

	body := `{"name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/_/accounts/companies/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 for unauthenticated POST, got %d", rec.Code)
	}
}

// ── companies/current ──────────────────────────────────────────────────────────

func TestCompaniesCurrentGetAsAdmin(t *testing.T) {
	mux := setupTestService(t)

	// admin1 is both creator (tenant_company.creator_id) and is_admin=1
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/current/", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["name"] != "Test Co" {
		t.Fatalf("expected name='Test Co', got %v", out["name"])
	}
	if out["member_is_admin"] != true {
		t.Fatalf("expected member_is_admin=true, got %v", out["member_is_admin"])
	}
	if out["member_is_creator"] != true {
		t.Fatalf("expected member_is_creator=true, got %v", out["member_is_creator"])
	}
}

func TestCompaniesCurrentGetAsNonAdminMember(t *testing.T) {
	mux := setupTestService(t)

	// Seed a non-admin member for c1
	db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m2','member1','c1',0,1,'','Member')`)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/current/", nil), "member1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["name"] != "Test Co" {
		t.Fatalf("expected name='Test Co', got %v", out["name"])
	}
	if out["member_is_admin"] != false {
		t.Fatalf("expected member_is_admin=false for non-admin member, got %v", out["member_is_admin"])
	}
	if out["member_is_creator"] != false {
		t.Fatalf("expected member_is_creator=false, got %v", out["member_is_creator"])
	}
}

func TestCompaniesCurrentGetUnauthenticated(t *testing.T) {
	mux := setupTestService(t)

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/c1/accounts/companies/current/", nil)
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("expected 401 for unauthenticated, got %d", rec.Code)
	}
}

func TestCompaniesCurrentGetNotFound(t *testing.T) {
	mux := setupTestService(t)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tenant/nonexist/accounts/companies/current/", nil), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("expected 404 for nonexistent company, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCompaniesCurrentPatchAsAdmin(t *testing.T) {
	mux := setupTestService(t)

	body := `{"name":"Updated Company Name"}`
	req := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/companies/current/", bytes.NewBufferString(body)), "admin1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["name"] != "Updated Company Name" {
		t.Fatalf("expected name='Updated Company Name', got %v", out["name"])
	}
	// Verify persistence
	c, _ := getCompanyByID("c1")
	if c == nil || c.Name != "Updated Company Name" {
		t.Fatalf("company name not persisted: %v", c)
	}
}

func TestCompaniesCurrentPatchAsNonAdmin(t *testing.T) {
	mux := setupTestService(t)

	// Seed a non-admin member for c1
	db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m3','member2','c1',0,1,'','Member2')`)

	body := `{"name":"Hacked Name"}`
	req := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/companies/current/", bytes.NewBufferString(body)), "member2")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("expected 403 for non-admin PATCH, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCompaniesCurrentPatchEmptyName(t *testing.T) {
	mux := setupTestService(t)

	body := `{"name":"  "}`
	req := withUser(httptest.NewRequest(http.MethodPatch, "/api/tenant/c1/accounts/companies/current/", bytes.NewBufferString(body)), "admin1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("expected 400 for empty name, got %d body=%s", rec.Code, rec.Body.String())
	}
}
