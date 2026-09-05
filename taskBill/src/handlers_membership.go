package main

import (
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

// handleMembership 查询租户会员等级（对外 API，需认证上下文中的 tenant_id）
func handleMembership(w http.ResponseWriter, r *http.Request) {
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	m, err := getMembership(tid)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"membership": membershipJSON(m),
	})
}

// handleInternalGetMembership 查询租户会员等级（内部 API）
func handleInternalGetMembership(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorJSON(w, http.StatusForbidden, "forbidden", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/internal/taskbill/membership/")
	path = strings.TrimSuffix(path, "/")
	tid, err := strconv.ParseInt(path, 10, 64)
	if err != nil || tid <= 0 {
		writeErrorJSON(w, http.StatusBadRequest, "invalid tenant_id", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	m, err := getMembership(tid)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"membership": membershipJSON(m),
	})
}

// handleInternalEnsureMembership 确保租户有会员记录（内部 API，幂等）
func handleInternalEnsureMembership(w http.ResponseWriter, r *http.Request) {
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

	m, err := ensureMembership(tid)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, err.Error(), tracelog.TraceIDFromContext(r.Context()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"membership": membershipJSON(m),
	})
}
