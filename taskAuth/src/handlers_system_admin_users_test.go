package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSystemAdminListUsersHydratesReferrerCode guards OPT-20260821-031: the
// admin users list must fill the 推荐人 column via the taskReferral internal
// batch lookup (best-effort), rendering the referrer's display name.
func TestSystemAdminListUsersHydratesReferrerCode(t *testing.T) {
	setupAuthTestDB(t)

	referrerID, _, err := createUserWithEmailLogin("referrer-031@test.com", "hash")
	if err != nil {
		t.Fatalf("create referrer: %v", err)
	}
	referredID, _, err := createUserWithEmailLogin("referred-031@test.com", "hash")
	if err != nil {
		t.Fatalf("create referred: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/referral/referrers/lookup/" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"referrers": map[string]map[string]string{
				referredID: {"referrer_user_id": referrerID, "channel_code": "DR2AKvP9J9"},
			},
		})
	}))
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

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
	var referred *map[string]interface{}
	for i := range resp.Users {
		if resp.Users[i]["id"] == referredID {
			referred = &resp.Users[i]
			break
		}
	}
	if referred == nil {
		t.Fatal("referred user not found in list response")
	}
	if code, _ := (*referred)["referrer_code"].(string); code != "referrer-031@test.com" {
		t.Fatalf("expected referrer_code %q, got %q", "referrer-031@test.com", code)
	}
}

// TestSystemAdminListUsersReferrerCodeEmptyWhenNoEdge ensures the 推荐人 column
// stays empty (not an error) when the referral service returns no edge.
func TestSystemAdminListUsersReferrerCodeEmptyWhenNoEdge(t *testing.T) {
	setupAuthTestDB(t)
	_, _, err := createUserWithEmailLogin("noedge-031@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"referrers": map[string]interface{}{}})
	}))
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

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
	if len(resp.Users) == 0 {
		t.Fatal("expected users in response")
	}
	for _, u := range resp.Users {
		if code, _ := u["referrer_code"].(string); code != "" {
			t.Fatalf("user %v unexpected referrer_code=%q", u["id"], code)
		}
	}
}

// TestSystemAdminListUsersIncludesEmailFields guards the admin users list
// response contract: each user must carry top-level email/phone/username
// lifted from login methods, so the frontend 邮箱 column renders the real
// address.
//
// Regression: the list previously returned only id + flags + login_methods,
// so UserListRow's email cell fell back to the raw user id — the seeded
// superadmin (id "bootstrap-admin", email author@example.com) displayed as
// "bootstrap-admin" instead of the email.
func TestSystemAdminListUsersIncludesEmailFields(t *testing.T) {
	setupAuthTestDB(t)

	// dataMigrate 022 seeds the bootstrap superadmin; act as that superuser.
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

	var bootstrapUser *map[string]interface{}
	for i := range resp.Users {
		if resp.Users[i]["id"] == "bootstrap-admin" {
			bootstrapUser = &resp.Users[i]
			break
		}
	}
	if bootstrapUser == nil {
		t.Fatal("bootstrap-admin user not found in list response")
	}

	wantEmail := mustConfAdminEmail(t)
	if email, _ := (*bootstrapUser)["email"].(string); email != wantEmail {
		t.Fatalf("expected email %q in list response, got %q", wantEmail, email)
	}
	for _, key := range []string{"phone", "username"} {
		if _, ok := (*bootstrapUser)[key]; !ok {
			t.Fatalf("expected top-level %q key in user response", key)
		}
	}
}

// TestSystemAdminListUsersEmailForRegularUser verifies the same lifting works
// for a normal registered user (email login method present).
func TestSystemAdminListUsersEmailForRegularUser(t *testing.T) {
	setupAuthTestDB(t)

	userID, _, err := createUserWithEmailLogin("regular-list@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
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
		t.Fatal("created user not found in list response")
	}
	if email, _ := (*found)["email"].(string); email != "regular-list@test.com" {
		t.Fatalf("expected email %q in list response, got %q", "regular-list@test.com", email)
	}
	if ll, _ := (*found)["last_login"].(string); ll != "" {
		t.Fatalf("expected empty last_login for never-logged-in user, got %q", ll)
	}
}

