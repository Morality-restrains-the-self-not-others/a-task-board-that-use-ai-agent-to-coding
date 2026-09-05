package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBillingStatisticsAggregates(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562496)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, project_id, user_id, workspace_id, task_id,
			billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES
		(?, ?, 'consumption', 30, 100, 70, 'consumption', NULL, NULL, NULL, NULL, NULL, 0, 'c1', 'txn-c1', '2026-06-15 10:00:00'),
		(?, ?, 'consumption', 10, 70, 60, 'consumption', NULL, NULL, NULL, NULL, NULL, 0, 'c2', 'txn-c2', '2026-07-05 10:00:00'),
		(?, ?, 'recharge', 100, 60, 160, 'user_recharge_paypal', NULL, NULL, NULL, NULL, NULL, 0, 'r1', 'txn-r1', '2026-07-06 10:00:00'),
		(?, ?, 'recharge', 50, 160, 210, 'admin_grant', NULL, NULL, NULL, NULL, NULL, 0, 'r2', 'txn-r2', '2026-07-07 10:00:00'),
		(?, ?, 'consumption', 55, 210, 210, 'resource_purchase', NULL, NULL, NULL, NULL, NULL, 0, 'order pay', 'txn-rp1', '2026-07-08 10:00:00')`,
		generateSnowflakeID(), acc.ID,
		generateSnowflakeID(), acc.ID,
		generateSnowflakeID(), acc.ID,
		generateSnowflakeID(), acc.ID,
		generateSnowflakeID(), acc.ID,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/tenant/850256677331562496/billing/statistics/?month_start=2026-07-01",
		nil,
	)
	rr := httptest.NewRecorder()
	handleBillingStatistics(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["month_start"] != "2026-07-01" {
		t.Fatalf("month_start=%v", body["month_start"])
	}
	if int(body["total_consumption_points"].(float64)) != 95 {
		t.Fatalf("total_consumption=%v", body["total_consumption_points"])
	}
	if int(body["monthly_consumption_points"].(float64)) != 65 {
		t.Fatalf("monthly_consumption=%v", body["monthly_consumption_points"])
	}
	// 累计支付 = PayPal 充值 100 + 订单实付 55；后台赠送 50 不计。
	if int(body["user_recharge_points"].(float64)) != 155 {
		t.Fatalf("user_recharge_points=%v", body["user_recharge_points"])
	}
	if int(body["total_consumption_cents"].(float64)) != 95 {
		t.Fatalf("total_consumption_cents=%v", body["total_consumption_cents"])
	}
	if int(body["monthly_consumption_cents"].(float64)) != 65 {
		t.Fatalf("monthly_consumption_cents=%v", body["monthly_consumption_cents"])
	}
	if int(body["user_recharge_cents"].(float64)) != 155 {
		t.Fatalf("user_recharge_cents=%v", body["user_recharge_cents"])
	}
}

// OPT-20260819-041 回归：退款流水（transaction_type='refund'，amount 为负）必须从
// 累计支付扣减，并输出退款扣减拆分字段。
func TestBillingStatisticsSubtractsRefunds(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562497)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, project_id, user_id, workspace_id, task_id,
			billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES
		(?, ?, 'recharge', 110, 0, 110, 'user_recharge_paypal', NULL, NULL, NULL, NULL, NULL, 0, '充值', 'txn-r', '2026-07-01 10:00:00'),
		(?, ?, 'consumption', 55, 110, 110, 'resource_purchase', NULL, NULL, NULL, NULL, NULL, 0, '订单实付', 'txn-rp', '2026-07-02 10:00:00'),
		(?, ?, 'refund', -30, 110, 80, 'refund', NULL, NULL, NULL, NULL, NULL, 0, '退款', 'refund:1', '2026-07-03 10:00:00')`,
		generateSnowflakeID(), acc.ID,
		generateSnowflakeID(), acc.ID,
		generateSnowflakeID(), acc.ID,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/tenant/850256677331562497/billing/statistics/?month_start=2026-07-01",
		nil,
	)
	rr := httptest.NewRecorder()
	handleBillingStatistics(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// 累计支付 = 充值 110 + 订单实付 55 − 退款 30 = 135
	if int(body["user_recharge_points"].(float64)) != 135 {
		t.Fatalf("user_recharge_points=%v, want 135", body["user_recharge_points"])
	}
	if int(body["user_recharge_cents"].(float64)) != 135 {
		t.Fatalf("user_recharge_cents=%v, want 135", body["user_recharge_cents"])
	}
	if int(body["refund_deduction_points"].(float64)) != 30 {
		t.Fatalf("refund_deduction_points=%v, want 30", body["refund_deduction_points"])
	}
	if int(body["refund_deduction_cents"].(float64)) != 30 {
		t.Fatalf("refund_deduction_cents=%v, want 30", body["refund_deduction_cents"])
	}
}

