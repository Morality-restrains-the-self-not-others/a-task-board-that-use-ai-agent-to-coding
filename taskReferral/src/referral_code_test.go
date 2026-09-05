package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func setupTestReferralDB(t *testing.T) {
	t.Helper()
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	// Create tables (MySQL DDL)
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS referral_code (
		  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		  user_id VARCHAR(255) NOT NULL,
		  status VARCHAR(20) NOT NULL DEFAULT 'pending',
		  applied_at VARCHAR(255) NOT NULL,
		  approved_at VARCHAR(255) NULL,
		  expires_at VARCHAR(255) NULL,
		  rejected_at VARCHAR(255) NULL,
		  reject_reason VARCHAR(512) NOT NULL DEFAULT '',
		  reviewed_by VARCHAR(512) NOT NULL DEFAULT '',
		  personal_intro VARCHAR(2000) NOT NULL DEFAULT '',
		  legal_name VARCHAR(64) NOT NULL DEFAULT '',
		  identity_bind_consented_at DATETIME(6) NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create referral_code table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS referral_policy (
		  singleton_key VARCHAR(64) NOT NULL PRIMARY KEY,
		  mode VARCHAR(20) NOT NULL DEFAULT 'approval',
		  message TEXT NOT NULL,
		  updated_at VARCHAR(255) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create referral_policy table: %v", err)
	}

	_, err = db.Exec(`
		INSERT IGNORE INTO referral_policy (
		  singleton_key, mode, message, updated_at
		) VALUES ('global', 'approval', '', NOW())`)
	if err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	ensureShareCodeTable(t)
	ensureQualificationAuditTable(t)
}

func ensureQualificationAuditTable(t *testing.T) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS referral_qualification_audit (
		  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		  application_id BIGINT NOT NULL,
		  user_id VARCHAR(255) NOT NULL,
		  action VARCHAR(32) NOT NULL,
		  reason VARCHAR(512) NOT NULL,
		  operator_id VARCHAR(255) NOT NULL,
		  idempotency_key VARCHAR(128) NOT NULL,
		  trace_id VARCHAR(64) NOT NULL DEFAULT '',
		  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  UNIQUE KEY uk_referral_qualification_audit_idem (idempotency_key)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create referral_qualification_audit: %v", err)
	}
}

const testValidPersonalIntro = "我是平台活跃用户，日常使用任务与云主机，希望通过推荐帮助同事上手本平台。"
const testValidLegalName = "张三"

func applyReferralWithConsent(userID, intro string) (referralApplyResult, error) {
	return applyReferralCode(userID, intro, testValidLegalName, true)
}

func setReferralPolicyForTest(t *testing.T, mode, message string) {
	t.Helper()
	now := timeNowUTC()
	if _, err := db.Exec(`
		UPDATE referral_policy
		SET mode = ?, message = ?, updated_at = ?
		WHERE singleton_key = ?`, mode, message, now, referralPolicyKey); err != nil {
		t.Fatalf("set policy: %v", err)
	}
}