func TestSystemAdminListUsersReturnsLastLogin(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("last-login-list@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	const stamped = "2026-08-21 10:30:00.000000"
	if _, err := db.Exec(`UPDATE auth_user SET last_login = ? WHERE id = ?`, stamped, userID); err != nil {
		t.Fatalf("stamp last_login: %v", err)
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
		t.Fatal("created user not found in list response")
	}
	got, _ := (*found)["last_login"].(string)
	if !strings.Contains(got, "2026-08-21") {
		t.Fatalf("last_login=%q want to contain 2026-08-21", got)
	}
}

// TestSystemAdminListUsersHydratesTenantCompanies fills 所属租户公司 from
// taskTenantService members/batch-get (active memberships only).
func TestSystemAdminListUsersHydratesTenantCompanies(t *testing.T) {
	setupAuthTestDB(t)

	userID, _, err := createUserWithEmailLogin("tenant-col@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/tenant/members/batch-get/" && r.URL.Path != "/api/internal/tenant/members/batch-get" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"members": []map[string]interface{}{
				{"user_id": userID, "company_id": "c-zeta", "company_name": "Zeta", "is_active": true},
				{"user_id": userID, "company_id": "c-acme", "company_name": "Acme", "is_active": true},
				{"user_id": userID, "company_id": "c-dead", "company_name": "Dead", "is_active": false},
			},
		})
	}))
	defer srv.Close()
	prev := cfg.TenantServiceURL
	cfg.TenantServiceURL = srv.URL
	t.Cleanup(func() { cfg.TenantServiceURL = prev })

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
		t.Fatal("created user not found in list response")
	}
	raw, ok := (*found)["tenant_companies"].([]interface{})
	if !ok {
		t.Fatalf("tenant_companies type %T", (*found)["tenant_companies"])
	}
	if len(raw) != 2 {
		t.Fatalf("expected 2 active companies, got %v", raw)
	}
	first, _ := raw[0].(map[string]interface{})
	if first["id"] != "c-acme" || first["name"] != "Acme" {
		t.Fatalf("expected Acme first, got %v", first)
	}
}

func TestSystemAdminListUsersTenantCompaniesEmptyWhenDown(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("tenant-down@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	prev := cfg.TenantServiceURL
	cfg.TenantServiceURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.TenantServiceURL = prev })

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
		t.Fatalf("unmarshal: %v", err)
	}
	for _, u := range resp.Users {
		if u["id"] != userID {
			continue
		}
		raw, _ := u["tenant_companies"].([]interface{})
		if len(raw) != 0 {
			t.Fatalf("expected empty tenant_companies, got %v", raw)
		}
	}
}

func TestSystemAdminListUsersHydratesProfitSharingQualification(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("psq-col@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/referral/qualification/active/batch/" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"qualifications": map[string]bool{userID: true},
		})
	}))
	defer srv.Close()
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

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
		t.Fatalf("unmarshal: %v", err)
	}
	for _, u := range resp.Users {
		if u["id"] != userID {
			continue
		}
		if u["has_profit_sharing_qualification"] != true {
			t.Fatalf("expected true qualification, got %v", u["has_profit_sharing_qualification"])
		}
		return
	}
	t.Fatal("created user not found")
}

func TestSystemAdminListUsersQualificationNullWhenReferralDown(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("psq-down@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = "http://127.0.0.1:1"
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })

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
		t.Fatalf("unmarshal: %v", err)
	}
	for _, u := range resp.Users {
		if u["id"] != userID {
			continue
		}
		if u["has_profit_sharing_qualification"] != nil {
			t.Fatalf("down service must yield null, got %v", u["has_profit_sharing_qualification"])
		}
		return
	}
	t.Fatal("created user not found")
}
