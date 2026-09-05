package main

// 推荐人本人分账：渠道聚合列表 + 手动分账。
//
//	GET  /api/billing/profit-sharing/referrer-orders/                 — 按渠道聚合（无订单号）
//	POST /api/billing/profit-sharing/referrer-orders/share-channel/   — 对该渠道可分账行批量发起
//	POST /api/billing/profit-sharing/referrer-orders/{id}/share/      — 单行（列表不再暴露 id）
//
// 约束：仅推荐人本人；禁止返回 openid / 微信分账单号 / 受推荐人订单号。

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

const referrerProfitSharingPathPrefix = "/api/billing/profit-sharing/referrer-orders/"

func handleReferrerProfitSharingOrders(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "missing user identity", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, referrerProfitSharingPathPrefix), "/")
	if r.Method == http.MethodPost && rest == "share-channel" {
		handleReferrerShareChannel(w, r, userID)
		return
	}
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/share/") {
		idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, referrerProfitSharingPathPrefix), "/share/")
		idStr = strings.Trim(idStr, "/")
		handleReferrerShareProfitSharing(w, r, userID, idStr)
		return
	}
	handleReferrerProfitSharingList(w, r, userID)
}

func handleReferrerShareProfitSharing(w http.ResponseWriter, r *http.Request, userID, idStr string) {
	id, err := parseIDField(idStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid profit sharing id", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	code, state, idempotent, errMsg := shareReferrerProfitSharingRecord(r, userID, id, time.Now().UTC())
	if code != http.StatusOK {
		writeErrorJSON(w, code, errMsg, tracelog.TraceIDFromContext(r.Context()))
		return
	}
	body := map[string]interface{}{"status": "ok", "state": state}
	if idempotent {
		body["idempotent"] = true
	}
	writeJSON(w, http.StatusOK, body)
}

// shareReferrerProfitSharingRecord 对单行分账：200/404/409/502/500。idempotent 表示 processing/finished 重放。
func shareReferrerProfitSharingRecord(r *http.Request, userID string, id int64, now time.Time) (int, string, bool, string) {
	var rec profitSharingRecord
	var paidAt string
	err := db.QueryRow(`
		SELECT ps.id, ps.out_profit_sharing_no, ps.order_id, ps.order_number, ps.tenant_id,
		       ps.referrer_user_id, ps.referrer_openid,
		       ps.commission_yuan_cents, ps.total_yuan_cents,
		       ps.status, COALESCE(ps.fail_reason, ''), o.paid_at
		FROM billing_profit_sharing ps
		INNER JOIN billing_resource_order o ON o.id = ps.order_id
		WHERE ps.id = ?`, id).Scan(
		&rec.ID, &rec.OutProfitSharingNo, &rec.OrderID, &rec.OrderNumber, &rec.TenantID,
		&rec.ReferrerUserID, &rec.ReferrerOpenid,
		&rec.CommissionYuanCents, &rec.TotalYuanCents,
		&rec.Status, &rec.FailReason, &paidAt,
	)
	if err == sql.ErrNoRows || (err == nil && rec.ReferrerUserID != userID) {
		return http.StatusNotFound, "", false, "profit sharing record not found"
	}
	if err != nil {
		return http.StatusInternalServerError, "", false, err.Error()
	}
	switch rec.Status {
	case psStatusProcessing:
		return http.StatusOK, "processing", true, ""
	case psStatusFinished:
		return http.StatusOK, "shared", true, ""
	}
	display, share := referrerProfitSharingDisplay(rec.Status, paidAt, now)
	if !share {
		return http.StatusConflict, "", false, "当前状态不可分账（" + display + "）"
	}
	if err := executeProfitSharing(r.Context(), rec); err != nil {
		markProfitSharingStatus(r.Context(), rec.ID, psStatusFailed, err.Error())
		slog.WarnContext(r.Context(), "referrer_share_profit_sharing_failed",
			"level", "warn",
			"referrer_user_id", rec.ReferrerUserID,
			"record_id", rec.ID,
			"error", err.Error(),
		)
		code, msg := profitSharingActionClientError(err)
		return code, "", false, msg
	}
	slog.InfoContext(r.Context(), "referrer_share_profit_sharing_ok",
		"level", "info",
		"referrer_user_id", rec.ReferrerUserID,
		"record_id", rec.ID,
		"out_profit_sharing_no", rec.OutProfitSharingNo,
	)
	return http.StatusOK, "shared", false, ""
}
