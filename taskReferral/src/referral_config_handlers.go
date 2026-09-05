package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

const referralConfigKey = "global"
const referralSettleDelayMin = 8
const defaultReferralRatePercent = 5
const referralRatioRangeDisplay = "5%"

type referralConfigRow struct {
	SettleDelayDays           int    `json:"settle_delay_days"`
	ProfitSharingRatioPercent int    `json:"profit_sharing_ratio_percent"`
	ReferralRatePercent       int    `json:"referral_rate_percent"`
	ProfitSharingRatioDisplay string `json:"profit_sharing_ratio_display"`
	ReferralRateDisplay       string `json:"referral_rate_display"`
	RatioRangeDisplay         string `json:"ratio_range_display"`
	UpdatedAt                 string `json:"updated_at"`
}

func defaultReferralConfigRow() referralConfigRow {
	return decorateReferralConfigRow(referralConfigRow{
		SettleDelayDays:           15,
		ProfitSharingRatioPercent: defaultReferralRatePercent,
		ReferralRatePercent:       defaultReferralRatePercent,
	})
}

func decorateReferralConfigRow(cfg referralConfigRow) referralConfigRow {
	rate := defaultReferralRatePercent
	cfg.ReferralRatePercent = rate
	cfg.ProfitSharingRatioPercent = rate
	cfg.ProfitSharingRatioDisplay = formatPercentDisplay(rate)
	cfg.ReferralRateDisplay = formatPercentDisplay(rate)
	cfg.RatioRangeDisplay = referralRatioRangeDisplay
	return cfg
}

func formatPercentDisplay(n int) string {
	if n < 1 {
		return ""
	}
	return fmt.Sprintf("%d%%", n)
}

func jsonBodyInt(body map[string]interface{}, key string) (int, bool) {
	v, ok := body[key]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return int(t), true
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func getReferralSettleConfig() (referralConfigRow, error) {
	if billDB == nil {
		return referralConfigRow{}, fmt.Errorf("billDB unavailable")
	}
	var cfg referralConfigRow
	err := billDB.QueryRow(`
		SELECT settle_delay_days, updated_at
		FROM billing_referral_config WHERE singleton_key = ?`, referralConfigKey,
	).Scan(&cfg.SettleDelayDays, &cfg.UpdatedAt)
	if err == sql.ErrNoRows {
		return defaultReferralConfigRow(), nil
	}
	if err != nil {
		return referralConfigRow{}, err
	}
	return decorateReferralConfigRow(cfg), nil
}

func updateReferralSettleConfig(days, _ int) (referralConfigRow, error) {
	if days < referralSettleDelayMin {
		return referralConfigRow{}, fmt.Errorf("settle_delay_days must be >= %d", referralSettleDelayMin)
	}
	if billDB == nil {
		return referralConfigRow{}, fmt.Errorf("billDB unavailable")
	}
	now := timeNowUTC()
	// OPT-20260825-012: 分成比例已常量化 5%，不再写 billing_referral_config 比例列（列将随 071 迁移删除）。
	_, err := billDB.Exec(`
		INSERT INTO billing_referral_config (
			singleton_key, settle_delay_days, updated_at
		) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			settle_delay_days = VALUES(settle_delay_days),
			updated_at = VALUES(updated_at)`,
		referralConfigKey, days, now)
	if err != nil {
		return referralConfigRow{}, err
	}
	return getReferralSettleConfig()
}

func attachReferralRatioDisplays(items []referralCodeRow) {
	display := formatPercentDisplay(defaultReferralRatePercent)
	for i := range items {
		items[i].ProfitSharingRatioDisplay = display
		items[i].ReferralRateDisplay = display
	}
}

func configuredReferralRateDisplay() string {
	return formatPercentDisplay(defaultReferralRatePercent)
}

func handleAdminReferralConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if _, ok := requireSuperuser(w, r); !ok {
			return
		}
		cfg, err := getReferralSettleConfig()
		if err != nil {
			log.Printf("[taskReferral] referral config get: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":             cfg,
			"settle_delay_min": referralSettleDelayMin,
		})

	case http.MethodPut, http.MethodPost:
		if _, ok := requireSuperuser(w, r); !ok {
			return
		}
		body, err := readJSONBody(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		current, err := getReferralSettleConfig()
		if err != nil {
			log.Printf("[taskReferral] referral config get for update: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
			return
		}
		days := current.SettleDelayDays
		if v, ok := jsonBodyInt(body, "settle_delay_days"); ok {
			days = v
		}
		if days < referralSettleDelayMin {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "settle_delay_days must be an integer >= " + strconv.Itoa(referralSettleDelayMin),
			})
			return
		}
		if _, ok := jsonBodyInt(body, "referral_rate_percent"); ok {
			log.Printf("[taskReferral] referral rate in request ignored; policy is fixed 5 percent trace_id=%s",
				tracelog.TraceIDFromContext(r.Context()))
		}
		cfg, err := updateReferralSettleConfig(days, defaultReferralRatePercent)
		if err != nil {
			log.Printf("[taskReferral] referral config update: %v", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		log.Printf("[taskReferral] event=REFERRAL_SETTLE_DELAY_UPDATED settle_delay_days=%d referral_rate_percent=5 trace_id=%s",
			cfg.SettleDelayDays, tracelog.TraceIDFromContext(r.Context()))
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":             cfg,
			"settle_delay_min": referralSettleDelayMin,
		})

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed"})
	}
}
