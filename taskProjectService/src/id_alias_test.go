package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func resetIDAliases(t *testing.T) {
	t.Helper()
	clearIDAliases()
	t.Cleanup(clearIDAliases)
}

func TestIsOverflowNegativeID(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"ws_-2309487803472456748", true},
		{"ws_-2740859684112864748", true},
		{"proj_-2304947540687519745", true},
		{"ws_1234567890123456789", false},
		{"proj_1234567890123456789", false},
		{"ws_-", false},
		{"task_-1", false},
		{"", false},
		{"ws_-abc", false},
	}
	for _, tc := range cases {
		if got := IsOverflowNegativeID(tc.id); got != tc.want {
			t.Errorf("IsOverflowNegativeID(%q)=%v want %v", tc.id, got, tc.want)
		}
	}
}

func TestKnownOverflowInventoryIsTwoWorkspacesAndOneProject(t *testing.T) {
	if len(KnownOverflowWorkspaceIDs) != 2 {
		t.Fatalf("KnownOverflowWorkspaceIDs len=%d want 2", len(KnownOverflowWorkspaceIDs))
	}
	if len(KnownOverflowProjectIDs) != 1 {
		t.Fatalf("KnownOverflowProjectIDs len=%d want 1", len(KnownOverflowProjectIDs))
	}
	for _, id := range KnownOverflowWorkspaceIDs {
		if !IsOverflowNegativeID(id) || !strings.HasPrefix(id, "ws_-") {
			t.Errorf("workspace inventory id %q is not ws_- overflow", id)
		}
	}
	for _, id := range KnownOverflowProjectIDs {
		if !IsOverflowNegativeID(id) || !strings.HasPrefix(id, "proj_-") {
			t.Errorf("project inventory id %q is not proj_- overflow", id)
		}
	}
}

func TestResolveStoredIDIdentityWhenEmpty(t *testing.T) {
	resetIDAliases(t)
	const overflow = "ws_-2309487803472456748"
	const snowflake = "ws_1234567890123456789"
	if got := resolveStoredID(idAliasKindWorkspace, overflow); got != overflow {
		t.Fatalf("empty map overflow resolve=%q want identity %q", got, overflow)
	}
	if got := resolveStoredID(idAliasKindWorkspace, snowflake); got != snowflake {
		t.Fatalf("empty map snowflake resolve=%q want identity %q", got, snowflake)
	}
}

func TestRegisterIDAliasMapsCanonicalToStored(t *testing.T) {
	resetIDAliases(t)
	const stored = "ws_-2309487803472456748"
	const canonical = "ws_1234567890123456789"
	if err := RegisterIDAlias(idAliasKindWorkspace, canonical, stored); err != nil {
		t.Fatalf("register: %v", err)
	}
	if got := resolveStoredID(idAliasKindWorkspace, canonical); got != stored {
		t.Fatalf("alias resolve=%q want stored %q", got, stored)
	}
	if got := resolveStoredID(idAliasKindWorkspace, stored); got != stored {
		t.Fatalf("overflow lookup must stay identity, got %q", got)
	}
	if err := RegisterIDAlias(idAliasKindWorkspace, stored, stored); err == nil {
		t.Fatal("expected error when canonical is itself overflow")
	}
	if err := RegisterIDAlias(idAliasKindProject, "proj_1", stored); err == nil {
		t.Fatal("expected error when stored prefix does not match kind")
	}
}

