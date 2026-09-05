package main

import (
	"net/http"

	"tracelog"
)

func handleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
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
	out := accountJSON(acc)
	if enabled, err := getRefundPolicyEnabled(r.Context()); err == nil {
		out["refund_enabled"] = enabled
	} else {
		out["refund_enabled"] = true
	}
	lots, err := listActiveCreditLots(r.Context(), tid)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	out["lots"] = lots
	writeJSON(w, http.StatusOK, out)
}

func handleUnitsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	rows, err := db.Query(`SELECT id, unit_type, name, price, unit, is_active, created_at, updated_at FROM billing_unit`)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id, price int64
		var unitType, name, unit string
		var active bool
		var created, updated string
		if err := rows.Scan(&id, &unitType, &name, &price, &unit, &active, &created, &updated); err != nil {
			continue
		}
		list = append(list, map[string]interface{}{
			"id": formatID(id), "unit_type": unitType, "name": name, "price_points": price,
			"unit": unit, "is_active": active, "created_at": created, "updated_at": updated,
		})
	}
	writeJSON(w, http.StatusOK, list)
}
