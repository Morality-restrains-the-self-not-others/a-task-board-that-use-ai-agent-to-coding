package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAidevParseYAML(t *testing.T) {
	setupTestDB(t)
	body := `{"yaml":"version: 1\nservice_id: foo\ntags:\n  - svc:foo\n"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/daydaymoney/parse-yaml", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleAidevParseYAML(rec, req, "t1")
	if rec.Code != 200 {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&out)
	if out["service_id"] != "foo" {
		t.Fatalf("out=%v", out)
	}
}

func TestAidevParseYAML_Forbidden(t *testing.T) {
	setupTestDB(t)
	body := `{"yaml":"version: 1\nservice_id: foo\nworkspace_id: w1\ntags:\n  - svc:foo\n"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/daydaymoney/parse-yaml", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleAidevParseYAML(rec, req, "t1")
	if rec.Code != 400 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestAidevResolve_MultiWorkspace(t *testing.T) {
	setupTestDB(t)

	createWS := func(name string) string {
		t.Helper()
		wsBody := `{"name":"` + name + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/workspaces/", strings.NewReader(wsBody))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCreateWorkspace(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("create ws %s: %d %s", name, rec.Code, rec.Body.String())
		}
		var ws map[string]interface{}
		json.NewDecoder(rec.Body).Decode(&ws)
		id, _ := ws["id"].(string)
		if id == "" {
			t.Fatalf("missing id for %s", name)
		}
		return id
	}
	w1 := createWS("WS1")
	w2 := createWS("WS2")

	p1 := `{"name":"ProjA","tags":["svc:demoSvc"],"workspaces_ids":["` + w1 + `"]}`
	p2 := `{"name":"ProjB","tags":["SVC:demosvc"],"workspaces_ids":["` + w2 + `"]}`
	for _, body := range []string{p1, p2} {
		req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(body))
		req.Header.Set("X-Auth-Tenant-Id", "t1")
		rec := httptest.NewRecorder()
		handleCreateProject(rec, req)
		if rec.Code != 201 {
			t.Fatalf("create project: %d %s", rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/daydaymoney/resolve?service_id=demoSvc", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleAidevResolve(rec, req, "t1")
	if rec.Code != 200 {
		t.Fatalf("resolve: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&out)
	matches, _ := out["matches"].([]interface{})
	if len(matches) < 2 {
		t.Fatalf("want >=2 matches, got %#v", out)
	}
	wsSeen := map[string]bool{}
	for _, raw := range matches {
		m := raw.(map[string]interface{})
		wsSeen[m["workspace_id"].(string)] = true
	}
	if !wsSeen[w1] || !wsSeen[w2] {
		t.Fatalf("workspaces not both present: %v", wsSeen)
	}
}

func TestAidevResolve_EmptyQuery(t *testing.T) {
	setupTestDB(t)
	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/daydaymoney/resolve", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleAidevResolve(rec, req, "t1")
	if rec.Code != 400 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestListProjects_TagFilter(t *testing.T) {
	setupTestDB(t)
	body := `{"name":"Tagged","tags":["svc:alpha"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(body))
	req.Header.Set("X-Auth-Tenant-Id", "t1")
	rec := httptest.NewRecorder()
	handleCreateProject(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create: %d", rec.Code)
	}
	body2 := `{"name":"Other","tags":["svc:beta"]}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/tenant/t1/projects/", strings.NewReader(body2))
	req2.Header.Set("X-Auth-Tenant-Id", "t1")
	rec2 := httptest.NewRecorder()
	handleCreateProject(rec2, req2)

	req3 := httptest.NewRequest(http.MethodGet, "/api/tenant/t1/projects/?tag=svc:alpha", nil)
	req3.Header.Set("X-Auth-Tenant-Id", "t1")
	rec3 := httptest.NewRecorder()
	handleListProjects(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("list: %d", rec3.Code)
	}
	var list []map[string]interface{}
	json.NewDecoder(rec3.Body).Decode(&list)
	if len(list) != 1 || list[0]["name"] != "Tagged" {
		t.Fatalf("list=%v", list)
	}
}
