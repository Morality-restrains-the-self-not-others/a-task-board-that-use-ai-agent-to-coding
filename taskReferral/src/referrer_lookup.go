package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// referrerEdge is the billing_referral_edge row a referred user belongs to.
type referrerEdge struct {
	ReferrerUserID string `json:"referrer_user_id"`
	ChannelCode    string `json:"channel_code"`
}

// handleInternalReferrersLookup 批量查询一批用户是被谁推荐的（billing_referral_edge）。
// 供 taskAuth system-admin 用户列表填充「推荐人」列（OPT-20260821-031）。
// 禁止 taskAuth 直连 task_bill —— 统一走本内部接口读边。
func handleInternalReferrersLookup(w http.ResponseWriter, r *http.Request) {
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
	idsRaw, _ := body["user_ids"].([]interface{})
	userIDs := make([]string, 0, len(idsRaw))
	seen := make(map[string]bool, len(idsRaw))
	for _, it := range idsRaw {
		id := strings.TrimSpace(fmt.Sprintf("%v", it))
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		userIDs = append(userIDs, id)
	}
	if len(userIDs) == 0 {
		writeJSON(w, http.StatusOK, map[string]interface{}{"referrers": map[string]interface{}{}})
		return
	}
	referrers, err := lookupReferrersBatch(userIDs)
	if err != nil {
		slog.Warn("referrer_lookup_failed", "error", err.Error(), "user_count", len(userIDs))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"referrers": referrers})
}

// lookupReferrersBatch 一次 IN 查询批量返回 referred_user_id → referrer 边。
func lookupReferrersBatch(userIDs []string) (map[string]referrerEdge, error) {
	out := make(map[string]referrerEdge, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := billDB.Query(
		`SELECT referred_user_id, referrer_user_id, channel_code
		 FROM billing_referral_edge
		 WHERE referred_user_id IN (`+strings.Join(placeholders, ",")+`) AND referrer_user_id != ''`, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var referred, referrer, channel string
		if err := rows.Scan(&referred, &referrer, &channel); err != nil {
			continue
		}
		out[referred] = referrerEdge{ReferrerUserID: referrer, ChannelCode: channel}
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	return out, nil
}
