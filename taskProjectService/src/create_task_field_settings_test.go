package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func doCreateTaskFieldSettings(method, tenant, ws, user, body string) *httptest.ResponseRecorder {
	url := fmt.Sprintf("/api/projects/workspaces/%s/create-task-field-settings/tenant_id/%s/", ws, tenant)
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

func TestCreateTaskFieldSettingsGetDefault(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	rec := doCreateTaskFieldSettings(http.MethodGet, "t1", "ws1", "u1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&body)
	fields, _ := body["fields"].(map[string]interface{})
	if len(fields) != len(knownCreateTaskFieldKeys) {
		t.Fatalf("expected %d keys, got %d (%v)", len(knownCreateTaskFieldKeys), len(fields), fields)
	}
	for _, k := range knownCreateTaskFieldKeys {
		want := true
		if defaultDisabledCreateTaskFieldKeys[k] {
			want = false
		}
		if fields[k] != want {
			t.Fatalf("expected %s=%v, got %v", k, want, fields[k])
		}
	}
}

func TestCreateTaskFieldSettingsPutGetRoundtrip(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "ws1")

	rec := doCreateTaskFieldSettings(http.MethodPut, "t1", "ws1", "u1", `{"fields":{"priority":false,"due_date":false}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec2 := doCreateTaskFieldSettings(http.MethodGet, "t1", "ws1", "u1", "")
	if rec2.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", rec2.Code)
	}
	var body map[string]interface{}
	json.NewDecoder(rec2.Body).Decode(&body)
	fields, _ := body["fields"].(map[string]interface{})
	if fields["priority"] != false || fields["due_date"] != false {
		t.Fatalf("expected priority/due_date false, got %v", fields)
	}
	if fields["task_kind"] != true {
		t.Fatalf("expected task_kind true default, got %v", fields["task_kind"])
	}
}

func TestCreateTaskFieldSettingsWorkspaceIsolation(t *testing.T) {
	setupTestDB(t)
	seedWorkspace(t, "t1", "wsA")
	seedWorkspace(t, "t1", "wsB")

	doCreateTaskFieldSettings(http.MethodPut, "t1", "wsA", "u1", `{"fields":{"priority":false}}`)
	doCreateTaskFieldSettings(http.MethodPut, "t1", "wsB", "u1", `{"fields":{"assignees":false}}`)

	recA := doCreateTaskFieldSettings(http.MethodGet, "t1", "wsA", "u1", "")
	recB := doCreateTaskFieldSettings(http.MethodGet, "t1", "wsB", "u1", "")
	var bodyA, bodyB map[string]interface{}
	json.NewDecoder(recA.Body).Decode(&bodyA)
	json.NewDecoder(recB.Body).Decode(&bodyB)
	fieldsA, _ := bodyA["fields"].(map[string]interface{})
	fieldsB, _ := bodyB["fields"].(map[string]interface{})
	if fieldsA["priority"] != false || fieldsA["assignees"] != true {
		t.Fatalf("wsA fields=%v", fieldsA)
	}
	if fieldsB["assignees"] != false || fieldsB["priority"] != true {
		t.Fatalf("wsB fields=%v", fieldsB)
	}
}

func TestNormalizeCreateTaskFieldSettingsUnknownIgnored(t *testing.T) {
	out := normalizeCreateTaskFieldSettings(map[string]interface{}{
		"priority":     false,
		"unknown_key":  false,
		"task_kind":    "false",
		"code_lang":    0,
		"auto_run":     "yes",
	})
	if out["priority"] != false {
		t.Fatalf("priority=%v", out["priority"])
	}
	if _, ok := out["unknown_key"]; ok {
		t.Fatalf("unknown_key should be dropped")
	}
	if out["task_kind"] != false {
		t.Fatalf("task_kind string false → bool false, got %v", out["task_kind"])
	}
	if out["code_lang"] != false {
		t.Fatalf("code_lang 0 → false, got %v", out["code_lang"])
	}
	if out["auto_run"] != true {
		t.Fatalf("auto_run yes → true, got %v", out["auto_run"])
	}
	if out["owner"] != true {
		t.Fatalf("missing owner should default true")
	}
	if out["structured_fields"] != false {
		t.Fatalf("missing structured_fields should default false, got %v", out["structured_fields"])
	}
}
