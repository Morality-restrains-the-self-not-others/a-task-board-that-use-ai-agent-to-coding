package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

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

func handleAdminReferralConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := getReferralConfig()
		if err != nil {
			log.Printf("[taskBill] referral config get: %v", err)
			writeErrorJSON(w, http.StatusInternalServerError, "db error", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":             cfg,
			"settle_delay_min": referralSettleDelayMin,
		})

	case http.MethodPut, http.MethodPost:
		body, err := readJSONBody(r)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		current, err := getReferralConfig()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "db error", tracelog.TraceIDFromContext(r.Context()))
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
			tracelog.EmitWithTrace(
				tracelog.TraceIDFromContext(r.Context()),
				"info",
				"referral rate in request ignored; policy is fixed 5 percent",
				"referral_config",
				map[string]string{"ignored_field": "referral_rate_percent"},
			)
		}
		if _, ok := jsonBodyInt(body, "profit_sharing_ratio_percent"); ok {
			tracelog.EmitWithTrace(
				tracelog.TraceIDFromContext(r.Context()),
				"info",
				"referral rate in request ignored; policy is fixed 5 percent",
				"referral_config",
				map[string]string{"ignored_field": "profit_sharing_ratio_percent"},
			)
		}
		cfg, err := updateReferralConfig(days, int(referralCommissionRateNum))
		if err != nil {
			log.Printf("[taskBill] referral config update: %v", err)
			writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
		tracelog.EmitWithTrace(
			tracelog.TraceIDFromContext(r.Context()),
			"info",
			"referral settle delay updated",
			"referral_config",
			map[string]string{
				"settle_delay_days":     strconv.Itoa(cfg.SettleDelayDays),
				"referral_rate_percent": strconv.Itoa(int(referralCommissionRateNum)),
			},
		)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":             cfg,
			"settle_delay_min": referralSettleDelayMin,
		})

	default:
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
	}
}
