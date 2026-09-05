package main

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"authz"
	"tracelog"
)

const adminRefreshProfitSharingMaxIDs = 50

// handleSystemAdminRefreshProfitSharingWechat POST /api/system-admin/profit-sharing/refresh-wechat/
// 对推荐人图内台账逐笔 QueryOrder；不回写本地 status，响应不含 openid。
func handleSystemAdminRefreshProfitSharingWechat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if strings.TrimSpace(r.Header.Get("X-Gateway-Auth-Verified")) != "1" {
		slog.WarnContext(r.Context(), "admin_profit_sharing_refresh_unauthorized",
			"level", "warn",
			"reason", "missing_gateway_verify",
		)
		writeErrorJSON(w, http.StatusUnauthorized, "authentication required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !authz.IsPlatformStaff(r) {
		slog.WarnContext(r.Context(), "admin_profit_sharing_refresh_forbidden",
			"level", "warn",
			"reason", "not_platform_staff",
		)
		writeErrorJSON(w, http.StatusForbidden, "superuser or staff required", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	referrerUserID := strings.TrimSpace(stringField(body, "referrer_user_id"))
	if referrerUserID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "referrer_user_id required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	rawIDs, _ := body["ids"].([]interface{})
	if len(rawIDs) == 0 {
		writeErrorJSON(w, http.StatusBadRequest, "ids required", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if len(rawIDs) > adminRefreshProfitSharingMaxIDs {
		writeErrorJSON(w, http.StatusBadRequest, "ids exceeds limit", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	ids := make([]int64, 0, len(rawIDs))
	seen := make(map[int64]struct{}, len(rawIDs))
	for _, raw := range rawIDs {
		id, err := parseIDField(raw)
		if err != nil || id <= 0 {
			writeErrorJSON(w, http.StatusBadRequest, "invalid id", tracelog.TraceIDFromContext(r.Context()))
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	inGraph, err := profitSharingIDsInReferrerGraph(referrerUserID, ids)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_profit_sharing_refresh_graph_failed",
			"level", "error",
			"error", err.Error(),
		)
		writeErrorJSON(w, http.StatusInternalServerError, "校验分账记录失败", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if len(inGraph) != len(ids) {
		slog.WarnContext(r.Context(), "admin_profit_sharing_refresh_foreign_ids",
			"level", "warn",
			"reason", "ids_not_in_referrer_graph",
			"referrer_user_id_len", len(referrerUserID),
			"requested", len(ids),
			"in_graph", len(inGraph),
		)
		writeErrorJSON(w, http.StatusBadRequest, "ids must belong to referrer graph", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	started := time.Now()
	items := refreshProfitSharingWechatStates(r, ids)
	slog.InfoContext(r.Context(), "admin_profit_sharing_refresh_ok",
		"level", "info",
		"referrer_user_id_len", len(referrerUserID),
		"count", len(items),
		"duration_ms", time.Since(started).Milliseconds(),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": items})
}
