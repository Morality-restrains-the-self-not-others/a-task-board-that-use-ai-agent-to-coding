package main

import (
	"testing"
)

func TestPersistWechatPayVouchersWritesLedgerDespitePrefixedPaymentRef(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	tenantID := int64(880401)
	orderID := generateSnowflakeID()
	outNo := "WX878981209491800064"
	txn := "4500000359202608221274536815"
	seedTradeNoOrder(t, orderID, tenantID, "ORD-VOUCHER", "wechat:"+outNo)

	now := utcNow()
	accID := generateSnowflakeID()
	if _, err := db.Exec(
		`INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at) VALUES (?, ?, 0, ?, ?)`,
		accID, tenantID, now, now,
	); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_payment_ledger (
			id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
			points, remaining_points, amount_minor, currency, billing_transaction_id, created_at, expires_at
		) VALUES (?, ?, ?, 'wechat', ?, '', 55, 55, 55, 'CNY', NULL, ?, ?)`,
		generateSnowflakeID(), tenantID, accID, outNo, now, now,
	); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	persistWechatPayVouchers(orderID, outNo, txn)

	var gotOut, gotTxn, cap string
	if err := db.QueryRow(
		`SELECT out_trade_no, wechat_transaction_id FROM billing_resource_order WHERE id = ?`,
		orderID,
	).Scan(&gotOut, &gotTxn); err != nil {
		t.Fatalf("load order: %v", err)
	}
	if gotOut != outNo || gotTxn != txn {
		t.Fatalf("order vouchers out=%q txn=%q want %s / %s", gotOut, gotTxn, outNo, txn)
	}
	if err := db.QueryRow(
		`SELECT provider_capture_id FROM billing_payment_ledger WHERE tenant_id = ? AND provider_ref = ?`,
		tenantID, outNo,
	).Scan(&cap); err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if cap != txn {
		t.Fatalf("ledger capture=%q want %s", cap, txn)
	}
}

func TestRecordWechatTransactionIDForOrderWithPrefixedPaymentRef(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	tenantID := int64(880402)
	orderID := generateSnowflakeID()
	outNo := "WX877909254923649024"
	txn := "4500000360202608195617269096"
	seedTradeNoOrder(t, orderID, tenantID, "ORD-VOUCHER-2", "wechat:"+outNo)
	now := utcNow()
	accID := generateSnowflakeID()
	if _, err := db.Exec(
		`INSERT INTO billing_account (id, tenant_id, balance, created_at, updated_at) VALUES (?, ?, 0, ?, ?)`,
		accID, tenantID, now, now,
	); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO billing_payment_ledger (
			id, tenant_id, account_id, channel, provider_ref, provider_capture_id,
			points, remaining_points, amount_minor, currency, billing_transaction_id, created_at, expires_at
		) VALUES (?, ?, ?, 'wechat', ?, '', 55, 55, 55, 'CNY', NULL, ?, ?)`,
		generateSnowflakeID(), tenantID, accID, outNo, now, now,
	); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	recordWechatTransactionIDForOrder(formatID(orderID), txn)

	var cap, gotTxn string
	if err := db.QueryRow(
		`SELECT provider_capture_id FROM billing_payment_ledger WHERE tenant_id = ? AND provider_ref = ?`,
		tenantID, outNo,
	).Scan(&cap); err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if cap != txn {
		t.Fatalf("ledger capture=%q want %s", cap, txn)
	}
	if err := db.QueryRow(
		`SELECT wechat_transaction_id FROM billing_resource_order WHERE id = ?`, orderID,
	).Scan(&gotTxn); err != nil {
		t.Fatalf("load order: %v", err)
	}
	if gotTxn != txn {
		t.Fatalf("order txn=%q want %s", gotTxn, txn)
	}
}
