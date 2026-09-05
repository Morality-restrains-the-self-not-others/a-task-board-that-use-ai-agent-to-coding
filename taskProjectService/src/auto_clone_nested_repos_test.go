package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBoolFieldDefault(t *testing.T) {
	if !boolFieldDefault(map[string]interface{}{}, "auto_clone_nested_repos", true) {
		t.Fatal("missing key should use default true")
	}
	if boolFieldDefault(map[string]interface{}{"auto_clone_nested_repos": false}, "auto_clone_nested_repos", true) {
		t.Fatal("false should win over default")
	}
	if !boolFieldDefault(map[string]interface{}{"auto_clone_nested_repos": float64(1)}, "auto_clone_nested_repos", false) {
		t.Fatal("1 should be true")
	}
	if boolFieldDefault(map[string]interface{}{"auto_clone_nested_repos": "0"}, "auto_clone_nested_repos", true) {
		t.Fatal("\"0\" should be false")
	}
}

func TestCreateProjectAutoCloneNestedReposDefaultTrue(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"AutoCloneDefault","description":"d","workspaces_ids":["ws1"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	req.Header.Set("X-Auth-User-Id", "u1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	v, ok := created["auto_clone_nested_repos"].(bool)
	if !ok || !v {
		t.Fatalf("expected auto_clone_nested_repos=true, got %#v", created["auto_clone_nested_repos"])
	}
}

func TestCreateAndUpdateProjectAutoCloneNestedReposFalse(t *testing.T) {
	setupTestDB(t)

	body := `{"name":"AutoCloneOff","description":"d","auto_clone_nested_repos":false,"workspaces_ids":["ws1"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects/tenant_id/t1/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if created["auto_clone_nested_repos"] != false {
		t.Fatalf("create false: got %#v", created["auto_clone_nested_repos"])
	}
	pid := created["id"].(string)

	body2 := `{"auto_clone_nested_repos":true}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/projects/"+pid+"/tenant_id/t1/", strings.NewReader(body2))
	req2.Header.Set("X-Resource-Id", pid)
	rec2 := httptest.NewRecorder()
	handleUpdateProject(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var updated map[string]interface{}
	_ = json.NewDecoder(rec2.Body).Decode(&updated)
	if updated["auto_clone_nested_repos"] != true {
		t.Fatalf("update true: got %#v", updated["auto_clone_nested_repos"])
	}
}
