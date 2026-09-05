package main

import (
	"testing"
)

func ensureBillingTransactionTable(t *testing.T) {
	t.Helper()
	_, err := billDB.Exec(`
		CREATE TABLE IF NOT EXISTS billing_transaction (
			id BIGINT NOT NULL PRIMARY KEY,
			account_id BIGINT NOT NULL,
			transaction_type VARCHAR(32) NOT NULL,
			amount BIGINT NOT NULL,
			balance_before BIGINT NOT NULL DEFAULT 0,
			balance_after BIGINT NOT NULL DEFAULT 0,
			points_source_type VARCHAR(64) NULL,
			user_id VARCHAR(64) NULL,
			usage_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
			description VARCHAR(255) NULL,
			transaction_id VARCHAR(100) NOT NULL,
			created_at DATETIME NOT NULL,
			UNIQUE KEY uq_txn (transaction_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create billing_transaction: %v", err)
	}
}

func insertPerfTxn(t *testing.T, id int64, accountID int64, txnType, source, userID string, amount int64, txnID, createdAt string) {
	t.Helper()
	var uid interface{}
	if userID == "" {
		uid = nil
	} else {
		uid = userID
	}
	_, err := billDB.Exec(`
		INSERT INTO billing_transaction (
			id, account_id, transaction_type, amount, points_source_type, user_id,
			transaction_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, accountID, txnType, amount, source, uid, txnID, createdAt)
	if err != nil {
		t.Fatalf("insert txn %s: %v", txnID, err)
	}
}

func insertPerfEdge(t *testing.T, referred, referrer, boundAt string) {
	t.Helper()
	_, err := billDB.Exec(`
		INSERT INTO billing_referral_edge (
			referred_user_id, referrer_user_id, referrer_tenant_id, bound_at, channel_code, created_at, updated_at
		) VALUES (?, ?, 1, ?, 'DEFAULT', 't', 't')`, referred, referrer, boundAt)
	if err != nil {
		t.Fatalf("insert edge: %v", err)
	}
}

// T37: 真实支付写在 transaction_type=recharge + points_source_type=user_recharge_*，
// 或 consumption + resource_purchase。误查 transaction_type IN (user_recharge_*) 会得到 0。
func TestGetReferralPerformance_CountsResourcePurchase(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()
	ensureBillingTransactionTable(t)

	insertPerfEdge(t, "referred-a", "referrer-a", "2026-08-21 00:00:00.000000")
	insertPerfTxn(t, 1, 100, "consumption", "resource_purchase", "referred-a", 55, "order:a", "2026-08-21 12:00:00")
	insertPerfTxn(t, 2, 100, "recharge", "admin_grant", "referred-a", 0, "grant:a", "2026-08-21 12:00:01")

	got, err := getReferralPerformance("referrer-a", "", "")
	if err != nil {
		t.Fatalf("getReferralPerformance: %v", err)
	}
	if got.ReferralCount != 1 {
		t.Fatalf("referral_count=%d want 1", got.ReferralCount)
	}
	if len(got.ReferredUsers) != 1 {
		t.Fatalf("referred_users=%d want 1", len(got.ReferredUsers))
	}
	if got.ReferredUsers[0].RechargeCount != 1 {
		t.Errorf("recharge_count=%d want 1 (resource_purchase; admin_grant 不计)", got.ReferredUsers[0].RechargeCount)
	}
	if got.ReferredUsers[0].RechargeTotal != "0.55" {
		t.Errorf("recharge_total=%q want 0.55", got.ReferredUsers[0].RechargeTotal)
	}
	if got.ReferredRechargeTotal != "0.55" {
		t.Errorf("referred_recharge_total=%q want 0.55", got.ReferredRechargeTotal)
	}
}

// T38: 历史 resource_purchase 常缺 user_id，须经同账户其它流水反查。
func TestGetReferralPerformance_NullUserIDViaAccount(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()
	ensureBillingTransactionTable(t)

	insertPerfEdge(t, "referred-b", "referrer-b", "2026-08-20 00:00:00.000000")
	insertPerfTxn(t, 11, 200, "recharge", "admin_grant", "referred-b", 0, "grant:b", "2026-08-20 11:00:00")
	insertPerfTxn(t, 12, 200, "consumption", "resource_purchase", "", 55, "order:b", "2026-08-20 11:10:00")

	got, err := getReferralPerformance("referrer-b", "", "")
	if err != nil {
		t.Fatalf("getReferralPerformance: %v", err)
	}
	if len(got.ReferredUsers) != 1 {
		t.Fatalf("referred_users=%d want 1", len(got.ReferredUsers))
	}
	if got.ReferredUsers[0].RechargeCount != 1 || got.ReferredUsers[0].RechargeTotal != "0.55" {
		t.Errorf("user=%s count=%d total=%q want count=1 total=0.55",
			got.ReferredUsers[0].UserID, got.ReferredUsers[0].RechargeCount, got.ReferredUsers[0].RechargeTotal)
	}
}

// T39: 打开被推荐人抽屉时向下推荐为 0，须回填 as_referred 自身支付，避免「0 人 / 暂无记录」。
func TestGetReferralPerformance_AsReferredIncludesOwnPayment(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()
	ensureBillingTransactionTable(t)

	insertPerfEdge(t, "referred-c", "referrer-c", "2026-08-21 12:41:17.000000")
	insertPerfTxn(t, 21, 300, "consumption", "resource_purchase", "referred-c", 55, "order:c", "2026-08-21 12:42:10")

	got, err := getReferralPerformance("referred-c", "", "")
	if err != nil {
		t.Fatalf("getReferralPerformance: %v", err)
	}
	if got.ReferralCount != 0 {
		t.Errorf("referral_count=%d want 0 (被推荐人无向下推荐)", got.ReferralCount)
	}
	if got.AsReferred == nil {
		t.Fatal("as_referred must be set when uid is a referred user")
	}
	if got.AsReferred.ReferrerUserID != "referrer-c" {
		t.Errorf("referrer_user_id=%q want referrer-c", got.AsReferred.ReferrerUserID)
	}
	if got.AsReferred.RechargeCount != 1 || got.AsReferred.RechargeTotal != "0.55" {
		t.Errorf("as_referred count=%d total=%q want 1 / 0.55", got.AsReferred.RechargeCount, got.AsReferred.RechargeTotal)
	}
	if got.OwnRechargeCount != 1 || got.OwnRechargeTotal != "0.55" {
		t.Errorf("own_recharge count=%d total=%q want 1 / 0.55", got.OwnRechargeCount, got.OwnRechargeTotal)
	}
}

// T40: 推荐绩效「自身支付」流水表——own_payments 返回自身实付明细（时间、金额、points_source_type），
// admin_grant 不计，按 created_at 倒序。
func TestGetReferralPerformance_OwnPaymentsList(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()
	ensureBillingTransactionTable(t)

	insertPerfEdge(t, "referred-e", "referrer-e", "2026-08-21 00:00:00.000000")
	insertPerfTxn(t, 41, 500, "recharge", "user_recharge_wechat", "referrer-e", 100, "recharge:e1", "2026-08-21 09:00:00")
	insertPerfTxn(t, 42, 500, "consumption", "resource_purchase", "referrer-e", 55, "order:e2", "2026-08-21 10:00:00")
	insertPerfTxn(t, 43, 500, "recharge", "admin_grant", "referrer-e", 0, "grant:e3", "2026-08-21 11:00:00")

	got, err := getReferralPerformance("referrer-e", "", "")
	if err != nil {
		t.Fatalf("getReferralPerformance: %v", err)
	}
	if got.OwnRechargeCount != 2 {
		t.Errorf("own_recharge_count=%d want 2 (admin_grant 不计)", got.OwnRechargeCount)
	}
	if len(got.OwnPayments) != 2 {
		t.Fatalf("own_payments=%d want 2", len(got.OwnPayments))
	}
	if got.OwnPayments[0].Amount != "0.55" || got.OwnPayments[0].PointsSourceType != "resource_purchase" {
		t.Errorf("first payment=%+v want amount=0.55 source=resource_purchase", got.OwnPayments[0])
	}
	if got.OwnPayments[1].Amount != "1.00" || got.OwnPayments[1].PointsSourceType != "user_recharge_wechat" {
		t.Errorf("second payment=%+v want amount=1.00 source=user_recharge_wechat", got.OwnPayments[1])
	}
	if got.OwnPayments[0].PaidAt == "" {
		t.Errorf("paid_at must not be empty")
	}
}

func TestGetReferralPerformance_WrongTxnTypeDoesNotCount(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()
	ensureBillingTransactionTable(t)

	insertPerfEdge(t, "referred-d", "referrer-d", "2026-08-01 00:00:00.000000")
	insertPerfTxn(t, 31, 400, "user_recharge_wechat", "user_recharge_wechat", "referred-d", 999, "legacy:d", "2026-08-01 10:00:00")

	got, err := getReferralPerformance("referrer-d", "", "")
	if err != nil {
		t.Fatalf("getReferralPerformance: %v", err)
	}
	if len(got.ReferredUsers) != 1 {
		t.Fatalf("referred_users=%d want 1", len(got.ReferredUsers))
	}
	if got.ReferredUsers[0].RechargeCount != 0 {
		t.Errorf("legacy transaction_type=user_recharge_wechat must not count; count=%d", got.ReferredUsers[0].RechargeCount)
	}
}

func TestGetReferralPerformance_UsesConfiguredReferralRate(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()
	ensureBillingTransactionTable(t)
	_, err := billDB.Exec(`
		CREATE TABLE IF NOT EXISTS billing_referral_config (
		  singleton_key VARCHAR(64) NOT NULL PRIMARY KEY,
		  settle_delay_days INTEGER NOT NULL DEFAULT 15,
		  profit_sharing_ratio_percent INTEGER NOT NULL DEFAULT 30,
		  referral_rate_percent INTEGER NOT NULL DEFAULT 5,
		  updated_at VARCHAR(255) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create config: %v", err)
	}
	if _, err := updateReferralSettleConfig(15, 12); err != nil {
		t.Fatalf("seed ratios: %v", err)
	}

	got, err := getReferralPerformance("referrer-rate", "", "")
	if err != nil {
		t.Fatalf("getReferralPerformance: %v", err)
	}
	if got.Commission == nil || got.Commission.CommissionRateDisplay != "5%" {
		t.Fatalf("commission_rate_display=%v want 5%% (fixed; ignore stored 12)", got.Commission)
	}
}
