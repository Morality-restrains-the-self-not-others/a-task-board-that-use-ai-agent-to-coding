package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// ── HTTP route tests (frontend convention alignment) ──
// 前端已迁移到 /api/referral/stats/user_id/{userId}/（taskFE e105bad），后端路由必须同步注册，
// 否则推荐收益统计接口 404 → 前端展示「获取推荐收益统计失败，请稍后重试」。

func TestReferralStatsRoute_NewConventionPath_Unauthenticated(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	mountRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/referral/stats/user_id/user1/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code == http.StatusNotFound {
		t.Fatal("route /api/referral/stats/user_id/{userId}/ 未注册：前端已按新约定路径调用，后端仍为旧路径 → 404")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 (route matched), got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReferralStatsRoute_NewConventionPath_Authenticated(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	mountRoutes(mux)

	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/referral/stats/user_id/user1/", nil), "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var out referralStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if out.ReferralCount != 0 {
		t.Errorf("expected 0 referrals, got %d", out.ReferralCount)
	}
}

func TestHandleReferralStats_IgnoresPathUserId(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()
	_, err := billDB.Exec(`
		INSERT INTO billing_referral_edge (referred_user_id, referrer_user_id, referrer_tenant_id, bound_at, channel_code, created_at, updated_at)
		VALUES ('victim', 'other-user', 1, '2026-08-01 00:00:00.000000', 'OTHER', 't', 't')`)
	if err != nil {
		t.Fatalf("edge: %v", err)
	}
	mux := http.NewServeMux()
	mountRoutes(mux)
	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/referral/stats/user_id/other-user/", nil), "caller")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out referralStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if out.ReferralCount != 0 {
		t.Fatalf("must not leak path user stats, count=%d", out.ReferralCount)
	}
}

func TestReferralStatsRoute_LegacyPathStillServed(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	mountRoutes(mux)

	req := withReferralUser(httptest.NewRequest(http.MethodGet, "/api/user/user1/profile/referral-stats/", nil), "user1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("legacy path expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// setupReferralStatsTestDB creates both the main DB and billDB for stats tests.
func setupReferralStatsTestDB(t *testing.T) (cleanup func()) {
	t.Helper()

	baseDSN := strings.TrimSpace(os.Getenv("TASKREFERRAL_MYSQL_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = "taskapp:taskapp123@tcp(127.0.0.1:3306)/"
	}
	if !strings.HasSuffix(baseDSN, "/") {
		baseDSN += "/"
	}

	// Main DB (referral code tables)
	mainName := "test_ref_stats_" + sanitizeDBName(t.Name()) + "_main"
	if len(mainName) > 64 {
		mainName = mainName[:64]
	}

	adminDSN := baseDSN + "?charset=utf8mb4&parseTime=true&multiStatements=true"
	adminDB, err := sql.Open("mysql", adminDSN)
	if err != nil {
		t.Fatalf("open admin: %v", err)
	}
	defer adminDB.Close()

	// Create main test DB
	if _, err := adminDB.Exec("CREATE DATABASE IF NOT EXISTS `" + mainName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create main test db %s: %v", mainName, err)
	}
	mainDSN := baseDSN + mainName + "?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true"
	db, err = sql.Open("mysql", mainDSN)
	if err != nil {
		adminDB.Exec("DROP DATABASE IF EXISTS `" + mainName + "`")
		t.Fatalf("open main test db: %v", err)
	}
	db.SetMaxOpenConns(1)

	// Create bill test DB for billing_referral_edge and billing_referral_commission_accrual
	billName := "test_ref_stats_" + sanitizeDBName(t.Name()) + "_bill"
	if len(billName) > 64 {
		billName = billName[:64]
	}
	if _, err := adminDB.Exec("CREATE DATABASE IF NOT EXISTS `" + billName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatalf("create bill test db %s: %v", billName, err)
	}
	billDSN := baseDSN + billName + "?charset=utf8mb4&parseTime=true&loc=Local&multiStatements=true"
	billDB, err = sql.Open("mysql", billDSN)
	if err != nil {
		adminDB.Exec("DROP DATABASE IF EXISTS `" + billName + "`")
		t.Fatalf("open bill test db: %v", err)
	}
	billDB.SetMaxOpenConns(1)

	// Create billing_referral_edge table in billDB
	_, err = billDB.Exec(`
		CREATE TABLE IF NOT EXISTS billing_referral_edge (
			id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			referred_user_id VARCHAR(255) NOT NULL,
			referrer_user_id VARCHAR(255) NOT NULL,
			referrer_tenant_id BIGINT NOT NULL DEFAULT 0,
			bound_at VARCHAR(255) NOT NULL,
			channel_code VARCHAR(32) NOT NULL DEFAULT '',
			commission_eligible TINYINT(1) NOT NULL DEFAULT 0,
			created_at VARCHAR(255) NOT NULL,
			updated_at VARCHAR(255) NOT NULL,
			UNIQUE KEY uk_referred (referred_user_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create billing_referral_edge table: %v", err)
	}

	// Create billing_referral_commission_accrual table in billDB
	_, err = billDB.Exec(`
		CREATE TABLE IF NOT EXISTS billing_referral_commission_accrual (
			id BIGINT NOT NULL PRIMARY KEY,
			referrer_user_id VARCHAR(255) NOT NULL,
			referred_user_id VARCHAR(255) NOT NULL,
			referrer_tenant_id BIGINT NOT NULL DEFAULT 0,
			source_txn_id VARCHAR(255) NOT NULL,
			source_txn_db_id BIGINT NULL,
			consumption_points BIGINT NOT NULL DEFAULT 0,
			commission_points BIGINT NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			consumed_at VARCHAR(255) NOT NULL,
			channel_code VARCHAR(32) NOT NULL DEFAULT '',
			settle_after VARCHAR(255) NOT NULL,
			settled_at VARCHAR(255) NULL,
			settle_txn_id VARCHAR(255) NULL,
			voided_at VARCHAR(255) NULL,
			void_reason TEXT NULL,
			created_at VARCHAR(255) NOT NULL,
			updated_at VARCHAR(255) NOT NULL,
			UNIQUE KEY uk_source_txn (source_txn_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create billing_referral_commission_accrual table: %v", err)
	}

	cleanup = func() {
		if db != nil {
			db.Close()
			db = nil
		}
		if billDB != nil {
			billDB.Close()
			billDB = nil
		}
		cleanDB, err := sql.Open("mysql", adminDSN)
		if err == nil {
			cleanDB.Exec("DROP DATABASE IF EXISTS `" + mainName + "`")
			cleanDB.Exec("DROP DATABASE IF EXISTS `" + billName + "`")
			cleanDB.Close()
		}
	}

	return cleanup
}

func sanitizeDBName(s string) string {
	return strings.ToLower(strings.NewReplacer(
		"/", "_", "-", "_", "(", "", ")", "", "*", "", "#", "",
	).Replace(s))
}

func TestGetReferralStats_NoData(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()

	stats, err := getReferralStats("user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.ReferralCount != 0 {
		t.Errorf("expected 0 referrals, got %d", stats.ReferralCount)
	}
	if stats.ReferralRateDisplay != "5%" {
		t.Errorf("expected rate '5%%', got %q", stats.ReferralRateDisplay)
	}
	if len(stats.MonthlyEarnings) != 0 {
		t.Errorf("expected 0 monthly earnings, got %d", len(stats.MonthlyEarnings))
	}
	if stats.Channels == nil {
		t.Errorf("channels must be empty slice not nil")
	}
}

func TestGetReferralStats_WithReferrals(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()

	// Insert referral edges
	now := "2026-07-01 00:00:00.000000"
	_, err := billDB.Exec(`
		INSERT INTO billing_referral_edge (referred_user_id, referrer_user_id, referrer_tenant_id, bound_at, created_at, updated_at)
		VALUES
			('ref1', 'user123', 1, ?, ?, ?),
			('ref2', 'user123', 1, ?, ?, ?),
			('ref3', 'user456', 1, ?, ?, ?)`, now, now, now, now, now, now, now, now, now)
	if err != nil {
		t.Fatalf("insert referral edges: %v", err)
	}

	stats, err := getReferralStats("user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.ReferralCount != 2 {
		t.Errorf("expected 2 referrals, got %d", stats.ReferralCount)
	}
}

func TestGetReferralStats_WithMonthlyEarnings(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()

	// Insert commission accruals
	_, err := billDB.Exec(`
		INSERT INTO billing_referral_commission_accrual (
			id, referrer_user_id, referred_user_id, referrer_tenant_id,
			source_txn_id, consumption_points, commission_points,
			status, consumed_at, settle_after, created_at, updated_at
		) VALUES
			(1, 'user123', 'ref1', 1, 'txn1', 10000, 500, 'settled', '2026-07-15 10:00:00.000000', '2026-07-23 10:00:00.000000', ?, ?),
			(2, 'user123', 'ref2', 1, 'txn2', 20000, 1000, 'pending', '2026-07-20 10:00:00.000000', '2026-07-28 10:00:00.000000', ?, ?),
			(3, 'user123', 'ref1', 1, 'txn3', 5000, 250, 'settled', '2026-06-10 10:00:00.000000', '2026-06-18 10:00:00.000000', ?, ?)`,
		"2026-07-01 00:00:00.000000", "2026-07-01 00:00:00.000000",
		"2026-07-01 00:00:00.000000", "2026-07-01 00:00:00.000000",
		"2026-07-01 00:00:00.000000", "2026-07-01 00:00:00.000000")
	if err != nil {
		t.Fatalf("insert accruals: %v", err)
	}

	stats, err := getReferralStats("user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stats.MonthlyEarnings) != 2 {
		t.Fatalf("expected 2 monthly entries, got %d", len(stats.MonthlyEarnings))
	}

	// Most recent first (ORDER BY month DESC)
	july := stats.MonthlyEarnings[0]
	if july.Month != "2026-07" {
		t.Errorf("expected first entry to be 2026-07, got %s", july.Month)
	}
	// 10000 + 20000 = 30000 points = 300.00 yuan
	if july.Consumption != "300.00" {
		t.Errorf("expected July consumption 300.00, got %s", july.Consumption)
	}
	// 500 + 1000 = 1500 points = 15.00 yuan
	if july.Commission != "15.00" {
		t.Errorf("expected July commission 15.00, got %s", july.Commission)
	}

	june := stats.MonthlyEarnings[1]
	if june.Month != "2026-06" {
		t.Errorf("expected second entry to be 2026-06, got %s", june.Month)
	}
	if june.Consumption != "50.00" {
		t.Errorf("expected June consumption 50.00, got %s", june.Consumption)
	}
	if june.Commission != "2.50" {
		t.Errorf("expected June commission 2.50, got %s", june.Commission)
	}
}

func TestGetReferralStats_ChannelAndDateFilter(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()

	_, err := billDB.Exec(`
		INSERT INTO billing_referral_edge (referred_user_id, referrer_user_id, referrer_tenant_id, bound_at, channel_code, created_at, updated_at)
		VALUES
			('a', 'user123', 1, '2026-07-10 00:00:00.000000', 'CODEWX', 't', 't'),
			('b', 'user123', 1, '2026-08-10 00:00:00.000000', 'CODEDY', 't', 't'),
			('c', 'user123', 1, '2026-06-10 00:00:00.000000', 'CODEWX', 't', 't')`)
	if err != nil {
		t.Fatalf("edges: %v", err)
	}
	_, err = billDB.Exec(`
		INSERT INTO billing_referral_commission_accrual (
			id, referrer_user_id, referred_user_id, referrer_tenant_id,
			source_txn_id, consumption_points, commission_points,
			status, consumed_at, channel_code, settle_after, created_at, updated_at
		) VALUES
			(11, 'user123', 'a', 1, 't1', 10000, 500, 'pending', '2026-07-15 10:00:00.000000', 'CODEWX', 'x', 'x', 'x'),
			(12, 'user123', 'b', 1, 't2', 20000, 1000, 'pending', '2026-08-15 10:00:00.000000', 'CODEDY', 'x', 'x', 'x')`)
	if err != nil {
		t.Fatalf("accruals: %v", err)
	}

	wx, err := getReferralStatsFiltered("user123", referralStatsFilter{ChannelCode: "CODEWX"})
	if err != nil {
		t.Fatalf("wx: %v", err)
	}
	if wx.ReferralCount != 2 {
		t.Fatalf("wx count=%d", wx.ReferralCount)
	}
	if len(wx.MonthlyEarnings) != 1 || wx.MonthlyEarnings[0].Commission != "5.00" {
		t.Fatalf("wx earnings=%v", wx.MonthlyEarnings)
	}

	july, err := getReferralStatsFiltered("user123", referralStatsFilter{FromDate: "2026-07-01", ToDate: "2026-07-31"})
	if err != nil {
		t.Fatalf("july: %v", err)
	}
	if july.ReferralCount != 1 {
		t.Fatalf("july count=%d", july.ReferralCount)
	}
	if len(july.MonthlyEarnings) != 1 || july.MonthlyEarnings[0].Month != "2026-07" {
		t.Fatalf("july earnings=%v", july.MonthlyEarnings)
	}
}

func ensureStatsBillingReferralConfig(t *testing.T, rate int) {
	t.Helper()
	_, err := billDB.Exec(`
		CREATE TABLE IF NOT EXISTS billing_referral_config (
		  singleton_key VARCHAR(64) NOT NULL PRIMARY KEY,
		  settle_delay_days INTEGER NOT NULL DEFAULT 15,
		  profit_sharing_ratio_percent INTEGER NOT NULL DEFAULT 30,
		  referral_rate_percent INTEGER NOT NULL DEFAULT 5,
		  updated_at VARCHAR(255) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		t.Fatalf("create billing_referral_config: %v", err)
	}
	if _, err := updateReferralSettleConfig(15, rate); err != nil {
		t.Fatalf("seed referral_rate_percent=%d: %v", rate, err)
	}
}

func TestGetReferralStats_FixedFivePercentIgnoresWechat(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()

	stats, err := getReferralStats("user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.ReferralRateDisplay != "5%" {
		t.Fatalf("expected fixed rate 5%%, got %q", stats.ReferralRateDisplay)
	}
}

func TestGetReferralStats_FixedFivePercentIgnoresStoredConfig(t *testing.T) {
	cleanup := setupReferralStatsTestDB(t)
	defer cleanup()
	ensureStatsBillingReferralConfig(t, 12)

	stats, err := getReferralStats("user123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.ReferralRateDisplay != "5%" {
		t.Fatalf("referral_rate_display=%q want 5%% (fixed; not from billing_referral_config)", stats.ReferralRateDisplay)
	}
}

func TestGetReferralStats_BillDBNotAvailable(t *testing.T) {
	// Save and restore billDB
	oldBillDB := billDB
	billDB = nil
	defer func() { billDB = oldBillDB }()

	_, err := getReferralStats("user123")
	if err == nil {
		t.Fatal("expected error when billDB is nil")
	}
	if err.Error() != "billDB unavailable" {
		t.Errorf("expected 'billDB unavailable', got %q", err.Error())
	}
}