func insertApprovedReferral(t *testing.T, userID string, expiresAt time.Time) {
	t.Helper()
	now := timeNowUTC()
	expStr := expiresAt.Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO referral_code (user_id, status, applied_at, approved_at, expires_at, reject_reason, reviewed_by, personal_intro)
		VALUES (?, 'approved', ?, ?, ?, '', '', '')`,
		userID, now, now, expStr)
	if err != nil {
		t.Fatalf("insert approved referral: %v", err)
	}
}

// setupStrictReferralDB mirrors production DDL: reject_reason / reviewed_by are
// NOT NULL without DEFAULT (Error 1364 if INSERT omits them).
func setupStrictReferralDB(t *testing.T) {
	t.Helper()
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS referral_code (
		  id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		  user_id VARCHAR(255) NOT NULL,
		  status VARCHAR(20) NOT NULL DEFAULT 'pending',
		  applied_at VARCHAR(255) NOT NULL,
		  approved_at VARCHAR(255) NULL,
		  expires_at VARCHAR(255) NULL,
		  rejected_at VARCHAR(255) NULL,
		  reject_reason TEXT NOT NULL,
		  reviewed_by TEXT NOT NULL,
		  personal_intro VARCHAR(2000) NOT NULL,
		  legal_name VARCHAR(64) NOT NULL DEFAULT '',
		  identity_bind_consented_at DATETIME(6) NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		t.Fatalf("create strict referral_code table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS referral_policy (
		  singleton_key VARCHAR(64) NOT NULL PRIMARY KEY,
		  mode VARCHAR(20) NOT NULL DEFAULT 'approval',
		  message TEXT NOT NULL,
		  updated_at VARCHAR(255) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		t.Fatalf("create referral_policy table: %v", err)
	}

	_, err = db.Exec(`
		INSERT IGNORE INTO referral_policy (
		  singleton_key, mode, message, updated_at
		) VALUES ('global', 'approval', '', NOW())`)
	if err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	ensureShareCodeTable(t)
	ensureQualificationAuditTable(t)
}

// ── applyReferralCode tests ──

// Regression: production schema has reject_reason/reviewed_by NOT NULL without
// DEFAULT; apply must supply empty strings (Error 1364 otherwise).
func TestApplyReferralCodeStrictSchemaNoDefaults(t *testing.T) {
	setupStrictReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	result, err := applyReferralWithConsent("user-strict-1", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply failed on strict schema: %v", err)
	}
	if result.Status != "pending" {
		t.Fatalf("expected pending, got %s", result.Status)
	}

	setReferralPolicyForTest(t, "open", "")
	result, err = applyReferralWithConsent("user-strict-2", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("open-mode apply failed on strict schema: %v", err)
	}
	if result.Status != "approved" {
		t.Fatalf("expected approved, got %s", result.Status)
	}
}

func TestApplyReferralCodeApprovalMode(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	result, err := applyReferralWithConsent("user-1", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "pending" {
		t.Fatalf("expected pending, got %s", result.Status)
	}
	if result.Message == "" {
		t.Fatalf("expected message")
	}
}

func TestApplyReferralCodeOpenMode(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "open", "限额放开")

	result, err := applyReferralWithConsent("user-2", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "approved" {
		t.Fatalf("expected approved, got %s", result.Status)
	}
	assertOpaqueShareCode(t, result.ReferralCode, "user-2")
	if result.ExpiresInDays != referralCodeValidityDays {
		t.Fatalf("expected %d days, got %d", referralCodeValidityDays, result.ExpiresInDays)
	}
}

func TestApplyReferralCodeDuplicatePending(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	_, err := applyReferralWithConsent("user-3", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("first apply failed: %v", err)
	}

	_, err = applyReferralWithConsent("user-3", testValidPersonalIntro)
	if err == nil {
		t.Fatalf("expected duplicate error")
	}
	if !errors.Is(err, errReferralAlreadyPending) {
		t.Fatalf("expected errReferralAlreadyPending, got %v", err)
	}
}

func TestApplyReferralCodeDuplicateActive(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	// Create active referral
	insertApprovedReferral(t, "user-4", time.Now().AddDate(0, 6, 0))

	_, err := applyReferralWithConsent("user-4", testValidPersonalIntro)
	if err == nil {
		t.Fatalf("expected duplicate active error")
	}
}

// ── getReferralCodeStatus tests ──

func TestGetReferralCodeStatusPending(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	_, err := applyReferralWithConsent("user-5", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	status, err := getReferralCodeStatus("user-5")
	if err != nil {
		t.Fatalf("get status failed: %v", err)
	}
	if status.ApplicationStatus == nil || *status.ApplicationStatus != "pending" {
		t.Fatalf("expected pending, got %v", status.ApplicationStatus)
	}
	if status.CanApply {
		t.Fatalf("expected CanApply=false when pending")
	}
	assertOpaqueShareCode(t, status.AccessCode, "user-5")
}

func TestGetReferralCodeStatusApproved(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	insertApprovedReferral(t, "user-6", time.Now().AddDate(0, 6, 0))

	status, err := getReferralCodeStatus("user-6")
	if err != nil {
		t.Fatalf("get status failed: %v", err)
	}
	if !status.HasActiveCode {
		t.Fatalf("expected HasActiveCode=true")
	}
	if status.ApplicationStatus == nil || *status.ApplicationStatus != "approved" {
		t.Fatalf("expected approved, got %v", status.ApplicationStatus)
	}
	assertOpaqueShareCode(t, status.AccessCode, "user-6")
	if status.ReferralCode == nil {
		t.Fatalf("expected referral_code when qualified")
	}
	assertOpaqueShareCode(t, *status.ReferralCode, "user-6")
	if *status.ReferralCode != status.AccessCode {
		t.Fatalf("qualified referral_code should match access_code")
	}
}

func TestGetReferralCodeStatusExpired(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	// Insert approved but already expired
	insertApprovedReferral(t, "user-7", time.Now().AddDate(0, -1, 0)) // 1 month ago

	status, err := getReferralCodeStatus("user-7")
	if err != nil {
		t.Fatalf("get status failed: %v", err)
	}
	// getActiveReferralCode filters expired records → falls through to "no active"
	if status.HasActiveCode {
		t.Fatalf("expected HasActiveCode=false for expired record")
	}
	if !status.CanApply {
		t.Fatalf("expected CanApply=true when expired (not rejected)")
	}
	assertOpaqueShareCode(t, status.AccessCode, "user-7")
}

func TestGetReferralCodeStatusNone(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	status, err := getReferralCodeStatus("user-8")
	if err != nil {
		t.Fatalf("get status failed: %v", err)
	}
	if status.ApplicationStatus != nil {
		t.Fatalf("expected nil status, got %s", *status.ApplicationStatus)
	}
	if !status.CanApply {
		t.Fatalf("expected CanApply=true with no application")
	}
	if status.HasActiveCode {
		t.Fatalf("expected HasActiveCode=false with no application")
	}
	assertOpaqueShareCode(t, status.AccessCode, "user-8")
	if status.ReferralRateDisplay != "5%" {
		t.Fatalf("default test stub rate want 5%%, got %q", status.ReferralRateDisplay)
	}
}

func TestGetReferralCodeStatus_IgnoresWechatRate(t *testing.T) {
	setupTestReferralDB(t)

	status, err := getReferralCodeStatus("user-rate")
	if err != nil {
		t.Fatalf("get status failed: %v", err)
	}
	if status.ReferralRateDisplay != "5%" {
		t.Fatalf("expected fixed policy 5%%, got %q", status.ReferralRateDisplay)
	}
}

// ── approve / reject tests ──

func TestApproveReferralApplication(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	result, err := applyReferralWithConsent("user-9", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "pending" {
		t.Fatalf("expected pending, got %s", result.Status)
	}

	// Find the pending application ID
	app := getPendingReferralCode("user-9")
	if app == nil {
		t.Fatalf("pending app not found")
	}

	res, err := approveReferralApplication(context.Background(), app.ID, "admin-1", "申请人介绍充分予以通过", "")
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if res["status"] != "approved" {
		t.Fatalf("expected approved, got %v", res["status"])
	}
	code, _ := res["referral_code"].(string)
	assertOpaqueShareCode(t, code, "user-9")
}

func TestRejectReferralApplication(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	result, err := applyReferralWithConsent("user-10", testValidPersonalIntro)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "pending" {
		t.Fatalf("expected pending, got %s", result.Status)
	}

	app := getPendingReferralCode("user-10")
	if app == nil {
		t.Fatalf("pending app not found")
	}

	res, err := rejectReferralApplication(context.Background(), app.ID, "admin-1", "不符合推荐人准入标准", "")
	if err != nil {
		t.Fatalf("reject failed: %v", err)
	}
	if res["status"] != "rejected" {
		t.Fatalf("expected rejected, got %v", res["status"])
	}

	// Verify cooldown
	status, err := getReferralCodeStatus("user-10")
	if err != nil {
		t.Fatalf("get status failed: %v", err)
	}
	if status.ApplicationStatus == nil || *status.ApplicationStatus != "rejected" {
		t.Fatalf("expected rejected status, got %v", status.ApplicationStatus)
	}
}

// ── expireReferralCodes test ──

func TestExpireReferralCodes(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	// Active (future expiry) — should NOT be expired
	insertApprovedReferral(t, "user-fresh", time.Now().AddDate(0, 3, 0))
	// Expired (past expiry) — SHOULD be expired
	insertApprovedReferral(t, "user-old", time.Now().AddDate(0, -1, 0))
	// Pending — should NOT be touched
	_, _ = applyReferralWithConsent("user-pending", testValidPersonalIntro)

	expireReferralCodes()

	// Verify user-old is now expired
	var status string
	err := db.QueryRow(`
		SELECT status FROM referral_code WHERE user_id = ?`, "user-old").Scan(&status)
	if err != nil {
		t.Fatalf("query user-old: %v", err)
	}
	if status != "expired" {
		t.Fatalf("expected expired, got %s", status)
	}

	// Verify user-fresh is still approved
	err = db.QueryRow(`
		SELECT status FROM referral_code WHERE user_id = ?`, "user-fresh").Scan(&status)
	if err != nil {
		t.Fatalf("query user-fresh: %v", err)
	}
	if status != "approved" {
		t.Fatalf("expected approved, got %s", status)
	}

	// Verify user-pending is still pending
	err = db.QueryRow(`
		SELECT status FROM referral_code WHERE user_id = ?`, "user-pending").Scan(&status)
	if err != nil {
		t.Fatalf("query user-pending: %v", err)
	}
	if status != "pending" {
		t.Fatalf("expected pending, got %s", status)
	}
}

// ── listReferralApplications test ──

func TestListReferralApplications(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	_, _ = applyReferralWithConsent("user-a", testValidPersonalIntro)
	_, _ = applyReferralWithConsent("user-b", testValidPersonalIntro)
	insertApprovedReferral(t, "user-c", time.Now().AddDate(0, 6, 0))

	items, total, err := listReferralApplications("pending", 50, 0)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 pending, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// List all
	items, total, err = listReferralApplications("", 50, 0)
	if err != nil {
		t.Fatalf("list all failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 total, got %d", total)
	}
}

// OPT-20260824-006: 申请列表每行水合该用户默认推荐码。
func TestListReferralApplicationsHydratesShareCodes(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	users := []string{"user-code-a", "user-code-b"}
	for _, u := range users {
		if _, err := applyReferralWithConsent(u, testValidPersonalIntro); err != nil {
			t.Fatalf("apply %s: %v", u, err)
		}
		code, err := ensureUserShareCode(u)
		if err != nil {
			t.Fatalf("ensure share code %s: %v", u, err)
		}
		if code == "" {
			t.Fatalf("share code for %s empty", u)
		}
	}

	items, _, err := listReferralApplications("pending", 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items=%d want 2", len(items))
	}
	byUser := map[string]referralCodeRow{}
	for _, it := range items {
		byUser[it.UserID] = it
	}
	for _, u := range users {
		it := byUser[u]
		if it.ShareCode == "" {
			t.Fatalf("share_code for %s empty, want hydrated default code", u)
		}
		want, _ := lookupShareCodeByUser(u)
		if it.ShareCode != want {
			t.Fatalf("share_code=%q want %q", it.ShareCode, want)
		}
	}
}

// 无默认分享码的用户 share_code 应为空且列表不报错。
func TestListReferralApplicationsEmptyShareCode(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")
	if _, err := applyReferralWithConsent("user-no-code", testValidPersonalIntro); err != nil {
		t.Fatalf("apply: %v", err)
	}
	items, _, err := listReferralApplications("pending", 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].ShareCode != "" {
		t.Fatalf("share_code=%q want empty for user without default code", items[0].ShareCode)
	}
}

func ensureBillingReferralConfigTable(t *testing.T) {
	t.Helper()
	billDB = db
	_, err := billDB.Exec(`
		CREATE TABLE IF NOT EXISTS billing_referral_config (
		  singleton_key VARCHAR(64) NOT NULL PRIMARY KEY,
		  settle_delay_days INTEGER NOT NULL DEFAULT 15,
		  profit_sharing_ratio_percent INTEGER NOT NULL DEFAULT 30,
		  referral_rate_percent INTEGER NOT NULL DEFAULT 5,
		  updated_at VARCHAR(255) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		t.Fatalf("create billing_referral_config: %v", err)
	}
}

