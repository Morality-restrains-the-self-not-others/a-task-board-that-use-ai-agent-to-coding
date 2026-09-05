package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalTenantPersonalDataMemberships(t *testing.T) {
	mux := setupTestService(t)
	// admin1 已播种：公司 c1 管理员成员 m1（见 people_api_test.go 播种模式）
	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/users/admin1/personal-data/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var wrapper struct {
		Data struct {
			Memberships []map[string]interface{} `json:"memberships"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(wrapper.Data.Memberships) != 1 {
		t.Fatalf("memberships=%d want 1", len(wrapper.Data.Memberships))
	}
	m := wrapper.Data.Memberships[0]
	if m["company_id"] != "c1" || m["user_id"] != "admin1" {
		t.Fatalf("unexpected membership: %v", m)
	}
}

func TestInternalTenantPersonalDataEmptyUser(t *testing.T) {
	mux := setupTestService(t)
	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/users/unknown-user-xyz/personal-data/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var wrapper struct {
		Data struct {
			Memberships []map[string]interface{} `json:"memberships"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrapper); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(wrapper.Data.Memberships) != 0 {
		t.Fatalf("memberships=%d want 0", len(wrapper.Data.Memberships))
	}
}

func TestInternalTenantPersonalDataRequiresSecret(t *testing.T) {
	mux := setupTestService(t)
	prev := cfg.InternalSecret
	cfg.InternalSecret = "tenant-secret"
	t.Cleanup(func() { cfg.InternalSecret = prev })

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/users/admin1/personal-data/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", w.Code)
	}
}
