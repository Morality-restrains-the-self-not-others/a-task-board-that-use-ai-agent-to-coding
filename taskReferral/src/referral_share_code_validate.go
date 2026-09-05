package main

import (
	"log/slog"
	"net/http"
	"strings"
)

// handleInternalShareCodeValidate 供 taskAuth 在注册邀请门禁（invite_code 策略开启）
// 时同步校验分享链接中的 access_code 是否为有效 active 分享码（fail-closed）：
// 有效 → taskAuth 豁免 invite_code 要求（分享链接即邀请凭证）；
// 未知/无效 → taskAuth 拒绝注册，保持闸门。
//
// 与 bind-from-code 一致：分享码不依赖推荐资格仍可用（资格只门控分润），
// 这里只校验 code 存在且 status='active'。
func handleInternalShareCodeValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"status": "method not allowed"})
		return
	}
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "forbidden"})
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "invalid json"})
		return
	}
	code := strings.TrimSpace(strField(body, "code"))
	if code == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"valid": false, "reason": "empty_code"})
		return
	}
	owner, err := lookupShareCodeOwner(code)
	if err != nil {
		slog.Warn("share_code_validate_failed",
			"error", err.Error(),
			"code_len", len(code),
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "db error"})
		return
	}
	if owner == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"valid": false, "reason": "unknown_code"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"valid": true, "owner_user_id": owner})
}
