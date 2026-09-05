package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEnsureDefaultGitIdentity_CreateAndIdempotent(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	body := `{"user_id":"u-ens","company_id":"c-ens","member_id":"m-ens","member_name":"Ens User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/git-identities/ensure-default/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Id", "internal")
	req.Header.Set("X-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalEnsureDefaultGitIdentity(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first create: want 201 got %d body=%s", rec.Code, rec.Body.String())
	}
	var first map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if first["created"] != true {
		t.Fatalf("expected created=true: %+v", first)
	}
	ident, _ := first["identity"].(map[string]interface{})
	email, _ := ident["git_user_email"].(string)
	wantEmail := BuildSystemGitEmail("m-ens", "c-ens")
	if email != wantEmail {
		t.Fatalf("email=%q want %q", email, wantEmail)
	}
	if name, _ := ident["git_user_name"].(string); name != "Ens User" {
		t.Fatalf("name=%q", name)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/internal/git-identities/ensure-default/", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Auth-User-Id", "internal")
	rec2 := httptest.NewRecorder()
	handleInternalEnsureDefaultGitIdentity(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("idempotent: want 200 got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var second map[string]interface{}
	_ = json.Unmarshal(rec2.Body.Bytes(), &second)
	if second["created"] != false {
		t.Fatalf("expected created=false: %+v", second)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(1) FROM task_git_identities WHERE user_id=? AND company_id=?`, "u-ens", "c-ens").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row, got %d", n)
	}
}

