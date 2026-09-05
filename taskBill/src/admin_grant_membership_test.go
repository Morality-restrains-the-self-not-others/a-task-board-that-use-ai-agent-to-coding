package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func membershipTierAndLock(t *testing.T, tenantID int64) (tier string, locked int64) {
	t.Helper()
	if err := db.QueryRow(
		`SELECT tier, admin_tier_locked FROM billing_membership WHERE tenant_id = ?`,
		tenantID,
	).Scan(&tier, &locked); err != nil {
		t.Fatalf("query membership: %v", err)
	}
	return tier, locked
}

func insertConsumptionCents(t *testing.T, tenantID, cents int64) {
	t.Helper()
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("billing account: %v", err)
	}
	now := utcNow()
	_, err = db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, description, transaction_id, created_at, usage_amount
		) VALUES (?, ?, 'consumption', ?, 0, 0, 'test', 'test consumption', ?, ?, 0)`,
		generateSnowflakeID(), acc.ID, cents, formatID(generateSnowflakeID()), now,
	)
	if err != nil {
		t.Fatalf("insert consumption: %v", err)
	}
}

func TestAdminGrantMembershipSetsVip1AndLocks(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9300000001)
	result, err := adminGrantResourcesWithMembership(context.Background(), tenantID, nil, "admin-1", "", MembershipTierVIP1, "开通购买")
	if err != nil {
		t.Fatalf("grant membership: %v", err)
	}
	tier, locked := membershipTierAndLock(t, tenantID)
	if tier != MembershipTierVIP1 || locked != 1 {
		t.Fatalf("tier=%s locked=%d, want vip1 locked", tier, locked)
	}
	mem, _ := result["membership"].(map[string]interface{})
	if mem["tier"] != MembershipTierVIP1 {
		t.Fatalf("result membership=%v", result["membership"])
	}
}

func TestAdminGrantMembershipRejectsInvalidTier(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9300000002)
	_, err := adminGrantResourcesWithMembership(context.Background(), tenantID, nil, "admin-1", "", "vip2", "")
	if err == nil {
		t.Fatal("expected invalid tier error")
	}
	if !strings.Contains(err.Error(), "invalid membership tier") {
		t.Fatalf("got %v", err)
	}
}

func TestAdminGrantMembershipOnlyNoOrder(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9300000003)
	if _, err := adminGrantResourcesWithMembership(context.Background(), tenantID, nil, "admin-1", "", MembershipTierVIP1, ""); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM billing_resource_order WHERE tenant_id = ? AND payment_method = 'admin_grant'`,
		tenantID,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("VIP-only must not create grant order, count=%d", n)
	}
}

func TestAdminGrantResourcesLeavesMembershipUnchanged(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9300000004)
	if _, err := ensureMembership(tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := adminGrantResources(context.Background(), tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 3, Reason: "帖"},
	}, "admin-1", ""); err != nil {
		t.Fatal(err)
	}
	m, err := getMembership(tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("membership missing")
	}
	if m.Tier != MembershipTierNormal || m.AdminTierLocked {
		t.Fatalf("tier=%s locked=%v, want normal unlocked", m.Tier, m.AdminTierLocked)
	}
}

func TestSyncMembershipConsumptionSkipsWhenAdminLocked(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9300000005)
	if _, err := adminGrantResourcesWithMembership(context.Background(), tenantID, nil, "admin-1", "", MembershipTierNormal, "纠错"); err != nil {
		t.Fatal(err)
	}
	insertConsumptionCents(t, tenantID, VIP1AutoUpgradeThresholdCents)
	if _, err := syncMembershipConsumption(context.Background(), tenantID); err != nil {
		t.Fatal(err)
	}
	tier, locked := membershipTierAndLock(t, tenantID)
	if tier != MembershipTierNormal || locked != 1 {
		t.Fatalf("locked tenant must stay normal, tier=%s locked=%d", tier, locked)
	}
}

func TestSyncMembershipConsumptionUpgradesUnlockedNormal(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID = int64(9300000006)
	if _, err := ensureMembership(tenantID); err != nil {
		t.Fatal(err)
	}
	insertConsumptionCents(t, tenantID, VIP1AutoUpgradeThresholdCents)
	m, err := syncMembershipConsumption(context.Background(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Tier != MembershipTierVIP1 {
		t.Fatalf("unlocked normal should auto-upgrade, tier=%s", m.Tier)
	}
}

func TestHandleAdminGrantResourcesParsesMembershipTier(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid = int64(9300000007)
	body := `{"resources":[],"membership_tier":"vip1","membership_reason":"http"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/9300000007/billing/accounts/admin_grant_points/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	tier, locked := membershipTierAndLock(t, tid)
	if tier != MembershipTierVIP1 || locked != 1 {
		t.Fatalf("tier=%s locked=%d", tier, locked)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	mem, _ := parsed["membership"].(map[string]interface{})
	if mem["tier"] != MembershipTierVIP1 {
		t.Fatalf("body membership=%v", parsed["membership"])
	}
}

func TestHandleAdminGrantResourcesReadsIdempotencyKeyHeader(t *testing.T) {
	mux, cleanup := setupTestMux(t)
	defer cleanup()
	const tid = int64(9300000008)
	body := `{"resources":[],"membership_tier":"vip1","membership_reason":"http-ik-header"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tenant/9300000008/billing/accounts/admin_grant_points/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "header-ik-9300000008")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first code=%d body=%s", rec.Code, rec.Body.String())
	}
	tier, locked := membershipTierAndLock(t, tid)
	if tier != MembershipTierVIP1 || locked != 1 {
		t.Fatalf("tier=%s locked=%d", tier, locked)
	}

	// 同一 header 幂等键重放：不得重复赠送/改级（body 已被首次请求消费，须新建请求）
	req2 := httptest.NewRequest(http.MethodPost, "/api/tenant/9300000008/billing/accounts/admin_grant_points/", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "header-ik-9300000008")
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("second code=%d body=%s", rec2.Code, rec2.Body.String())
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(rec2.Body.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["idempotent"] != true {
		t.Fatalf("second response idempotent=%v body=%s", parsed["idempotent"], rec2.Body.String())
	}
}
