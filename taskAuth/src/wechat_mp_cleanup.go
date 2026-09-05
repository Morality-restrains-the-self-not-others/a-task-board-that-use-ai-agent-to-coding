package main

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tracelog"
)

// cleanupWechatMpStaleRows 分批删除微信服务号挂起与票据表中的过期行（OPT-20260826-003）。
//
// auth_wechat_mp_subscribe_pending：unionid 未对应平台用户的挂起行，用户绑定后即删除；
// 残留超过 pendingMaxAgeDays（默认 30d）的视为无法认领，按批清理。
// auth_wechat_mp_follow_ticket：动态 scene 票据，expire_at 超过 ticketMaxAgeDays（默认 7d）
// 的票已彻底过期，按批清理。禁止业务进程内 ticker（元规则 51），由 taskEvents timer
// 周期调用内部端点执行。返回 (pendingDeleted, ticketDeleted, err)。
func cleanupWechatMpStaleRows(pendingMaxAgeDays int, ticketMaxAgeDays int, limit int64) (int64, int64, error) {
	if pendingMaxAgeDays <= 0 {
		pendingMaxAgeDays = 30
	}
	if ticketMaxAgeDays <= 0 {
		ticketMaxAgeDays = 7
	}
	if limit <= 0 {
		limit = 500
	}
	now := time.Now().UTC()
	pendingCutoff := now.AddDate(0, 0, -pendingMaxAgeDays).Format("2006-01-02 15:04:05.000000")
	ticketCutoff := now.AddDate(0, 0, -ticketMaxAgeDays).Format("2006-01-02 15:04:05.000000")

	var pendingTotal, ticketTotal int64

	// 挂起行：created_at 超期未认领（绑定成功即删除，残留即无法认领）。
	for {
		res, err := db.Exec(`DELETE FROM auth_wechat_mp_subscribe_pending WHERE created_at < ? LIMIT ?`, pendingCutoff, limit)
		if err != nil {
			return pendingTotal, ticketTotal, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return pendingTotal, ticketTotal, err
		}
		pendingTotal += n
		if n < limit {
			break
		}
	}

	// 票据行：expire_at 已过期超过 ticketMaxAgeDays。
	for {
		res, err := db.Exec(`DELETE FROM auth_wechat_mp_follow_ticket WHERE expire_at < ? LIMIT ?`, ticketCutoff, limit)
		if err != nil {
			return pendingTotal, ticketTotal, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return pendingTotal, ticketTotal, err
		}
		ticketTotal += n
		if n < limit {
			break
		}
	}
	return pendingTotal, ticketTotal, nil
}

// handleInternalWechatMpCleanup POST /api/internal/taskauth/wechat-mp-cleanup/
// 由 taskEvents timer（wechat_mp_cleanup）周期调用，分批删除过期挂起/票据行。
func handleInternalWechatMpCleanup(w http.ResponseWriter, r *http.Request) {
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pendingMaxAgeDays := 30
	if raw := strings.TrimSpace(r.URL.Query().Get("pending_max_age_days")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			pendingMaxAgeDays = v
		}
	}
	ticketMaxAgeDays := 7
	if raw := strings.TrimSpace(r.URL.Query().Get("ticket_max_age_days")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			ticketMaxAgeDays = v
		}
	}
	limit := int64(500)
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			limit = v
		}
	}
	pendingDeleted, ticketDeleted, err := cleanupWechatMpStaleRows(pendingMaxAgeDays, ticketMaxAgeDays, limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "wechat_mp_cleanup",
			"error", err.Error(), "trace_id", tracelog.TraceIDFromContext(r.Context()))
		writeErrorDetail(w, r, http.StatusInternalServerError, "cleanup failed")
		return
	}
	slog.InfoContext(r.Context(), "wechat_mp_cleanup_ok",
		"pending_deleted", pendingDeleted, "ticket_deleted", ticketDeleted,
		"trace_id", tracelog.TraceIDFromContext(r.Context()))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pending_deleted": pendingDeleted,
		"ticket_deleted":  ticketDeleted,
	})
}
