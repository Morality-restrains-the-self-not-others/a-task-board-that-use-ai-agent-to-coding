package main

import (
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

// handleInternalGetAccount — 只读查询 BillingAccount。创建由 taskEvents 消费者负责。
func handleInternalGetAccount(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/taskbill/accounts/")
	path = strings.TrimSuffix(path, "/")
	tid, err := strconv.ParseInt(path, 10, 64)
	if err != nil || tid <= 0 {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	acc, err := getBillingAccount(tid)
	if err == ErrAccountNotFound {
		writeErrorJSON(w, http.StatusNotFound, "account not found", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	taskPrice, _ := getUnitPriceCents(ResourceTypeTaskPost)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"account":         accountJSON(acc),
		"task_post_price": taskPrice,
		"balance":         acc.Balance,
	})
}

func handleInternalGetOrCreateAccount(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	acc, created, err := getOrCreateBillingAccount(tid, false)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	taskPrice, _ := getUnitPriceCents(ResourceTypeTaskPost)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"account": accountJSON(acc), "created": created,
		"task_post_price": taskPrice,
		"balance":         acc.Balance,
	})
}

func handleInternalCheckServerStartBalance(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body, _ := readJSONBody(r)
	tid, err := parseIDField(body["tenant_id"])
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if err := checkBalanceForServerStart(tid); err != nil {
		if ib, ok := err.(*InsufficientBalanceError); ok {
			writeJSON(w, http.StatusPaymentRequired, map[string]interface{}{
				"error": ib.Error(), "balance_points": ib.BalancePoints, "required_points": ib.RequiredPoints,
			})
			return
		}
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
