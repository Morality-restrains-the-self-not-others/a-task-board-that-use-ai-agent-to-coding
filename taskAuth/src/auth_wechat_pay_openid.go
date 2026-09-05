package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"tracelog"
)

// wechatPayOpenIDRow is the app-scoped WeChat identity used as PERSONAL_OPENID
// for profit-sharing receivers. OpenID is appid-scoped and cannot be reused
// across apps (https://pay.weixin.qq.com/doc/v3/merchant/4012528995).
type wechatPayOpenIDRow struct {
	AppKey string
	AppID  string
	OpenID string
}

func lookupWechatPayOpenID(userID, preferredAppID string) (wechatPayOpenIDRow, bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return wechatPayOpenIDRow{}, false, nil
	}
	preferredAppID = strings.TrimSpace(preferredAppID)
	row := wechatPayOpenIDRow{}
	err := db.QueryRow(`
		SELECT app_key, app_id, openid
		FROM wechat_identity
		WHERE user_id = ? AND openid != '' AND app_id != ''
		ORDER BY CASE WHEN app_id = ? THEN 0 ELSE 1 END, updated_at DESC
		LIMIT 1`, userID, preferredAppID,
	).Scan(&row.AppKey, &row.AppID, &row.OpenID)
	if err == sql.ErrNoRows {
		return wechatPayOpenIDRow{}, false, nil
	}
	if err != nil {
		return wechatPayOpenIDRow{}, false, err
	}
	row.AppKey = strings.TrimSpace(row.AppKey)
	row.AppID = strings.TrimSpace(row.AppID)
	row.OpenID = strings.TrimSpace(row.OpenID)
	if row.OpenID == "" || row.AppID == "" {
		return wechatPayOpenIDRow{}, false, nil
	}
	return row, true, nil
}

// handleInternalWechatPayOpenID GET /api/internal/users/id/{user_id}/wechat-pay-openid/
// 供 taskBill 登记微信分账接收方。不向公开 API 暴露 openid。
func handleInternalWechatPayOpenID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorDetail(w, r, http.StatusForbidden, "forbidden")
		return
	}
	userID := strings.Trim(r.PathValue("user_id"), "/")
	if userID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	preferred := strings.TrimSpace(r.URL.Query().Get("preferred_app_id"))
	row, ok, err := lookupWechatPayOpenID(userID, preferred)
	if err != nil {
		slog.ErrorContext(r.Context(), "wechat_pay_openid_lookup_failed",
			"user_id", userID,
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if !ok {
		writeErrorDetail(w, r, http.StatusNotFound, "wechat identity not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"user_id": userID,
		"app_key": row.AppKey,
		"app_id":  row.AppID,
		"openid":  row.OpenID,
	})
}
