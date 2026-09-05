package main

import (
	"context"
	"strings"
	"testing"
)

func seedVIP1Membership(t *testing.T, tenantID int64) {
	t.Helper()
	now := utcNow()
	_, err := db.Exec(
		`INSERT INTO billing_membership (id, tenant_id, tier, cumulative_consumption_cents, created_at, updated_at)
		 VALUES (?, ?, 'vip1', 10000, ?, ?)`,
		generateSnowflakeID(), tenantID, now, now,
	)
	if err != nil {
		t.Fatalf("seed vip1: %v", err)
	}
}

func TestCreateOrder_GitlabDiskUnknownRegion(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9200000001
	seedVIP1Membership(t, tenantID)
	_, _, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: 1, Region: "no-such-region"},
	})
	if err == nil {
		t.Fatal("expected error for unknown region")
	}
	if !strings.Contains(err.Error(), "region not found") {
		t.Fatalf("got %v", err)
	}
}

func TestCreateOrder_GitlabDiskWritesSelectedRegion(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9200000002
	seedVIP1Membership(t, tenantID)
	order, items, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if len(items) != 1 || items[0].Region != "tencent-sh-1" {
		t.Fatalf("items=%+v", items)
	}
	var stored string
	if err := db.QueryRow(
		`SELECT i.region FROM billing_resource_order_item i
		 JOIN billing_resource_order o ON o.id = i.order_id
		 WHERE o.tenant_id = ? AND i.resource_type = ?`,
		tenantID, ResourceTypeGitlabDisk,
	).Scan(&stored); err != nil {
		t.Fatalf("query item order_id=%d: %v", order.ID, err)
	}
	if stored != "tencent-sh-1" {
		t.Fatalf("stored region=%q", stored)
	}
}