func TestGetWorkspaceOverflowNegativeIDStillOpens(t *testing.T) {
	setupTestDB(t)
	resetIDAliases(t)
	stored := KnownOverflowWorkspaceIDs[0]
	if _, err := db.Exec(
		`INSERT INTO project_workspace_entries(id, name, company_id) VALUES(?,?,?)`,
		stored, "溢出工作区", "t-overflow",
	); err != nil {
		t.Fatalf("insert overflow workspace: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t-overflow/workspaces/"+stored+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t-overflow")
	req.Header.Set("X-Trace-Id", "overflow-ws-get")
	rec := httptest.NewRecorder()
	handleWorkspacesRoute(rec, req, "t-overflow", []string{stored})
	if rec.Code != http.StatusOK {
		t.Fatalf("GET overflow workspace: %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["id"] != stored {
		t.Fatalf("id=%v want stored overflow %s", body["id"], stored)
	}
	if body["name"] != "溢出工作区" {
		t.Fatalf("name=%v want 溢出工作区", body["name"])
	}
}

func TestGetWorkspaceCanonicalAliasReturnsSameStoredID(t *testing.T) {
	setupTestDB(t)
	resetIDAliases(t)
	stored := KnownOverflowWorkspaceIDs[1]
	const canonical = "ws_9876543210987654321"
	if _, err := db.Exec(
		`INSERT INTO project_workspace_entries(id, name, company_id) VALUES(?,?,?)`,
		stored, "别名工作区", "t-alias",
	); err != nil {
		t.Fatalf("insert overflow workspace: %v", err)
	}
	if err := RegisterIDAlias(idAliasKindWorkspace, canonical, stored); err != nil {
		t.Fatalf("register alias: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t-alias/workspaces/"+canonical+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t-alias")
	req.Header.Set("X-Trace-Id", "alias-ws-get")
	rec := httptest.NewRecorder()
	handleWorkspacesRoute(rec, req, "t-alias", []string{canonical})
	if rec.Code != http.StatusOK {
		t.Fatalf("GET alias workspace: %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["id"] != stored {
		t.Fatalf("id=%v want stored overflow PK %s (alias must not rewrite JSON id)", body["id"], stored)
	}
}

func TestGetProjectOverflowNegativeIDStillOpens(t *testing.T) {
	setupTestDB(t)
	resetIDAliases(t)
	stored := KnownOverflowProjectIDs[0]
	if _, err := db.Exec(
		`INSERT INTO project_entries(id, name, company_id) VALUES(?,?,?)`,
		stored, "溢出项目", "t-overflow",
	); err != nil {
		t.Fatalf("insert overflow project: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t-overflow/projects/"+stored+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t-overflow")
	req.Header.Set("X-Trace-Id", "overflow-proj-get")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t-overflow", []string{stored})
	if rec.Code != http.StatusOK {
		t.Fatalf("GET overflow project: %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["id"] != stored {
		t.Fatalf("id=%v want stored overflow %s", body["id"], stored)
	}
}

func TestGetProjectCanonicalAliasReturnsSameStoredID(t *testing.T) {
	setupTestDB(t)
	resetIDAliases(t)
	stored := KnownOverflowProjectIDs[0]
	const canonical = "proj_1112223334445556667"
	if _, err := db.Exec(
		`INSERT INTO project_entries(id, name, company_id) VALUES(?,?,?)`,
		stored, "别名项目", "t-alias-p",
	); err != nil {
		t.Fatalf("insert overflow project: %v", err)
	}
	if err := RegisterIDAlias(idAliasKindProject, canonical, stored); err != nil {
		t.Fatalf("register alias: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/t-alias-p/projects/"+canonical+"/", nil)
	req.Header.Set("X-Auth-Tenant-Id", "t-alias-p")
	req.Header.Set("X-Trace-Id", "alias-proj-get")
	rec := httptest.NewRecorder()
	handleProjectsRoute(rec, req, "t-alias-p", []string{canonical})
	if rec.Code != http.StatusOK {
		t.Fatalf("GET alias project: %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["id"] != stored {
		t.Fatalf("id=%v want stored overflow PK %s", body["id"], stored)
	}
}

func TestInternalGetWorkspaceOverflowAndAlias(t *testing.T) {
	setupTestDB(t)
	resetIDAliases(t)
	stored := KnownOverflowWorkspaceIDs[0]
	const canonical = "ws_5555555555555555555"
	if _, err := db.Exec(
		`INSERT INTO project_workspace_entries(id, name, company_id) VALUES(?,?,?)`,
		stored, "内部溢出工作区", "t-internal",
	); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := RegisterIDAlias(idAliasKindWorkspace, canonical, stored); err != nil {
		t.Fatalf("register: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/workspaces/"+canonical+"/", nil)
	req.Header.Set("X-Trace-Id", "internal-alias-ws")
	rec := httptest.NewRecorder()
	handleInternalGetWorkspace(rec, req, canonical)
	if rec.Code != http.StatusOK {
		t.Fatalf("internal GET alias: %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["id"] != stored {
		t.Fatalf("id=%v want %s", body["id"], stored)
	}
}

func TestInternalWorkspacesBatchGetResolvesAlias(t *testing.T) {
	setupTestDB(t)
	resetIDAliases(t)
	stored := KnownOverflowWorkspaceIDs[1]
	const canonical = "ws_4444444444444444444"
	if _, err := db.Exec(
		`INSERT INTO project_workspace_entries(id, name, company_id) VALUES(?,?,?)`,
		stored, "批量别名工作区", "t-batch",
	); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := RegisterIDAlias(idAliasKindWorkspace, canonical, stored); err != nil {
		t.Fatalf("register: %v", err)
	}

	body := `{"workspace_ids":["` + canonical + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/workspaces/batch-get/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleInternalWorkspacesBatchGet(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("batch-get: %d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	list, _ := payload["workspaces"].([]interface{})
	if len(list) != 1 {
		t.Fatalf("workspaces=%v want 1 row", payload["workspaces"])
	}
	row, _ := list[0].(map[string]interface{})
	if row["id"] != stored {
		t.Fatalf("id=%v want stored %s", row["id"], stored)
	}
}
