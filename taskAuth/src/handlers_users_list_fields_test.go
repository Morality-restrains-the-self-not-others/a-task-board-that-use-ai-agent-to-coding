package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// OPT-20260807-009: 内部列表端点 GET /api/internal/users/ 须同步补齐
// email/phone/username 顶层字段（此前仅 login_methods），与系统管理列表一致。
func TestInternalListUsersIncludesTopLevelFields(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"

	userID, _, err := createUserWithEmailLogin("internal-list@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/internal/users/", nil)
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleListUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Users []map[string]interface{} `json:"users"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	var found *map[string]interface{}
	for i := range resp.Users {
		if resp.Users[i]["id"] == userID {
			found = &resp.Users[i]
			break
		}
	}
	if found == nil {
		t.Fatal("created user not found in internal list response")
	}
	if email, _ := (*found)["email"].(string); email != "internal-list@test.com" {
		t.Fatalf("expected email %q in internal list response, got %q", "internal-list@test.com", email)
	}
	for _, key := range []string{"phone", "username"} {
		if _, ok := (*found)[key]; !ok {
			t.Fatalf("expected top-level %q key in internal list response", key)
		}
	}
}

// OPT-20260807-010: 系统管理列表 username 统一取自 auth_user_profile.username
// （与 buildUserDetailJSON 一致），而非仅 username 登录方式 identifier。
func TestSystemAdminListUsernamePrefersProfile(t *testing.T) {
	setupAuthTestDB(t)

	userID, _, err := createUserWithEmailLogin("profile-uname@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := upsertUserProfile(userID, "profile-nick-1"); err != nil {
		t.Fatalf("set profile username: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/users/", nil)
	req.Header.Set("X-User-Id", "bootstrap-admin")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Users []map[string]interface{} `json:"users"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	var found *map[string]interface{}
	for i := range resp.Users {
		if resp.Users[i]["id"] == userID {
			found = &resp.Users[i]
			break
		}
	}
	if found == nil {
		t.Fatal("created user not found in system-admin list response")
	}
	if username, _ := (*found)["username"].(string); username != "profile-nick-1" {
		t.Fatalf("expected username from auth_user_profile %q, got %q", "profile-nick-1", username)
	}
}
