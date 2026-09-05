package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestToggleActiveSetsExclusive(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth1")
	seedCloudAuth(t, "t1", "auth2")

	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/cloud/toggle-active/", strings.NewReader(`{"id":"auth1"}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()
	handleToggleActive(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"is_active":true`) {
		t.Fatalf("expected active true: %s", rec.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/cloud/toggle-active/", strings.NewReader(`{"id":"auth2"}`))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	req2.Header.Set("X-User-Id", "test-user")
	rec2 := httptest.NewRecorder()
	handleToggleActive(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("auth2 status=%d", rec2.Code)
	}
	if authIsActive("t1", "auth1", "access_key") {
		t.Fatalf("auth1 should be deactivated after auth2 toggle")
	}
	if !authIsActive("t1", "auth2", "access_key") {
		t.Fatalf("auth2 should be active")
	}
}

func TestActiveListReturnsRows(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth1")
	_, err := setExclusiveActiveMethod("t1", "aliyun", "access_key", "auth1")
	if err != nil {
		t.Fatalf("seed active: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/cloud/active-list/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()
	handleActiveList(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"auth_instance_id":"auth1"`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestToggleActiveFalseDeactivates(t *testing.T) {
	setupCloudTestDB(t)
	seedCloudAuth(t, "t1", "auth1")
	if _, err := setExclusiveActiveMethod("t1", "aliyun", "access_key", "auth1"); err != nil {
		t.Fatalf("seed active: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/cloud/toggle-active/", strings.NewReader(`{"id":"auth1","is_active":false}`))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-User-Id", "test-user")
	rec := httptest.NewRecorder()
	handleToggleActive(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if authIsActive("t1", "auth1", "access_key") {
		t.Fatalf("toggle is_active:false must deactivate")
	}
}
