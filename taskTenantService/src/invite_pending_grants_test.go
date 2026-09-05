package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseInviteGrants_OK(t *testing.T) {
	body := map[string]interface{}{
		"grants": []interface{}{
			map[string]interface{}{"group_key": "people.access", "effect": "operate"},
			map[string]interface{}{"group_key": "people.access.subject_list", "effect": "view"},
		},
	}
	grants, err := parseInviteGrants(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 2 {
		t.Fatalf("expected 2 grants, got %d", len(grants))
	}
}

func TestParseInviteGrants_Invalid(t *testing.T) {
	_, err := parseInviteGrants(map[string]interface{}{"grants": "x"})
	if err == nil {
		t.Fatal("expected error for non-array grants")
	}
	_, err = parseInviteGrants(map[string]interface{}{
		"grants": []interface{}{map[string]interface{}{"group_key": "", "effect": "operate"}},
	})
	if err == nil {
		t.Fatal("expected error for empty group_key")
	}
}

func TestInviteWithPendingGrantsStoresJSON(t *testing.T) {
	mux := setupTestService(t)
	ensurePendingGrantsColumn(t)

	body := `{"invite_method":"link","company_member_name":"Grantee","role":"member","workspace_id":"ws1","expiration_days":7,"grants":[{"group_key":"people.access","effect":"operate"},{"group_key":"people.access.save_actions","effect":"operate"}]}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var raw string
	err := db.QueryRow(`SELECT COALESCE(CAST(pending_grants AS CHAR),'') FROM tenant_invitation WHERE company_id='c1' AND company_member_name='Grantee' ORDER BY created_at DESC LIMIT 1`).Scan(&raw)
	if err != nil {
		t.Fatalf("query pending_grants: %v", err)
	}
	if !strings.Contains(raw, "people.access") {
		t.Fatalf("expected pending_grants to contain people.access, got %q", raw)
	}
}

func TestInviteAdminIgnoresPendingGrants(t *testing.T) {
	mux := setupTestService(t)
	ensurePendingGrantsColumn(t)

	body := `{"invite_method":"link","company_member_name":"AdminInvitee","role":"admin","workspace_id":"ws1","expiration_days":7,"grants":[{"group_key":"people.access","effect":"operate"}]}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var raw sql.NullString
	err := db.QueryRow(`SELECT CAST(pending_grants AS CHAR) FROM tenant_invitation WHERE company_id='c1' AND company_member_name='AdminInvitee' ORDER BY created_at DESC LIMIT 1`).Scan(&raw)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if raw.Valid && strings.TrimSpace(raw.String) != "" && raw.String != "null" {
		t.Fatalf("admin invite should ignore grants, got %v", raw.String)
	}
}

func TestJoinAppliesPendingGrantsViaAuth(t *testing.T) {
	mux := setupTestService(t)
	ensurePendingGrantsColumn(t)

	var applied bool
	authSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/internal/authz/apply-member-grants/" {
			http.NotFound(w, r)
			return
		}
		applied = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "role-x", "name": "custom_c1_test"})
	}))
	defer authSrv.Close()
	old := cfg.TaskAuthURL
	cfg.TaskAuthURL = authSrv.URL
	t.Cleanup(func() { cfg.TaskAuthURL = old })

	body := `{"invite_method":"link","company_member_name":"JoinGrantee","role":"member","workspace_id":"ws1","expiration_days":7,"grants":[{"group_key":"people.access","effect":"operate"}]}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/invite/", bytes.NewBufferString(body)), "admin1")
	rec := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec, req)
	var inviteOut map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &inviteOut)
	token, _ := inviteOut["invite_token"].(string)
	if token == "" {
		t.Fatalf("no token: %s", rec.Body.String())
	}

	joinBody := `{"token":"` + token + `"}`
	req3 := withUser(httptest.NewRequest(http.MethodPost, "/api/tenant/c1/accounts/members/join/", bytes.NewBufferString(joinBody)), "joingrantee1")
	rec3 := httptest.NewRecorder()
	gatewayUserMiddleware(mux).ServeHTTP(rec3, req3)
	if rec3.Code != 201 {
		t.Fatalf("join expected 201, got %d body=%s", rec3.Code, rec3.Body.String())
	}
	if !applied {
		t.Fatal("expected apply-member-grants to be called")
	}
	var roleName string
	err := db.QueryRow(`SELECT role_name FROM tenant_member_role WHERE company_id='c1' AND member_id=(
		SELECT id FROM tenant_company_member WHERE user_id='joingrantee1' AND company_id='c1' LIMIT 1)`).Scan(&roleName)
	if err != nil {
		t.Fatalf("member role: %v", err)
	}
	if roleName != "custom_c1_test" {
		t.Fatalf("expected custom role, got %q", roleName)
	}
}
