package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strings"

	"tracelog"
)

// shareCodeOwnerDetail 推荐码反查结果：码 → 持有者（具体的人）。
type shareCodeOwnerDetail struct {
	Code        string `json:"code"`
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	ChannelName string `json:"channel_name"`
	IsDefault   bool   `json:"is_default"`
	Status      string `json:"status"`
}

// lookupShareCodeOwnerDetail 通过推荐码反向查询持有者信息（含用户名）。
// 仅匹配 status='active' 的码（与 lookupShareCodeOwner 语义一致）；未知/停用返回 nil。
// 用户名来自 auth_user_profile（taskAuth 库），缺失时置空，不阻断主结果。
func lookupShareCodeOwnerDetail(code string) (*shareCodeOwnerDetail, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	var row shareCodeOwnerDetail
	var isDefault int
	err := db.QueryRow(
		`SELECT code, user_id, channel_name, is_default, status
		 FROM referral_share_code
		 WHERE code = ? AND status = 'active'
		 LIMIT 1`, code,
	).Scan(&row.Code, &row.UserID, &row.ChannelName, &isDefault, &row.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	row.IsDefault = isDefault == 1

	// 补充用户名（auth_user_profile，COALESCE 防缺列/空值）；authDB 不可用时静默降级
	if authDB != nil {
		var username string
		if err := authDB.QueryRow(
			`SELECT COALESCE(username, '') FROM auth_user_profile WHERE user_id = ?`, row.UserID,
		).Scan(&username); err == nil {
			row.Username = strings.TrimSpace(username)
		}
	}
	return &row, nil
}

// handleAdminShareCodeLookup 管理后台「推荐码反查」：输入推荐码 → 找到持有该码的具体用户。
// 与 applications 等 admin 端点一致：requireSuperuser 鉴权（网关 JWT + X-User-Id）。
// 未命中返回 200 {found:false}（前端友好提示，404 无必要）。
func handleAdminShareCodeLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if _, ok := requireSuperuser(w, r); !ok {
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code is required"})
		return
	}

	detail, err := lookupShareCodeOwnerDetail(code)
	if err != nil {
		slog.ErrorContext(r.Context(), "share_code_lookup_failed",
			"level", "error",
			"code_len", len(code),
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error", "detail": "查询失败，请稍后重试"})
		return
	}

	if detail == nil {
		slog.InfoContext(r.Context(), "share_code_lookup_miss",
			"level", "info",
			"code_len", len(code),
			"found", false,
			"trace_id", tracelog.TraceIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusOK, map[string]interface{}{"found": false})
		return
	}

	slog.InfoContext(r.Context(), "share_code_lookup_hit",
		"level", "info",
		"code_len", len(code),
		"found", true,
		"user_id", detail.UserID,
		"trace_id", tracelog.TraceIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"found":        true,
		"code":         detail.Code,
		"user_id":      detail.UserID,
		"username":     detail.Username,
		"channel_name": detail.ChannelName,
		"is_default":   detail.IsDefault,
		"status":       detail.Status,
	})
}
