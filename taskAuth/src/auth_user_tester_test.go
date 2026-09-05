package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseSystemAdminUserListFiltersAcceptsTester(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?role=tester", nil)
	f := parseSystemAdminUserListFilters(req)
	if f.role != "tester" {
		t.Fatalf("role=%q", f.role)
	}
	clauses, _ := f.localWhere(nil, "", false)
	joined := strings.Join(clauses, " AND ")
	if !strings.Contains(joined, "is_tester") {
		t.Fatalf("tester filter SQL=%q", joined)
	}
}

func TestParseSystemAdminUserListFiltersTenantExcludesTester(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?role=tenant", nil)
	f := parseSystemAdminUserListFilters(req)
	clauses, _ := f.localWhere(nil, "", false)
	joined := strings.Join(clauses, " AND ")
	if !strings.Contains(joined, "is_tester") {
		t.Fatalf("tenant filter must exclude testers: %q", joined)
	}
}

func TestEventTopicMapUserTesterFlagChanged(t *testing.T) {
	if eventTopicMap["UserTesterFlagChanged"] != "user-tester-flag-changed" {
		t.Fatalf("topic=%q", eventTopicMap["UserTesterFlagChanged"])
	}
}

func TestPatchUserAsAdminTesterForcesTenant(t *testing.T) {
	setupAuthTestDB(t)
	targetID, _, err := createUserWithEmailLogin("tester-flag@test.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	body, _ := json.Marshal(map[string]bool{"is_tester": true, "is_tenant": false})
	req := httptest.NewRequest(http.MethodPatch, "/api/system-admin/users/"+targetID+"/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", "bootstrap-admin")
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var isTester, isTenant int
	if err := db.QueryRow(`SELECT COALESCE(is_tester,0), COALESCE(is_tenant,0) FROM auth_user WHERE id = ?`, targetID).Scan(&isTester, &isTenant); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if isTester != 1 || isTenant != 1 {
		t.Fatalf("want tester+tenant, got tester=%d tenant=%d", isTester, isTenant)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["is_tester"] != true {
		t.Fatalf("json is_tester=%v", payload["is_tester"])
	}
}

func TestSystemAdminListUsersRoleTesterFilter(t *testing.T) {
	setupAuthTestDB(t)
	testerID, _, err := createUserWithEmailLogin("role-tester@test.com", "hash")
	if err != nil {
		t.Fatalf("tester: %v", err)
	}
	tenantID, _, err := createUserWithEmailLogin("role-tenant-only@test.com", "hash")
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_tester = 1, is_tenant = 1 WHERE id = ?`, testerID); err != nil {
		t.Fatalf("mark tester: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_user SET is_tenant = 1, is_tester = 0 WHERE id = ?`, tenantID); err != nil {
		t.Fatalf("mark tenant: %v", err)
	}

	_, users, _ := listSystemAdminUsers(t, "role=tester", "bootstrap-admin")
	ids := userIDsInList(users)
	if !ids[testerID] {
		t.Fatalf("expected tester, got %v", ids)
	}
	if ids[tenantID] {
		t.Fatalf("plain tenant must not appear in role=tester, got %v", ids)
	}

	_, users, _ = listSystemAdminUsers(t, "role=tenant", "bootstrap-admin")
	ids = userIDsInList(users)
	if !ids[tenantID] {
		t.Fatalf("expected tenant-only, got %v", ids)
	}
	if ids[testerID] {
		t.Fatalf("tester must not appear in role=tenant, got %v", ids)
	}
}
