package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReferralConfigDefaults(t *testing.T) {
	cfg := defaultReferralConfig()
	if cfg.ProfitSharingRatioPercent != 5 || cfg.ReferralRatePercent != 5 {
		t.Fatalf("defaults=%+v want single rate 5", cfg)
	}
	if cfg.ProfitSharingRatioDisplay != "5%" || cfg.ReferralRateDisplay != "5%" {
		t.Fatalf("display=%s/%s", cfg.ProfitSharingRatioDisplay, cfg.ReferralRateDisplay)
	}
	if cfg.RatioRangeDisplay != "5%" {
		t.Fatalf("range=%s want 5%%", cfg.RatioRangeDisplay)
	}
}

func TestGetReferralRatePercentIgnoresStoredConfig(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	if _, err := updateReferralConfig(15, 12); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := getReferralRatePercent(); got != 5 {
		t.Fatalf("getReferralRatePercent=%d want 5 (never from config)", got)
	}
	got, err := getReferralConfig()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ProfitSharingRatioPercent != 5 || got.ReferralRatePercent != 5 {
		t.Fatalf("reload=%+v want 5 even after writing 12", got)
	}
}

// OPT-20260825-012 回归：billing_referral_config 比例列已随 071 迁移删除，
// updateReferralConfig 不再引用这两列仍可正常更新。
func TestUpdateReferralConfigWithoutRatioColumns(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	var ratioCols int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = 'billing_referral_config'
		  AND COLUMN_NAME IN ('profit_sharing_ratio_percent', 'referral_rate_percent')`,
	).Scan(&ratioCols); err != nil {
		t.Fatalf("check ratio columns: %v", err)
	}
	if ratioCols != 0 {
		t.Fatalf("ratio columns still exist (count=%d); 071 migration must drop them", ratioCols)
	}
	if _, err := updateReferralConfig(15, 12); err != nil {
		t.Fatalf("update without ratio columns: %v", err)
	}
	got, err := getReferralConfig()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SettleDelayDays != 15 {
		t.Fatalf("settle_delay_days=%d want 15", got.SettleDelayDays)
	}
}

func TestHandleAdminReferralConfigIgnoresRateBody(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)

	body := `{"referral_rate_percent":18}`
	req := httptest.NewRequest(http.MethodPost, "/api/internal/taskbill/referral/config/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleAdminReferralConfig(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data referralConfig `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.Data.ProfitSharingRatioPercent != 5 || out.Data.ReferralRatePercent != 5 {
		t.Fatalf("data=%+v want fixed 5", out.Data)
	}
	if out.Data.RatioRangeDisplay != "5%" {
		t.Fatalf("range=%s", out.Data.RatioRangeDisplay)
	}
}

func TestReferralCommissionPointsIgnoresStoredRate(t *testing.T) {
	cleanup := setupMySQLTestDB(t)
	t.Cleanup(cleanup)
	if _, err := updateReferralConfig(15, 10); err != nil {
		t.Fatalf("update: %v", err)
	}
	if got := referralCommissionPoints(10000); got != 500 {
		t.Fatalf("10000 @fixed 5%% -> %d want 500", got)
	}
}