func TestInternalLookupGitIdentity(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	if _, err := db.Exec(`
		INSERT INTO task_git_identities (id, user_id, company_id, label, git_user_name, git_user_email, is_default, created_at, updated_at)
		VALUES ('gid-lookup', 'u-lookup', 'c-lookup', '', 'Ljy', 'ljy@example.com', 0, NOW(), NOW())`); err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	post := func(t *testing.T, body string) (*httptest.ResponseRecorder, bool) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/git-identities/lookup/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Secret", "test-secret")
		rec := httptest.NewRecorder()
		handleInternalLookupGitIdentity(rec, req)
		if rec.Code != http.StatusOK {
			return rec, false
		}
		return rec, true
	}

	t.Run("found", func(t *testing.T) {
		rec, ok := post(t, `{"identity_id":"gid-lookup","user_id":"u-lookup","company_id":"c-lookup"}`)
		if !ok {
			t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
		}
		var out struct {
			Found        bool   `json:"found"`
			GitUserName  string `json:"git_user_name"`
			GitUserEmail string `json:"git_user_email"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if !out.Found || out.GitUserName != "Ljy" || out.GitUserEmail != "ljy@example.com" {
			t.Fatalf("out=%+v", out)
		}
	})

	t.Run("user_id mismatch returns not found", func(t *testing.T) {
		rec, ok := post(t, `{"identity_id":"gid-lookup","user_id":"other","company_id":"c-lookup"}`)
		if !ok {
			t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
		}
		var out struct {
			Found bool `json:"found"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.Found {
			t.Fatal("expected found=false on user_id mismatch")
		}
	})

	t.Run("company_id mismatch returns not found", func(t *testing.T) {
		rec, ok := post(t, `{"identity_id":"gid-lookup","user_id":"u-lookup","company_id":"other-company"}`)
		if !ok {
			t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
		}
		var out struct {
			Found bool `json:"found"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.Found {
			t.Fatal("expected found=false on company_id mismatch")
		}
	})

	t.Run("unknown identity returns not found", func(t *testing.T) {
		rec, ok := post(t, `{"identity_id":"missing","user_id":"u-lookup","company_id":"c-lookup"}`)
		if !ok {
			t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
		}
		var out struct {
			Found bool `json:"found"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.Found {
			t.Fatal("expected found=false for unknown identity")
		}
	})

	t.Run("missing params returns 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/git-identities/lookup/", strings.NewReader(`{"user_id":"u-lookup"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Secret", "test-secret")
		rec := httptest.NewRecorder()
		handleInternalLookupGitIdentity(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("want 400 got %d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("forbidden without internal secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/git-identities/lookup/", strings.NewReader(`{"identity_id":"gid-lookup","user_id":"u-lookup"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleInternalLookupGitIdentity(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("want 403 got %d body=%s", rec.Code, rec.Body.String())
		}
	})
}

// TestInternalLookupGitIdentity_InternalUserBypassesEmptySecret 回归：
// 生产 conf shared.internalSecret 为空时，仅 X-Auth-User-Id=internal 必须放行
// （Cloud layer-git-push prepare 的服务间调用约定）。无该头仍 403，避免空 secret 把端口敞开放行。
func TestInternalLookupGitIdentity_InternalUserBypassesEmptySecret(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = ""
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	if _, err := db.Exec(`
		INSERT INTO task_git_identities (id, user_id, company_id, label, git_user_name, git_user_email, is_default, created_at, updated_at)
		VALUES ('gid-empty-sec', 'u-empty', 'c-empty', '', 'Ljy', 'ljy@example.com', 0, NOW(), NOW())`); err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	body := `{"identity_id":"gid-empty-sec","user_id":"u-empty","company_id":"c-empty"}`

	t.Run("forbidden without internal user when secret empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/git-identities/lookup/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleInternalLookupGitIdentity(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("want 403 got %d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "internal only") {
			t.Fatalf("body=%s", rec.Body.String())
		}
	})

	t.Run("ok with X-Auth-User-Id internal when secret empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/internal/git-identities/lookup/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Auth-User-Id", "internal")
		rec := httptest.NewRecorder()
		handleInternalLookupGitIdentity(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
		}
		var out struct {
			Found        bool   `json:"found"`
			GitUserName  string `json:"git_user_name"`
			GitUserEmail string `json:"git_user_email"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if !out.Found || out.GitUserName != "Ljy" || out.GitUserEmail != "ljy@example.com" {
			t.Fatalf("out=%+v", out)
		}
	})
}

func TestTenantMemberGitIdentities_ForbiddenWithoutPerm(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "m1", "user_id": "u-target", "company_id": "t1", "member_name": "T",
		})
	}))
	t.Cleanup(srv.Close)
	prevTenantURL := cfg.TaskTenantServiceURL
	cfg.TaskTenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantServiceURL = prevTenantURL })

	req := httptest.NewRequest(http.MethodGet, "/api/git-identities/tenant/t1/member/m1/", nil)
	req.Header.Set("X-Auth-User-Id", "u-other")
	// no member:manage / group-members:manage
	rec := httptest.NewRecorder()
	handleTenantMemberGitIdentities(rec, req, "t1", "m1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTenantMemberGitIdentities_SelfAllowed(t *testing.T) {
	setupTestDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "m1", "user_id": "u-self", "company_id": "t1", "member_name": "Self",
		})
	}))
	t.Cleanup(srv.Close)
	prevTenantURL := cfg.TaskTenantServiceURL
	cfg.TaskTenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantServiceURL = prevTenantURL })

	req := httptest.NewRequest(http.MethodGet, "/api/git-identities/tenant/t1/member/m1/", nil)
	req.Header.Set("X-Auth-User-Id", "u-self")
	rec := httptest.NewRecorder()
	handleTenantMemberGitIdentities(rec, req, "t1", "m1")
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTenantMemberGitIdentities_GroupAdminInScopeAllowed(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "user-in-admin-groups") {
			_ = json.NewEncoder(w).Encode(map[string]bool{"in_admin_group": true})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "m1", "user_id": "u-target", "company_id": "t1", "member_name": "T",
		})
	}))
	t.Cleanup(srv.Close)
	prevTenantURL := cfg.TaskTenantServiceURL
	cfg.TaskTenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantServiceURL = prevTenantURL })

	req := httptest.NewRequest(http.MethodGet, "/api/git-identities/tenant/t1/member/m1/", nil)
	req.Header.Set("X-Auth-User-Id", "u-admin")
	req.Header.Set("X-User-Id", "u-admin")
	req.Header.Set("X-Tenant-Perms", "t1:group-members:manage")
	rec := httptest.NewRecorder()
	handleTenantMemberGitIdentities(rec, req, "t1", "m1")
	if rec.Code != http.StatusOK {
		t.Fatalf("group-admin in-scope: want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTenantMemberGitIdentities_GroupAdminOutOfScopeDenied(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "user-in-admin-groups") {
			_ = json.NewEncoder(w).Encode(map[string]bool{"in_admin_group": false})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "m1", "user_id": "u-target", "company_id": "t1", "member_name": "T",
		})
	}))
	t.Cleanup(srv.Close)
	prevTenantURL := cfg.TaskTenantServiceURL
	cfg.TaskTenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantServiceURL = prevTenantURL })

	req := httptest.NewRequest(http.MethodGet, "/api/git-identities/tenant/t1/member/m1/", nil)
	req.Header.Set("X-Auth-User-Id", "u-admin")
	req.Header.Set("X-User-Id", "u-admin")
	req.Header.Set("X-Tenant-Perms", "t1:group-members:manage")
	rec := httptest.NewRecorder()
	handleTenantMemberGitIdentities(rec, req, "t1", "m1")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("group-admin out-of-scope: want 403 got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestTenantMemberGitIdentities_TenantAdminBypassesGroupBoundary(t *testing.T) {
	setupTestDB(t)
	prevSecret := cfg.InternalSecret
	cfg.InternalSecret = "test-secret"
	t.Cleanup(func() { cfg.InternalSecret = prevSecret })

	// user-in-admin-groups 返回 false，但 member:manage 应直接放行（任意组穿透）。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "user-in-admin-groups") {
			_ = json.NewEncoder(w).Encode(map[string]bool{"in_admin_group": false})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "m1", "user_id": "u-target", "company_id": "t1", "member_name": "T",
		})
	}))
	t.Cleanup(srv.Close)
	prevTenantURL := cfg.TaskTenantServiceURL
	cfg.TaskTenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TaskTenantServiceURL = prevTenantURL })

	req := httptest.NewRequest(http.MethodGet, "/api/git-identities/tenant/t1/member/m1/", nil)
	req.Header.Set("X-Auth-User-Id", "u-admin")
	req.Header.Set("X-User-Id", "u-admin")
	req.Header.Set("X-Tenant-Perms", "t1:member:manage")
	rec := httptest.NewRecorder()
	handleTenantMemberGitIdentities(rec, req, "t1", "m1")
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant_admin bypass: want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
}
