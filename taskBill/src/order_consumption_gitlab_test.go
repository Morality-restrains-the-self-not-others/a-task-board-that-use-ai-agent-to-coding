package main

import (
	"context"
	"testing"
)

func TestOrderJSONGitlabDiskPurchaseAppearsInConsumption(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	ctx := context.Background()
	const tenantID int64 = 9610000101
	seedVIP1Membership(t, tenantID)

	order, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(ctx, order.ID, "wechat", "ut-gl-disk-1", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}

	loaded, items, err := loadOrder(tenantID, order.ID)
	if err != nil {
		t.Fatalf("loadOrder: %v", err)
	}
	js := orderJSON(loaded, items)
	rc, _ := js["resource_consumption"].(map[string]interface{})
	disk, _ := rc["gitlab_disk"].(map[string]interface{})
	if disk == nil {
		t.Fatalf("missing gitlab_disk consumption: %v", js["resource_consumption"])
	}
	if n, ok := asInt64(disk["granted"]); !ok || n != GitlabDiskMinPurchaseGB {
		t.Fatalf("granted=%v want %d", disk["granted"], GitlabDiskMinPurchaseGB)
	}
	if n, ok := asInt64(disk["remaining"]); !ok || n != GitlabDiskMinPurchaseGB {
		t.Fatalf("remaining=%v want %d (unused)", disk["remaining"], GitlabDiskMinPurchaseGB)
	}
	if n, ok := asInt64(disk["consumed"]); !ok || n != 0 {
		t.Fatalf("consumed=%v want 0", disk["consumed"])
	}
	if disk["region"] != "tencent-sh-1" {
		t.Fatalf("region=%v", disk["region"])
	}
	if disk["source_kind"] != grantSourcePurchase {
		t.Fatalf("source_kind=%v", disk["source_kind"])
	}
	if _, has := rc["task_post"]; has {
		t.Fatalf("gitlab-only order must not emit task_post zeros: %v", rc["task_post"])
	}
}

func TestOrderJSONGitlabDiskConsumptionAllocatesRegionUsage(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	ctx := context.Background()
	const tenantID int64 = 9610000102
	seedVIP1Membership(t, tenantID)

	order, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(ctx, order.ID, "wechat", "ut-gl-disk-used", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}
	usedBytes := int64(2) * int64(bytesPerGiB)
	if _, err := db.Exec(
		`UPDATE billing_tenant_gitlab_resource SET disk_used_bytes = ? WHERE tenant_id = ? AND region = ?`,
		usedBytes, tenantID, "tencent-sh-1",
	); err != nil {
		t.Fatalf("set disk_used_bytes: %v", err)
	}

	loaded, items, err := loadOrder(tenantID, order.ID)
	if err != nil {
		t.Fatalf("loadOrder: %v", err)
	}
	disk := orderJSON(loaded, items)["resource_consumption"].(map[string]interface{})["gitlab_disk"].(map[string]interface{})
	if n, ok := asInt64(disk["consumed"]); !ok || n != 2 {
		t.Fatalf("consumed=%v want 2", disk["consumed"])
	}
	if n, ok := asInt64(disk["remaining"]); !ok || n != GitlabDiskMinPurchaseGB-2 {
		t.Fatalf("remaining=%v want %d", disk["remaining"], GitlabDiskMinPurchaseGB-2)
	}
}

func TestOrderJSONGitlabTrafficPurchaseAppearsInConsumption(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	ctx := context.Background()
	const tenantID int64 = 9610000103
	seedVIP1Membership(t, tenantID)

	diskOrder, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabDisk, Quantity: GitlabDiskMinPurchaseGB, Region: "tencent-sh-1", DiskMonths: 1},
	})
	if err != nil {
		t.Fatalf("create disk: %v", err)
	}
	if err := markOrderPaid(ctx, diskOrder.ID, "wechat", "ut-gl-disk-pre", tenantID); err != nil {
		t.Fatalf("pay disk: %v", err)
	}

	trafficOrder, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeGitlabTraffic, Quantity: 20, Region: "tencent-sh-1"},
	})
	if err != nil {
		t.Fatalf("create traffic: %v", err)
	}
	if err := markOrderPaid(ctx, trafficOrder.ID, "wechat", "ut-gl-traffic-1", tenantID); err != nil {
		t.Fatalf("pay traffic: %v", err)
	}

	loaded, items, err := loadOrder(tenantID, trafficOrder.ID)
	if err != nil {
		t.Fatalf("loadOrder: %v", err)
	}
	js := orderJSON(loaded, items)
	rc, _ := js["resource_consumption"].(map[string]interface{})
	tr, _ := rc["gitlab_traffic"].(map[string]interface{})
	if tr == nil {
		t.Fatalf("missing gitlab_traffic: %v", rc)
	}
	if n, ok := asInt64(tr["granted"]); !ok || n != 20 {
		t.Fatalf("granted=%v want 20", tr["granted"])
	}
	if _, has := rc["gitlab_disk"]; has {
		t.Fatalf("traffic order must not copy sibling disk line: %v", rc["gitlab_disk"])
	}
}
