package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompanyCreatorInternal_LocalSSOT(t *testing.T) {
	mux := setupTestService(t)
	// c1 already seeded in setupTestService

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/companies/creator?company_id=c1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["found"] != true || out["creator_id"] != "admin1" {
		t.Fatalf("out=%v", out)
	}

	reqByID := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/companies/by-id?company_id=c1", nil)
	recByID := httptest.NewRecorder()
	mux.ServeHTTP(recByID, reqByID)
	if recByID.Code != 200 {
		t.Fatalf("by-id status=%d body=%s", recByID.Code, recByID.Body.String())
	}
	var co map[string]interface{}
	_ = json.Unmarshal(recByID.Body.Bytes(), &co)
	if co["name"] != "Test Co" {
		t.Fatalf("by-id=%v", co)
	}

	reqSearch := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/companies/search?q=Test", nil)
	recSearch := httptest.NewRecorder()
	mux.ServeHTTP(recSearch, reqSearch)
	if recSearch.Code != 200 {
		t.Fatalf("search status=%d", recSearch.Code)
	}
}
