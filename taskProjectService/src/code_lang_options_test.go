package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func doCodeLangOptions(method, tenant, ws, user, body string) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/api/projects/workspaces/%s/code-lang-options/tenant_id/%s/", ws, tenant)
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenant)
	if user != "" {
		req.Header.Set("X-Auth-User-Id", user)
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mountRoutes(mux)
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	return rec
}

func TestCodeLangOptionsGetDefault(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	rec := doCodeLangOptions(http.MethodGet, "t1", "ws1", "u1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	opts, _ := body["options"].([]interface{})
	if len(opts) != 3 {
		t.Fatalf("expected 3 default options, got %d (%v)", len(opts), opts)
	}
	if fmt.Sprint(opts[0]) != "go" || fmt.Sprint(opts[1]) != "rust" || fmt.Sprint(opts[2]) != "js" {
		t.Fatalf("unexpected defaults: %v", opts)
	}
}

func TestCodeLangOptionsPutGetRoundtrip(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	rec := doCodeLangOptions(http.MethodPut, "t1", "ws1", "u1", `{"options":["go","js","python"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec2 := doCodeLangOptions(http.MethodGet, "t1", "ws1", "u1", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", rec2.Code)
	}
	var body map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&body)
	opts, _ := body["options"].([]interface{})
	if len(opts) != 3 {
		t.Fatalf("expected 3 options, got %d", len(opts))
	}
}

func TestCodeLangOptionsWorkspaceIsolation(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "wsA")
	seedWorkspace(t, "t1", "wsB")

	doCodeLangOptions(http.MethodPut, "t1", "wsA", "u1", `{"options":["go"]}`)
	doCodeLangOptions(http.MethodPut, "t1", "wsB", "u1", `{"options":["rust","js"]}`)

	recA := doCodeLangOptions(http.MethodGet, "t1", "wsA", "u1", "")
	recB := doCodeLangOptions(http.MethodGet, "t1", "wsB", "u1", "")
	var bodyA, bodyB map[string]interface{}
	json.NewDecoder(recA.Body).Decode(&bodyA)
	json.NewDecoder(recB.Body).Decode(&bodyB)
	optsA, _ := bodyA["options"].([]interface{})
	optsB, _ := bodyB["options"].([]interface{})
	if len(optsA) != 1 || fmt.Sprint(optsA[0]) != "go" {
		t.Fatalf("wsA options=%v", optsA)
	}
	if len(optsB) != 2 {
		t.Fatalf("wsB options=%v", optsB)
	}
}

func TestNormalizeCodeLangOptionsDedupEmpty(t *testing.T) {
	opts, err := normalizeCodeLangOptions([]string{" js ", "JS", "", "go"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(opts) != 2 || opts[0] != "js" || opts[1] != "go" {
		t.Fatalf("got %v", opts)
	}
}
