package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// seedInviteCleanupTenant seeds a tenant with one active member and one pending
// (is_accepted=0, unexpired) invitation for that tenant.
func seedInviteCleanupTenant(t *testing.T, tenantID, userID, memberID, inviteID, token string) {
	t.Helper()
	mustExec(t, `INSERT INTO tenant_company (id, name, creator_id) VALUES (?, 'InviteCleanup Co', ?)`, tenantID, userID)
	mustExec(t, `INSERT INTO tenant_company_member (id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES (?,?,?,1,1,'ws-clean','Owner')`, memberID, userID, tenantID)
	mustExec(t, `INSERT INTO tenant_invitation
		(id, company_id, is_admin, workspace_id, invite_method, invite_target, invitation_token, invitation_token_expires_at, is_accepted, company_member_name)
		VALUES (?,?,0,'ws-clean','link','target',?, DATE_ADD(NOW(), INTERVAL 7 DAY), 0, 'Invited')`,
		inviteID, tenantID, token)
}

func mustExec(t *testing.T, q string, args ...interface{}) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

func inviteExpired(t *testing.T, inviteID string) bool {
	t.Helper()
	var expired int
	if err := db.QueryRow(
		`SELECT COUNT(1) FROM tenant_invitation WHERE id=? AND invitation_token_expires_at < NOW()`,
		inviteID).Scan(&expired); err != nil {
		t.Fatalf("query invite expiry: %v", err)
	}
	return expired == 1
}

func TestCleanupStaleInvitesForUserInvalidatesWhenSoleMember(t *testing.T) {
	setupTestService(t)
	seedInviteCleanupTenant(t, "t-clean-1", "u-clean-1", "m-clean-1", "i-clean-1", "tok-clean-1")

	n, err := cleanupStaleInvitesForUser("u-clean-1")
	if err != nil {
		t.Fatalf("cleanupStaleInvitesForUser: %v", err)
	}
	if n != 1 {
		t.Fatalf("invalidated=%d want 1", n)
	}
	if !inviteExpired(t, "i-clean-1") {
		t.Fatal("expected pending invite to be invalidated (expires in the past)")
	}
}

func TestCleanupStaleInvitesKeepsWhenOtherActiveMember(t *testing.T) {
	setupTestService(t)
	seedInviteCleanupTenant(t, "t-clean-2", "u-clean-2", "m-clean-2", "i-clean-2", "tok-clean-2")
	// Second active member in the same tenant.
	mustExec(t, `INSERT INTO tenant_company_member (id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m-clean-2b','u-clean-2b','t-clean-2',0,1,'ws-clean','Coworker')`)

	n, err := cleanupStaleInvitesForUser("u-clean-2")
	if err != nil {
		t.Fatalf("cleanupStaleInvitesForUser: %v", err)
	}
	if n != 0 {
		t.Fatalf("invalidated=%d want 0", n)
	}
	if inviteExpired(t, "i-clean-2") {
		t.Fatal("expected invite to stay valid while tenant has another active member")
	}
}

func TestCleanupStaleInvitesInvalidatesWhenMembershipInactive(t *testing.T) {
	setupTestService(t)
	seedInviteCleanupTenant(t, "t-clean-3", "u-clean-3", "m-clean-3", "i-clean-3", "tok-clean-3")
	// Even if the deleting user's membership row was deactivated, the tenant has no
	// remaining active members, so pending invites must still be invalidated.
	mustExec(t, `UPDATE tenant_company_member SET is_active=0 WHERE id='m-clean-3'`)

	n, err := cleanupStaleInvitesForUser("u-clean-3")
	if err != nil {
		t.Fatalf("cleanupStaleInvitesForUser: %v", err)
	}
	if n != 1 {
		t.Fatalf("invalidated=%d want 1", n)
	}
	if !inviteExpired(t, "i-clean-3") {
		t.Fatal("expected pending invite to be invalidated when no active members remain")
	}
}

func TestInternalCleanupStaleInvitesHandler(t *testing.T) {
	setupTestService(t)
	seedInviteCleanupTenant(t, "t-clean-4", "u-clean-4", "m-clean-4", "i-clean-4", "tok-clean-4")

	// GET is not allowed.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/account-deletion/cleanup-invites/", nil)
	router := http.NewServeMux()
	router.HandleFunc("/api/internal/tenant/account-deletion/cleanup-invites/", handleInternalCleanupStaleInvites)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d want 405", rec.Code)
	}

	// Missing user_id → 400.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/internal/tenant/account-deletion/cleanup-invites/", bytes.NewBufferString(`{}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing user_id status=%d want 400", rec.Code)
	}

	// Valid POST → 200 with invalidated count.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/internal/tenant/account-deletion/cleanup-invites/", bytes.NewBufferString(`{"user_id":"u-clean-4"}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST status=%d want 200 body=%s", rec.Code, rec.Body.String())
	}
	if !inviteExpired(t, "i-clean-4") {
		t.Fatal("expected handler to invalidate the pending invite")
	}
}
