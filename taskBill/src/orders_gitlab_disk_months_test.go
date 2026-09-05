package main

import (
	"context"
	"strings"
	"testing"
)

// OPT-20260903-010: createOrder 计价须按购买月数（单价×GB×月数）而非只按 GB。

func TestCreateOrderGitlabDiskSubtotalMultipliesMonths(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9610000110
	seedVIP1Membership(t, tenantID)
	ctx := context.Background()

	order, items, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 3},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	it := items[0]
	wantSub := DefaultGitlabDiskUnitPriceCents * GitlabDiskMinPurchaseGB * 3
	if it.SubtotalYuanCents != wantSub {
		t.Fatalf("subtotal=%d want %d (price×GB×months)", it.SubtotalYuanCents, wantSub)
	}
	if it.DiskMonths != 3 {
		t.Fatalf("item.DiskMonths=%d want 3", it.DiskMonths)
	}
	if order.TotalYuanCents != wantSub {
		t.Fatalf("order.total=%d want %d", order.TotalYuanCents, wantSub)
	}

	// 落库后再加载：磁盘行须带回 disk_months
	loaded, loadedItems, err := loadOrder(tenantID, order.ID)
	if err != nil {
		t.Fatalf("loadOrder: %v", err)
	}
	_ = loaded
	if len(loadedItems) != 1 || loadedItems[0].DiskMonths != 3 {
		t.Fatalf("reloaded items=%+v want DiskMonths=3", loadedItems)
	}
	js := orderJSON(loaded, loadedItems)
	rows := js["items"].([]map[string]interface{})
	if len(rows) != 1 {
		t.Fatalf("orderJSON items=%d want 1", len(rows))
	}
	if m, ok := asInt64(rows[0]["disk_months"]); !ok || m != 3 {
		t.Fatalf("orderJSON disk_months=%v want 3", rows[0]["disk_months"])
	}
}

func TestCreateOrderGitlabDiskDefaultsMonthsToOne(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9610000111
	seedVIP1Membership(t, tenantID)

	// DiskMonths 缺省（0）按 1 计：展示金额与旧行为一致
	_, items, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1"},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	wantSub := DefaultGitlabDiskUnitPriceCents * GitlabDiskMinPurchaseGB
	if items[0].SubtotalYuanCents != wantSub {
		t.Fatalf("subtotal=%d want %d", items[0].SubtotalYuanCents, wantSub)
	}
	if items[0].DiskMonths != 1 {
		t.Fatalf("DiskMonths=%d want 1", items[0].DiskMonths)
	}
}

func TestCreateOrderGitlabDiskRejectsMonthsOverMax(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9610000112
	seedVIP1Membership(t, tenantID)

	_, _, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 37},
	})
	if err == nil {
		t.Fatal("expected error for disk_months=37")
	}
	if !strings.Contains(err.Error(), "1～36") {
		t.Fatalf("got %v", err)
	}
}

// OPT-20260903-010 + 012：支付按订单行月数写 disk_months；
// 首次购买（待管理员开通）到期日留空，由开通成功时按开通日起算。
func TestMarkOrderPaidGitlabDiskRecordsMonthsPendingAdmin(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	const tenantID int64 = 9610000113
	seedVIP1Membership(t, tenantID)

	order, _, err := createOrder(context.Background(), tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 3},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(context.Background(), order.ID, "wechat", "ut-gl-disk-months", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}

	var diskMonths int64
	var expiresAt, status string
	if err := db.QueryRow(`
		SELECT disk_months, disk_expires_at, provisioning_status FROM billing_tenant_gitlab_resource
		WHERE tenant_id = ? AND region = ?`, tenantID, "tencent-sh-1",
	).Scan(&diskMonths, &expiresAt, &status); err != nil {
		t.Fatalf("scan resource row: %v", err)
	}
	if diskMonths != 3 {
		t.Fatalf("disk_months=%d want 3（来自订单行月数）", diskMonths)
	}
	if strings.TrimSpace(expiresAt) != "" {
		t.Fatalf("disk_expires_at=%q want empty（首次待开通，到期自开通日起算）", expiresAt)
	}
	if status != "pending_admin" {
		t.Fatalf("provisioning_status=%q want pending_admin", status)
	}
}
