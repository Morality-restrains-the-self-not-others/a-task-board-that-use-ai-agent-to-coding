package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResourceQuotasTaskPostGiftedPurchased(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	ctx := context.Background()
	const tenantID = int64(9610000001)

	if _, err := adminGrantResources(ctx, tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 10},
	}, "admin-1", ""); err != nil {
		t.Fatalf("grant: %v", err)
	}

	order, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 5},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(ctx, order.ID, "wechat", "ut-ref-1", tenantID); err != nil {
		t.Fatalf("markOrderPaid: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tenant/"+formatID(tenantID)+"/billing/quotas/", nil)
	rr := httptest.NewRecorder()
	handleResourceQuotas(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if int64(body["task_post_quota"].(float64)) != 15 {
		t.Fatalf("total=%v want 15", body["task_post_quota"])
	}
	if int64(body["task_post_quota_gifted"].(float64)) != 10 {
		t.Fatalf("gifted=%v want 10", body["task_post_quota_gifted"])
	}
	if int64(body["task_post_quota_purchased"].(float64)) != 5 {
		t.Fatalf("purchased=%v want 5", body["task_post_quota_purchased"])
	}

	var sourceKind string
	var orderID int64
	if err := db.QueryRow(`
		SELECT source_kind, COALESCE(order_id, 0) FROM billing_resource_grant
		WHERE tenant_id = ? AND source_kind = 'purchase' AND resource_type = 'task_post'`,
		tenantID,
	).Scan(&sourceKind, &orderID); err != nil {
		t.Fatalf("purchase lot: %v", err)
	}
	if sourceKind != grantSourcePurchase || orderID != order.ID {
		t.Fatalf("lot source=%s order=%d want purchase/%d", sourceKind, orderID, order.ID)
	}
}

func TestConsumeTaskPostQuotaPrefersGiftThenPurchase(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	ctx := context.Background()
	const tenantID = int64(9610000002)

	grantOut, err := adminGrantResources(ctx, tenantID, []ResourceGrantInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	}, "admin-1", "")
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	giftOrderID, err := parseIDField(grantOut["order_id"])
	if err != nil || giftOrderID == 0 {
		t.Fatalf("gift order_id=%v err=%v", grantOut["order_id"], err)
	}

	order, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(ctx, order.ID, "wechat", "ut-ref-2", tenantID); err != nil {
		t.Fatalf("pay: %v", err)
	}

	if _, err := consumeAndRecordTaskPostQuota(ctx, tenantID, "task-gift", "", "u1", "", "idem-gift"); err != nil {
		t.Fatalf("consume gift: %v", err)
	}
	var giftRemain, buyRemain int64
	_ = db.QueryRow(`SELECT remaining FROM billing_resource_grant WHERE tenant_id=? AND source_kind='gift' AND resource_type='task_post'`, tenantID).Scan(&giftRemain)
	_ = db.QueryRow(`SELECT remaining FROM billing_resource_grant WHERE tenant_id=? AND source_kind='purchase' AND resource_type='task_post'`, tenantID).Scan(&buyRemain)
	if giftRemain != 0 || buyRemain != 1 {
		t.Fatalf("after gift consume remaining gift=%d buy=%d", giftRemain, buyRemain)
	}
	var related int64
	if err := db.QueryRow(`SELECT COALESCE(related_order_id,0) FROM billing_transaction WHERE task_id='task-gift'`).Scan(&related); err != nil {
		t.Fatalf("related: %v", err)
	}
	if related != giftOrderID {
		t.Fatalf("gift consume related_order_id=%d want %d", related, giftOrderID)
	}

	if _, err := consumeAndRecordTaskPostQuota(ctx, tenantID, "task-buy", "", "u1", "", "idem-buy"); err != nil {
		t.Fatalf("consume buy: %v", err)
	}
	_ = db.QueryRow(`SELECT remaining FROM billing_resource_grant WHERE tenant_id=? AND source_kind='purchase' AND resource_type='task_post'`, tenantID).Scan(&buyRemain)
	if buyRemain != 0 {
		t.Fatalf("purchase remaining=%d want 0", buyRemain)
	}
	if err := db.QueryRow(`SELECT COALESCE(related_order_id,0) FROM billing_transaction WHERE task_id='task-buy'`).Scan(&related); err != nil {
		t.Fatalf("related buy: %v", err)
	}
	if related != order.ID {
		t.Fatalf("purchase consume related_order_id=%d want %d", related, order.ID)
	}

	loaded, items, err := loadOrder(tenantID, order.ID)
	if err != nil {
		t.Fatalf("loadOrder: %v", err)
	}
	js := orderJSON(loaded, items)
	rc, _ := js["resource_consumption"].(map[string]interface{})
	tp, _ := rc["task_post"].(map[string]interface{})
	if n, _ := asInt64(tp["granted"]); n != 1 {
		t.Fatalf("consumption=%v", tp)
	}
	if n, _ := asInt64(tp["remaining"]); n != 0 {
		t.Fatalf("consumption=%v", tp)
	}
	if n, _ := asInt64(tp["consumed"]); n != 1 {
		t.Fatalf("consumption=%v", tp)
	}
	events, _ := tp["events"].([]interface{})
	if len(events) != 1 {
		t.Fatalf("events=%v", tp["events"])
	}
}

