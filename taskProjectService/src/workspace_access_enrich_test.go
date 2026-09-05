package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildWorkspaceCollaborators_PureGo(t *testing.T) {
	setupTestDB(t)
	tenantID := "850256677331562496"
	caller := "850256676127797248"
	wsID := "861623708318031872"
	_, _ = db.Exec(`INSERT INTO project_workspace_entries(id,company_id,name) VALUES(?,?,?)`, wsID, tenantID, "研发")
	_, _ = db.Exec(
		`INSERT INTO project_workspace_accesses(id,workspace_id,user_id,group_id,permission) VALUES(?,?,?,?,?)`,
		"wa1", wsID, caller, "", "admin",
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members/resolve", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "cm1", "user_id": caller, "company_id": tenantID,
			"is_admin": true, "member_name": "公司创建者",
		})
	})
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id": "cm1", "user_id": caller, "company_id": tenantID,
				"is_admin": true, "member_name": "公司创建者",
				"member_avatar_url": "/api/tenant/" + tenantID + "/accounts/members/cm1/avatar",
				"created_at":        "2026-06-04T06:16:28Z",
			},
			{
				"id": "cm2", "user_id": "u-other", "company_id": tenantID,
				"is_admin": false, "member_name": "其他人",
			},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	old := cfg.TaskTenantURL
	cfg.TaskTenantURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantURL = old })

	rows, _ := loadWorkspaceAccessRows(wsID)
	out, status, err := buildWorkspaceCollaborators(tenantID, caller, rows)
	if err != nil || status != 200 {
		t.Fatalf("buildWorkspaceCollaborators: status=%d err=%v", status, err)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 collaborator (access-filtered), got %d", len(out))
	}
	if out[0]["member_name"] != "公司创建者" {
		t.Fatalf("member_name=%v", out[0]["member_name"])
	}
	if out[0]["user"] != caller {
		t.Fatalf("user=%v", out[0]["user"])
	}
	wantAvatar := "/api/tenant/" + tenantID + "/accounts/members/cm1/avatar"
	if out[0]["member_avatar_url"] != wantAvatar {
		t.Fatalf("member_avatar_url=%v want %s", out[0]["member_avatar_url"], wantAvatar)
	}
	perms, _ := out[0]["permissions"].([]map[string]interface{})
	if len(perms) != 1 || perms[0]["workspace_id"] != wsID {
		t.Fatalf("permissions=%v", out[0]["permissions"])
	}
}

func TestEnrichWorkspaceAccessRows_UserAndGroup(t *testing.T) {
	setupTestDB(t)
	tenantID := "t1"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members/resolve", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "cm9", "user_id": "u1", "company_id": tenantID, "member_name": "Alice",
		})
	})
	mux.HandleFunc("/api/internal/tenant/groups/by-id", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "g1", "name": "Core"})
	})
	mux.HandleFunc("/api/internal/tenant/groups/members", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"user_ids": []string{"u2", "u3"}})
	})
	mux.HandleFunc("/api/accounts/users/", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "u1", "email": "alice@example.com",
			"login_methods": []map[string]interface{}{
				{"method_type": "email", "identifier": "alice@example.com", "is_verified": true},
			},
		})
	})
	mux.HandleFunc("/api/internal/tenant/companies/creator", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"found": true, "creator_id": "u1",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	oldTenant := cfg.TaskTenantURL
	oldAuth := cfg.TaskAuthURL
	cfg.TaskTenantURL = srv.URL
	cfg.TaskAuthURL = srv.URL
	t.Cleanup(func() {
		cfg.TaskTenantURL = oldTenant
		cfg.TaskAuthURL = oldAuth
	})

	rows := []map[string]interface{}{
		{"id": "a1", "workspace_id": "ws1", "user_id": "u1", "group_id": "", "permission": "admin"},
		{"id": "a2", "workspace_id": "ws1", "user_id": "", "group_id": "g1", "permission": "view"},
	}
	out, err := enrichWorkspaceAccessRows(tenantID, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	ui, _ := out[0]["user_info"].(map[string]interface{})
	if ui["member_name"] != "Alice" || ui["company_member_id"] != "cm9" {
		t.Fatalf("user_info=%v", ui)
	}
	if ui["email"] != "alice@example.com" {
		t.Fatalf("email=%v", ui["email"])
	}
	if ui["is_tenant"] != true {
		t.Fatalf("is_tenant=%v", ui["is_tenant"])
	}
	gi, _ := out[1]["group_info"].(map[string]interface{})
	if gi["name"] != "Core" || gi["memberCount"] != 2 {
		t.Fatalf("group_info=%v", gi)
	}
}

func TestHandleWorkspaceCollaborators_HTTP(t *testing.T) {
	setupTestDB(t)
	tenantID := "t-http"
	wsID := "ws-http"
	userID := "u-http"
	_, _ = db.Exec(`INSERT INTO project_workspace_entries(id,company_id,name) VALUES(?,?,?)`, wsID, tenantID, "ws")
	_, _ = db.Exec(
		`INSERT INTO project_workspace_accesses(id,workspace_id,user_id,group_id,permission) VALUES(?,?,?,?,?)`,
		"wa-http", wsID, userID, "", "admin",
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/internal/tenant/members/resolve", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "cm", "user_id": userID, "company_id": tenantID, "member_name": "Bob",
		})
	})
	mux.HandleFunc("/api/internal/tenant/members", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "cm", "user_id": userID, "company_id": tenantID, "member_name": "Bob", "is_admin": true},
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	old := cfg.TaskTenantURL
	cfg.TaskTenantURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantURL = old })

	req := httptest.NewRequest(http.MethodGet,
		"/api/tenant/"+tenantID+"/projects/workspace-access/workspace-collaborators/?workspace_id="+wsID, nil)
	req.Header.Set("X-Auth-User-Id", userID)
	rr := httptest.NewRecorder()
	handleProjectsWorkspaceAccess(rr, req, tenantID, []string{"workspace-collaborators"})
	if rr.Code != 200 {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Bob") {
		t.Fatalf("body=%s", rr.Body.String())
	}
}
