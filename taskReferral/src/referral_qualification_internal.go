package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// handleInternalQualificationActive 供 taskBill 在微信支付预下单 / 分账落库时
// 现查推荐人当前是否有活跃资格（ADR-0033）。只读 referral_code，不改绑边快照。
func handleInternalQualificationActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"status": "method not allowed"})
		return
	}
	if !requireInternalSecret(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"status": "forbidden"})
		return
	}
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "user_id required"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"active":  referrerHasActiveQualification(userID),
	})
}

// handleInternalQualificationActiveBatch 供 taskAuth 超管用户列表一次查出当前页资格，避免 N+1。
func handleInternalQualificationActiveBatch(w http.ResponseWriter, r *http.Request) {
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
	seen := map[string]bool{}
	for _, it := range idsRaw {
		id := strings.TrimSpace(fmt.Sprintf("%v", it))
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		userIDs = append(userIDs, id)
	}
	if len(userIDs) > maxQualificationBatch {
		writeJSON(w, http.StatusBadRequest, map[string]string{"status": "user_ids exceeds 200"})
		return
	}
	qualifications, err := lookupActiveQualificationsBatch(userIDs)
	if err != nil {
		slog.Warn("qualification_batch_lookup_failed", "error", err.Error(), "user_count", len(userIDs))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"qualifications": qualifications})
}
