package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// OPT-20260824-XXX: admin 推荐码反查 —— 通过推荐码反向找到持有该码的具体用户。
// 覆盖: 命中(active)/未命中/停用码/空参数/非 superuser 拒绝。

func seedShareCodeForLookup(t *testing.T, code, userID, channelName string, isDefault int, status string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO referral_share_code (code, user_id, channel_name, is_default, status, created_at)
		VALUES (?, ?, ?, ?, ?, NOW())`,
		code, userID, channelName, isDefault, status)
	if err != nil {
		t.Fatalf("insert share code %s: %v", code, err)
	}
}

func seedUserProfileForLookup(t *testing.T, userID, username string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO auth_user_profile (user_id, username) VALUES (?, ?)`, userID, username)
	if err != nil {
		t.Fatalf("insert auth_user_profile %s: %v", userID, err)
	}
}

// setupShareCodeLookupTestDB 在 setupTestMux 基础上补充 auth_user_profile 表
//（反查「具体的人」需要用户名）。
func setupShareCodeLookupTestDB(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := setupTestMux(t)
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS auth_user_profile (
			user_id VARCHAR(255) NOT NULL PRIMARY KEY,
			username VARCHAR(255) NOT NULL DEFAULT ''
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil {
		t.Fatalf("create auth_user_profile: %v", err)
	}
	// 清理，保证用例间无残留
	if _, err := db.Exec(`DELETE FROM referral_share_code`); err != nil {
		t.Fatalf("clean referral_share_code: %v", err)
	}
	return mux
}

func TestAdminShareCodeLookupFound(t *testing.T) {
	mux := setupShareCodeLookupTestDB(t)
	seedShareCodeForLookup(t, "DR2AKvP9J9", "ref-owner-1", "默认", 1, "active")
	seedUserProfileForLookup(t, "ref-owner-1", "赖金燕")

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/share-code/lookup/?code=DR2AKvP9J9", nil)
	req.Header.Set("X-User-Id", "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if found, _ := resp["found"].(bool); !found {
		t.Fatalf("expected found=true, got %v", resp)
	}
	if v := resp["user_id"]; v != "ref-owner-1" {
		t.Fatalf("user_id=%v", v)
	}
	if v := resp["username"]; v != "赖金燕" {
		t.Fatalf("username=%v", v)
	}
	if v := resp["channel_name"]; v != "默认" {
		t.Fatalf("channel_name=%v", v)
	}
	if v := resp["status"]; v != "active" {
		t.Fatalf("status=%v", v)
	}
	if isDefault, _ := resp["is_default"].(bool); !isDefault {
		t.Fatalf("is_default=%v", resp["is_default"])
	}
}

func TestAdminShareCodeLookupNotFound(t *testing.T) {
	mux := setupShareCodeLookupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/share-code/lookup/?code=NO_SUCH_CODE", nil)
	req.Header.Set("X-User-Id", "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if found, _ := resp["found"].(bool); found {
		t.Fatalf("expected found=false, got %v", resp)
	}
}

func TestAdminShareCodeLookupDisabledCode(t *testing.T) {
	mux := setupShareCodeLookupTestDB(t)
	seedShareCodeForLookup(t, "DISABLED01", "ref-owner-2", "默认", 1, "disabled")

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/share-code/lookup/?code=DISABLED01", nil)
	req.Header.Set("X-User-Id", "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if found, _ := resp["found"].(bool); found {
		t.Fatalf("disabled code must not resolve, got %v", resp)
	}
}

func TestAdminShareCodeLookupEmptyCode(t *testing.T) {
	mux := setupShareCodeLookupTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/share-code/lookup/", nil)
	req.Header.Set("X-User-Id", "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty code, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminShareCodeLookupForbiddenNonSuperuser(t *testing.T) {
	mux := setupShareCodeLookupTestDB(t)
	seedShareCodeForLookup(t, "DR2AKvP9J9", "ref-owner-1", "默认", 1, "active")

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/share-code/lookup/?code=DR2AKvP9J9", nil)
	req.Header.Set("X-User-Id", "regular-user-1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-superuser, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLookupShareCodeOwnerDetailUsernameMissing(t *testing.T) {
	mux := setupShareCodeLookupTestDB(t)
	seedShareCodeForLookup(t, "NO_PROFILE01", "ref-owner-3", "默认", 1, "active")

	req := httptest.NewRequest(http.MethodGet, "/api/system-admin/referral/share-code/lookup/?code=NO_PROFILE01", nil)
	req.Header.Set("X-User-Id", "super1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"found":true`) || !strings.Contains(body, `"username":""`) {
		t.Fatalf("user without profile must resolve with empty username, got %s", body)
	}
}
