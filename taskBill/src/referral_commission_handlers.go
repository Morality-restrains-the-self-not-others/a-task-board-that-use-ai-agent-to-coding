package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tracelog"
)

func handleInternalReferralSyncEdge(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	referrer := strings.TrimSpace(stringField(body, "referrer_user_id"))
	referred := strings.TrimSpace(stringField(body, "referred_user_id"))
	boundAt := strings.TrimSpace(stringField(body, "bound_at"))
	channelCode := strings.TrimSpace(stringField(body, "channel_code"))
	eligible := boolField(body, "commission_eligible")
	tenantID, err := parseIDField(body["referrer_tenant_id"])
	if err != nil || tenantID <= 0 {
		writeErrorJSON(w, http.StatusBadRequest, "invalid referrer_tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err := upsertReferralEdge(referrer, referred, tenantID, boundAt, channelCode, eligible); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	backfilled, err := backfillReferralAccrualsForReferredUser(r.Context(), referred)
	if err != nil {
		slog.Warn("referral backfill failed", "error", err.Error(), "user_id", referred)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"backfilled": strconv.Itoa(backfilled),
	})
}

func handleInternalReferralCommissionSummary(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	referrer := strings.TrimSpace(r.URL.Query().Get("referrer_user_id"))
	if referrer == "" && r.Method == http.MethodPost {
		body, _ := readJSONBody(r)
		referrer = strings.TrimSpace(stringField(body, "referrer_user_id"))
	}
	if referrer == "" {
		writeErrorJSON(w, http.StatusBadRequest, "referrer_user_id required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	out, err := referralCommissionSummary(referrer)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func handleInternalReferralSettleDue(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	now := time.Now().UTC()
	body, _ := readJSONBody(r)
	if v := strings.TrimSpace(stringField(body, "now")); v != "" {
		if t, err := parseBillingTime(v); err == nil {
			now = t
		}
	}
	n, err := settleDueReferralCommissions(r.Context(), now)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"settled": strconv.Itoa(n),
		"status":  "ok",
	})
}

func handleInternalReferralVoid(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	sourceTxn := strings.TrimSpace(stringField(body, "source_txn_id"))
	reason := strings.TrimSpace(stringField(body, "reason"))
	if reason == "" {
		reason = "void"
	}
	if err := voidReferralAccrualBySourceTxn(sourceTxn, reason); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func handleInternalReferralConsumptionMonthly(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	rawIDs, _ := body["user_ids"].([]interface{})
	userIDs := make([]string, 0, len(rawIDs))
	for _, v := range rawIDs {
		switch t := v.(type) {
		case string:
			if s := strings.TrimSpace(t); s != "" {
				userIDs = append(userIDs, s)
			}
		case json.Number:
			userIDs = append(userIDs, t.String())
		default:
			s := strings.TrimSpace(fmt.Sprintf("%v", v))
			if s != "" && s != "<nil>" {
				userIDs = append(userIDs, s)
			}
		}
	}
	totals, err := consumptionMonthlyTotals(userIDs)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"totals": totals,
	})
}
