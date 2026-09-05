package main

import (
	"context"
	"strings"
	"testing"
)

func TestMigratedGitlabDiskUnitPriceIsFourYuan(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	var price int64
	if err := db.QueryRow(`SELECT price FROM billing_unit WHERE unit_type = 'gitlab_disk'`).Scan(&price); err != nil {
		t.Fatalf("read gitlab_disk: %v", err)
	}
	if price != DefaultGitlabDiskUnitPriceCents {
		t.Fatalf("gitlab_disk.price=%d want %d", price, DefaultGitlabDiskUnitPriceCents)
	}

	got, err := getUnitPriceCents(ResourceTypeGitlabDisk)
	if err != nil {
		t.Fatalf("getUnitPriceCents: %v", err)
	}
	if got != DefaultGitlabDiskUnitPriceCents {
		t.Fatalf("getUnitPriceCents=%d want %d", got, DefaultGitlabDiskUnitPriceCents)
	}
}

func TestGetUnitPriceCentsFallsBackToGitlabDiskDefaultWhenMissing(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	if _, err := db.Exec(`DELETE FROM billing_unit WHERE unit_type = 'gitlab_disk'`); err != nil {
		t.Fatalf("delete gitlab_disk: %v", err)
	}

	got, err := getUnitPriceCents(ResourceTypeGitlabDisk)
	if err != nil {
		t.Fatalf("getUnitPriceCents: %v", err)
	}
	if got != DefaultGitlabDiskUnitPriceCents {
		t.Fatalf("fallback getUnitPriceCents=%d want %d", got, DefaultGitlabDiskUnitPriceCents)
	}

	rp, err := getCurrentResourcePricing()
	if err != nil {
		t.Fatalf("getCurrentResourcePricing: %v", err)
	}
	if rp.GitlabDiskUnitPriceCents != DefaultGitlabDiskUnitPriceCents {
		t.Fatalf("fallback GitlabDiskUnitPriceCents=%d want %d", rp.GitlabDiskUnitPriceCents, DefaultGitlabDiskUnitPriceCents)
	}
}

func TestCreateOrderGitlabDiskRejectsBelowMinPurchase(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000010
	seedVIP1Membership(t, tenantID)
	_, _, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB - 1, Region: "tencent-sh-1", DiskMonths: 1},
	})
	if err == nil {
		t.Fatal("expected error for 9 GB")
	}
	if !strings.Contains(err.Error(), "起购") || !strings.Contains(err.Error(), "10") {
		t.Fatalf("got %v", err)
	}
}

func TestCreateOrderGitlabDiskAllowsMinPurchase(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000011
	seedVIP1Membership(t, tenantID)
	order, items, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if order == nil || len(items) != 1 || items[0].Quantity != GitlabDiskMinPurchaseGB {
		t.Fatalf("items=%+v", items)
	}
	wantSub := DefaultGitlabDiskUnitPriceCents * GitlabDiskMinPurchaseGB
	if items[0].SubtotalYuanCents != wantSub {
		t.Fatalf("subtotal=%d want %d", items[0].SubtotalYuanCents, wantSub)
	}
}

func TestMarkOrderPaidGitlabDiskAllowsTenGB(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9400000012
	seedVIP1Membership(t, tenantID)
	order, _, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(context.Background(), order.ID, "wechat", "mock-disk-10gb", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}
	var diskGB int64
	if err := db.QueryRow(
		`SELECT disk_gb FROM billing_tenant_gitlab_resource WHERE tenant_id = ? AND region = ?`,
		tenantID, "tencent-sh-1",
	).Scan(&diskGB); err != nil {
		t.Fatalf("scan disk_gb: %v", err)
	}
	if diskGB < GitlabDiskMinPurchaseGB {
		t.Fatalf("disk_gb=%d want >= %d", diskGB, GitlabDiskMinPurchaseGB)
	}
}
