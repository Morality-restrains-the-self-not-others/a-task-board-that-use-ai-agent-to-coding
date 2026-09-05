package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func doWorkspaceSettings(method, path string, form url.Values) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("X-Auth-User-Id", "u1")
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	return rec
}

func TestWorkspaceInfoGet(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	rec := doWorkspaceSettings(http.MethodGet, "/api/workspace/info/?workspace_id=ws1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body["status"])
	}
	ws, _ := body["workspace"].(map[string]interface{})
	if ws == nil || ws["name"] == nil || ws["description"] == nil {
		t.Fatalf("expected workspace{name,description}, got %v", body["workspace"])
	}
	if ws["name"] != "WS ws1" {
		t.Fatalf("expected name=WS ws1, got %v", ws["name"])
	}
}

func TestWorkspaceInfoMissingWorkspaceID(t *testing.T) {
	setupTestDB(t)
	rec := doWorkspaceSettings(http.MethodGet, "/api/workspace/info/", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceInfoNotFound(t *testing.T) {
	setupTestDB(t)
	rec := doWorkspaceSettings(http.MethodGet, "/api/workspace/info/?workspace_id=nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceSettingsUpdateBasicInfo(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	form := url.Values{}
	form.Set("action", "update_basic_info")
	form.Set("workspace_id", "ws1")
	form.Set("name", "新名称")
	form.Set("description", "新描述")
	rec := doWorkspaceSettings(http.MethodPost, "/api/workspace/settings/", form)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body)
	}

	// 回读验证
	rec2 := doWorkspaceSettings(http.MethodGet, "/api/workspace/info/?workspace_id=ws1", nil)
	var got map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&got)
	ws, _ := got["workspace"].(map[string]interface{})
	if ws["name"] != "新名称" || ws["description"] != "新描述" {
		t.Fatalf("expected updated name/description, got %v", ws)
	}
}

func TestWorkspaceSettingsUnsupportedAction(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	// 半成品区块（通知/安全/高级）：明确 501 而非网关 502
	for _, action := range []string{"update_notification_settings", "update_security_settings", "update_advanced_settings"} {
		form := url.Values{}
		form.Set("action", action)
		form.Set("workspace_id", "ws1")
		rec := doWorkspaceSettings(http.MethodPost, "/api/workspace/settings/", form)
		if rec.Code != http.StatusNotImplemented {
			t.Fatalf("action %s: expected 501, got %d: %s", action, rec.Code, rec.Body.String())
		}
	}
}

func TestWorkspaceSettingsRequiresAuth(t *testing.T) {
	setupTestDB(t)
	req := httptest.NewRequest(http.MethodPost, "/api/workspace/settings/", strings.NewReader("action=update_basic_info&workspace_id=ws1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without user, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceInfoMethodNotAllowed(t *testing.T) {
	setupTestDB(t)
	rec := doWorkspaceSettings(http.MethodDelete, "/api/workspace/info/?workspace_id=ws1", nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rec.Code, rec.Body.String())
	}
}
