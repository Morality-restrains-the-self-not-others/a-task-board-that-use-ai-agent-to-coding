package main

import (
	"database/sql"
	"fmt"
)

const referralConfigKey = "global"
const referralSettleDelayMin = 8
const referralRatioRangeDisplay = "5%"

type referralConfig struct {
	SettleDelayDays           int    `json:"settle_delay_days"`
	ProfitSharingRatioPercent int    `json:"profit_sharing_ratio_percent"`
	ReferralRatePercent       int    `json:"referral_rate_percent"`
	ProfitSharingRatioDisplay string `json:"profit_sharing_ratio_display"`
	ReferralRateDisplay       string `json:"referral_rate_display"`
	RatioRangeDisplay         string `json:"ratio_range_display"`
	UpdatedAt                 string `json:"updated_at"`
}

func defaultReferralConfig() referralConfig {
	cfg := referralConfig{
		SettleDelayDays: referralSettleDelayDays,
	}
	return decorateReferralConfig(cfg)
}

func decorateReferralConfig(cfg referralConfig) referralConfig {
	rate := int(referralCommissionRateNum)
	cfg.ReferralRatePercent = rate
	cfg.ProfitSharingRatioPercent = rate
	cfg.ProfitSharingRatioDisplay = formatCommissionRateDisplay(int64(rate))
	cfg.ReferralRateDisplay = formatCommissionRateDisplay(int64(rate))
	cfg.RatioRangeDisplay = referralRatioRangeDisplay
	return cfg
}

func getReferralConfig() (referralConfig, error) {
	if db == nil {
		return defaultReferralConfig(), nil
	}
	var cfg referralConfig
	err := db.QueryRow(`
		SELECT settle_delay_days, updated_at
		FROM billing_referral_config WHERE singleton_key = ?`, referralConfigKey,
	).Scan(&cfg.SettleDelayDays, &cfg.UpdatedAt)
	if err == sql.ErrNoRows {
		return defaultReferralConfig(), nil
	}
	if err != nil {
		return referralConfig{}, err
	}
	return decorateReferralConfig(cfg), nil
}

func getReferralSettleDelayDays() int {
	cfg, err := getReferralConfig()
	if err != nil || cfg.SettleDelayDays < referralSettleDelayMin {
		return referralSettleDelayDays
	}
	return cfg.SettleDelayDays
}

func getProfitSharingRatioPercent() int64 {
	return getReferralRatePercent()
}

// getReferralRatePercent 分成比例固定 5%，禁止从 billing_referral_config 读取。
func getReferralRatePercent() int64 {
	return referralCommissionRateNum
}

func updateReferralConfig(settleDelayDays, _ int) (referralConfig, error) {
	if settleDelayDays < referralSettleDelayMin {
		return referralConfig{}, fmt.Errorf("settle_delay_days must be >= %d", referralSettleDelayMin)
	}
	now := utcNow()
	// OPT-20260825-012: 分成比例已常量化 5%，不再写 billing_referral_config 比例列（列将随 071 迁移删除）。
	_, err := db.Exec(`
		INSERT INTO billing_referral_config (
			singleton_key, settle_delay_days, updated_at
		) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			settle_delay_days = VALUES(settle_delay_days),
			updated_at = VALUES(updated_at)`,
		referralConfigKey, settleDelayDays, now)
	if err != nil {
		return referralConfig{}, err
	}
	return getReferralConfig()
}
