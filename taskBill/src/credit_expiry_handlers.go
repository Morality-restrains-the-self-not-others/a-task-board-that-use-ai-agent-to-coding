package main

import (
	"net/http"
	"strconv"
	"strings"
	"tracelog"
)

func handleInternalCreditLots(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	qp := r.URL.Query()
	var tenantID int64
	if v := strings.TrimSpace(qp.Get("tenant_id")); v != "" {
		tid, err := strconv.ParseInt(v, 10, 64)
		if err != nil || tid <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		tenantID = tid
		if err := expireCreditLots(r.Context(), tenantID); err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
			return
		}
	}
	lots, err := listCreditLotsFiltered(r.Context(), tenantID, qp.Get("user_id"))
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"lots": lots})
}

func handleInternalUserRecharges(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	qp := r.URL.Query()
	userID := strings.TrimSpace(qp.Get("user_id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "user_id is required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	var tenantID int64
	if v := strings.TrimSpace(qp.Get("tenant_id")); v != "" {
		tid, err := strconv.ParseInt(v, 10, 64)
		if err != nil || tid <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		tenantID = tid
	}
	list, err := listUserRecharges(r.Context(), userID, tenantID)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"recharges": list})
}

func handleInternalExpireCreditLots(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err := expireCreditLots(r.Context(), tid); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	acc, _, err := getOrCreateBillingAccount(tid, false)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"balance": acc.Balance,
	})
}
