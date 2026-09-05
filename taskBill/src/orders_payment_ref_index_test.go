package main

import "testing"

// OPT-20260821-038: 管理端交易单号查询对 payment_ref 做等值匹配（listOrdersByTradeNo），
// 须有独立索引，避免订单累积后全表扫描。
func TestPaymentRefIndexExists(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	defer cleanup()
	var n int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'billing_resource_order'
		  AND INDEX_NAME = 'idx_billing_resource_order_payment_ref'
	`).Scan(&n); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if n != 1 {
		t.Fatalf("payment_ref index count=%d want 1", n)
	}
}