// 本月/累计消耗不得计入已退款、已取消订单的 resource_purchase（账单页 1.10 虚高）。
func TestBillingStatisticsExcludesVoidedOrderPurchasesFromConsumption(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()

	tenantID := int64(850256677331562498)
	acc, _, err := getOrCreateBillingAccount(tenantID, false)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	now := "2026-07-08 10:00:00"
	paidNum, refundedNum, cancelledNum := "ORD-PAID-1", "ORD-REF-1", "ORD-CAN-1"
	_, err = db.Exec(`
		INSERT INTO billing_resource_order (id, tenant_id, order_number, status, total_yuan_cents, created_at)
		VALUES
		(?, ?, ?, 'paid', 55, ?),
		(?, ?, ?, 'refunded', 55, ?),
		(?, ?, ?, 'cancelled', 55, ?)`,
		generateSnowflakeID(), tenantID, paidNum, now,
		generateSnowflakeID(), tenantID, refundedNum, now,
		generateSnowflakeID(), tenantID, cancelledNum, now,
	)
	if err != nil {
		t.Fatalf("seed orders: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, balance_before, balance_after,
			points_source_type, project_id, user_id, workspace_id, task_id,
			billing_unit_id, usage_amount, description, transaction_id, created_at
		) VALUES
		(?, ?, 'consumption', 10, 100, 90, 'quota_consumption', NULL, NULL, NULL, NULL, NULL, 0, '用量', 'txn-usage', '2026-07-05 10:00:00'),
		(?, ?, 'consumption', 55, 90, 90, 'resource_purchase', NULL, NULL, NULL, NULL, NULL, 0, '有效订单', ?, '2026-07-08 10:00:00'),
		(?, ?, 'consumption', 55, 90, 90, 'resource_purchase', NULL, NULL, NULL, NULL, NULL, 0, '已退款订单', ?, '2026-07-08 11:00:00'),
		(?, ?, 'consumption', 55, 90, 90, 'resource_purchase', NULL, NULL, NULL, NULL, NULL, 0, '已取消订单', ?, '2026-07-08 12:00:00')`,
		generateSnowflakeID(), acc.ID,
		generateSnowflakeID(), acc.ID, "order:"+paidNum,
		generateSnowflakeID(), acc.ID, "order:"+refundedNum,
		generateSnowflakeID(), acc.ID, "order:"+cancelledNum,
	)
	if err != nil {
		t.Fatalf("insert txns: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/tenant/850256677331562498/billing/statistics/?month_start=2026-07-01",
		nil,
	)
	rr := httptest.NewRecorder()
	handleBillingStatistics(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// 用量 10 + 仍为 paid 的订单 55；refunded/cancelled 的 55×2 不计。
	if int(body["total_consumption_points"].(float64)) != 65 {
		t.Fatalf("total_consumption_points=%v, want 65 body=%v", body["total_consumption_points"], body)
	}
	if int(body["monthly_consumption_points"].(float64)) != 65 {
		t.Fatalf("monthly_consumption_points=%v, want 65", body["monthly_consumption_points"])
	}
	if int(body["total_consumption_cents"].(float64)) != 65 {
		t.Fatalf("total_consumption_cents=%v, want 65", body["total_consumption_cents"])
	}
}