func TestListReferralApplicationsIncludesConfiguredRatios(t *testing.T) {
	setupTestReferralDB(t)
	ensureBillingReferralConfigTable(t)
	setReferralPolicyForTest(t, "approval", "")
	if _, err := updateReferralSettleConfig(15, 12); err != nil {
		t.Fatalf("seed ratios: %v", err)
	}
	_, _ = applyReferralWithConsent("user-ratio-1", testValidPersonalIntro)

	items, _, err := listReferralApplications("pending", 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least 1 item")
	}
	if items[0].ProfitSharingRatioDisplay != "5%" {
		t.Fatalf("profit_sharing_ratio_display=%q want 5%%", items[0].ProfitSharingRatioDisplay)
	}
	if items[0].ReferralRateDisplay != "5%" {
		t.Fatalf("referral_rate_display=%q want 5%%", items[0].ReferralRateDisplay)
	}
}

func TestListReferralApplicationsUsesGlobalSingleRate(t *testing.T) {
	setupTestReferralDB(t)
	ensureBillingReferralConfigTable(t)
	setReferralPolicyForTest(t, "approval", "")
	if _, err := updateReferralSettleConfig(15, 12); err != nil {
		t.Fatalf("seed global ratio: %v", err)
	}
	_, _ = applyReferralWithConsent("user-ratio-a", testValidPersonalIntro)
	_, _ = applyReferralWithConsent("user-ratio-b", testValidPersonalIntro)

	items, _, err := listReferralApplications("pending", 50, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) < 2 {
		t.Fatalf("items=%d want >=2", len(items))
	}
	for _, it := range items {
		if it.ReferralRateDisplay != "5%" || it.ProfitSharingRatioDisplay != "5%" {
			t.Fatalf("user=%s display=%s/%s want fixed 5%%", it.UserID, it.ProfitSharingRatioDisplay, it.ReferralRateDisplay)
		}
	}
}

func TestUpdateReferralSettleConfigIgnoresRate(t *testing.T) {
	setupTestReferralDB(t)
	ensureBillingReferralConfigTable(t)
	cfg, err := updateReferralSettleConfig(15, 4)
	if err != nil {
		t.Fatalf("update with out-of-range rate should ignore rate: %v", err)
	}
	if cfg.ReferralRatePercent != 5 {
		t.Fatalf("rate=%d want fixed 5", cfg.ReferralRatePercent)
	}
	cfg, err = updateReferralSettleConfig(15, 31)
	if err != nil {
		t.Fatalf("update with 31 should ignore rate: %v", err)
	}
	if cfg.ReferralRatePercent != 5 {
		t.Fatalf("rate=%d want fixed 5", cfg.ReferralRatePercent)
	}
}
