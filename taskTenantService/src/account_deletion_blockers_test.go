package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func seedPendingInvitation(t *testing.T, id string, expiresSQL string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO tenant_invitation
		(id, company_id, is_admin, workspace_id, invite_method, invite_target,
		 invitation_token, invitation_token_expires_at, is_accepted, company_member_name)
		VALUES (?, 'c1', 0, 'ws1', 'email', 'guest@example.com', ?, `+expiresSQL+`, 0, 'Guest')`,
		id, "tok-"+id)
	if err != nil {
		t.Fatalf("seed invitation %s: %v", id, err)
	}
}

func blockerCodes(blockers []tenantDeletionBlocker) []string {
	codes := make([]string, 0, len(blockers))
	for _, b := range blockers {
		codes = append(codes, b.Code)
	}
	return codes
}

func TestCollectTenantDeletionBlockers_PendingInviteDoesNotBlock(t *testing.T) {
	setupTestService(t)
	seedPendingInvitation(t, "inv-live", "DATE_ADD(NOW(), INTERVAL 7 DAY)")

	blockers, err := collectTenantDeletionBlockers("admin1")
	if err != nil {
		t.Fatalf("collectTenantDeletionBlockers: %v", err)
	}
	for _, b := range blockers {
		if b.Code == "TENANT_INVITE_PENDING" {
			t.Fatalf("pending member invites must not block account deletion, got %+v codes=%v", b, blockerCodes(blockers))
		}
	}
}

func TestCollectTenantDeletionBlockers_ExpiredInviteDoesNotBlock(t *testing.T) {
	setupTestService(t)
	seedPendingInvitation(t, "inv-expired", "DATE_SUB(NOW(), INTERVAL 1 DAY)")

	blockers, err := collectTenantDeletionBlockers("admin1")
	if err != nil {
		t.Fatalf("collectTenantDeletionBlockers: %v", err)
	}
	for _, b := range blockers {
		if b.Code == "TENANT_INVITE_PENDING" {
			t.Fatalf("expired invites must not block account deletion, got %+v", b)
		}
	}
}

func TestCollectTenantDeletionBlockers_SoleAdminWithPeersStillBlocks(t *testing.T) {
	setupTestService(t)
	_, err := db.Exec(`INSERT INTO tenant_company_member
		(id, user_id, company_id, is_admin, is_active, workspace_id, member_name)
		VALUES ('m2','member1','c1',0,1,'ws1','Member')`)
	if err != nil {
		t.Fatalf("seed peer member: %v", err)
	}

	blockers, err := collectTenantDeletionBlockers("admin1")
	if err != nil {
		t.Fatalf("collectTenantDeletionBlockers: %v", err)
	}
	found := false
	for _, b := range blockers {
		if b.Code == "TENANT_SOLE_ADMIN" && b.Blocking {
			found = true
			if b.ActionURL != tenantMembersActionURL("c1") {
				t.Fatalf("TENANT_SOLE_ADMIN action_url=%q want people/manage", b.ActionURL)
			}
		}
		if b.Code == "TENANT_INVITE_PENDING" {
			t.Fatalf("invite pending must not appear even when sole-admin blocks, got %+v", b)
		}
	}
	if !found {
		t.Fatalf("expected TENANT_SOLE_ADMIN when unique admin has other members, codes=%v", blockerCodes(blockers))
	}
}

func TestInternalAccountDeletionBlockersOmitsInvitePending(t *testing.T) {
	mux := setupTestService(t)
	seedPendingInvitation(t, "inv-http", "DATE_ADD(NOW(), INTERVAL 7 DAY)")

	req := httptest.NewRequest(http.MethodGet, "/api/internal/tenant/users/admin1/account-deletion-blockers/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Blockers []tenantDeletionBlocker `json:"blockers"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	for _, b := range payload.Blockers {
		if b.Code == "TENANT_INVITE_PENDING" {
			t.Fatalf("HTTP precheck must not return TENANT_INVITE_PENDING, got %+v", b)
		}
	}
}
