package main

import (
	"context"
	"testing"
)

func TestCheckBalanceForServerStart_DoesNotRequireWallet(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000102
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if err := checkBalanceForServerStart(tenantID); err != nil {
		t.Fatalf("zero wallet must not block server start: %v", err)
	}
}

func TestChargeGitlabTraffic_DoesNotDebitWallet(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000103
	const startBalance int64 = 50
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, generateSnowflakeID(), tenantID, startBalance, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_tenant_gitlab_resource (
			tenant_id, region, disk_gb, traffic_prepaid_gb, disk_months, disk_expires_at,
			disk_used_bytes, traffic_used_gb, provisioning_status, created_at, updated_at
		) VALUES (?, ?, 1, 10, 1, '', 0, 0, 'active', ?, ?)`,
		tenantID, defaultGitlabRegion, now, now); err != nil {
		t.Fatalf("seed quota: %v", err)
	}
	result, err := chargeGitlabTraffic(context.Background(), tenantID, 1, false, "ut-traffic-no-wallet", "", "", "", "", defaultGitlabRegion)
	if err != nil {
		t.Fatalf("chargeGitlabTraffic: %v", err)
	}
	if skipped, _ := result["skipped"].(bool); skipped {
		t.Fatalf("expected billed usage, got skipped=%v result=%v", skipped, result)
	}
	var balance int64
	if err := db.QueryRow(`SELECT balance FROM billing_account WHERE tenant_id = ?`, tenantID).Scan(&balance); err != nil {
		t.Fatalf("query balance: %v", err)
	}
	if balance != startBalance {
		t.Fatalf("balance %d → %d; traffic metering must not debit wallet", startBalance, balance)
	}
}

func TestChargeGitlabTraffic_RejectsWhenNotPurchased(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9300000104
	now := utcNow()
	if _, err := db.Exec(`
		INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?)`, generateSnowflakeID(), tenantID, now, now); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	_, err := chargeGitlabTraffic(context.Background(), tenantID, 1, false, "ut-traffic-no-prepaid", "", "", "", "", "tencent-sh-1")
	if err == nil {
		t.Fatal("expected traffic quota error")
	}
	tq, ok := err.(*TrafficQuotaExceededError)
	if !ok {
		t.Fatalf("err type %T: %v", err, err)
	}
	if tq.Code != trafficGateNotPurchased {
		t.Fatalf("code=%s", tq.Code)
	}
}