func TestConsumeTaskPostRenewalRecordsOrder(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	ctx := context.Background()
	const tenantID = int64(9610000003)
	order, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 1},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(ctx, order.ID, "wechat", "ut-ref-3", tenantID); err != nil {
		t.Fatalf("pay: %v", err)
	}
	if _, err := consumeTaskPostRenewal(ctx, tenantID, "task-renew", "", "u1", "", "", "idem-renew"); err != nil {
		t.Fatalf("renew: %v", err)
	}
	var related int64
	var txnID string
	if err := db.QueryRow(`SELECT COALESCE(related_order_id,0), transaction_id FROM billing_transaction WHERE task_id='task-renew'`).Scan(&related, &txnID); err != nil {
		t.Fatalf("txn: %v", err)
	}
	if related != order.ID {
		t.Fatalf("related=%d want %d", related, order.ID)
	}
	loaded, items, err := loadOrder(tenantID, order.ID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	js := orderJSON(loaded, items)
	tp := js["resource_consumption"].(map[string]interface{})["task_post"].(map[string]interface{})
	ev := tp["events"].([]interface{})[0].(map[string]interface{})
	if ev["action"] != "renewal" {
		t.Fatalf("action=%v", ev["action"])
	}
}

func TestRefundZerosPurchaseLot(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	ctx := context.Background()
	const tenantID = int64(9610000004)
	order, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 3},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	if err := markOrderPaid(ctx, order.ID, "wechat", "ut-ref-4", tenantID); err != nil {
		t.Fatalf("pay: %v", err)
	}
	now := utcNow()
	if err := revokeOrderResourcesOnRefund(ctx, db, order.ID, tenantID, now); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	var remaining int64
	if err := db.QueryRow(`SELECT remaining FROM billing_resource_grant WHERE order_id=? AND source_kind='purchase'`, order.ID).Scan(&remaining); err != nil {
		t.Fatalf("lot: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("remaining=%d want 0", remaining)
	}
}

func TestEnsureTaskPostPurchaseLotsBackfill(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	ctx := context.Background()
	const tenantID = int64(9610000005)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	order, _, err := createOrder(ctx, tenantID, []orderItemInput{
		{ResourceType: ResourceTypeTaskPost, Quantity: 4},
	})
	if err != nil {
		t.Fatalf("createOrder: %v", err)
	}
	now := utcNow()
	if _, err := db.Exec(`UPDATE billing_resource_order SET status='paid', payment_method='wechat', paid_at=? WHERE id=?`, now, order.ID); err != nil {
		t.Fatalf("force paid: %v", err)
	}
	if _, err := db.Exec(`UPDATE billing_account SET task_post_quota=2 WHERE id=?`, acc.ID); err != nil {
		t.Fatalf("quota: %v", err)
	}
	if err := ensureTaskPostPurchaseLots(ctx, tenantID); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	var qty, rem int64
	if err := db.QueryRow(`SELECT quantity, remaining FROM billing_resource_grant WHERE order_id=? AND source_kind='purchase'`, order.ID).Scan(&qty, &rem); err != nil {
		t.Fatalf("lot: %v", err)
	}
	if qty != 4 || rem != 2 {
		t.Fatalf("qty=%d rem=%d want 4/2 (LIFO leftover purchased remaining)", qty, rem)
	}
	if err := ensureTaskPostPurchaseLots(ctx, tenantID); err != nil {
		t.Fatalf("ensure 2: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM billing_resource_grant WHERE order_id=? AND source_kind='purchase'`, order.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("lots=%d want 1 (idempotent)", n)
	}
}
