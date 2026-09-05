package main

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// handleUpsertUserProfile is an internal endpoint for upserting user profile data
// (username display cache). Replaces Django upsert-user-profile intent.
// POST /api/internal/users/{user_id}/profile/
func handleUpsertUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	username := strField(body, "username")
	if err := upsertUserProfile(userID, username); err != nil {
		log.Printf("[taskAuth] upsert profile failed for user %s: %v", userID, err)
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func upsertUserProfile(userID, username string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	// avatar 显式写 ''（TEXT 列不允许 DEFAULT，严格模式下省略会报 1364）；
	// 更新时只改 username/updated_at，不覆盖已有 avatar。
	_, err := db.Exec(`
		INSERT INTO auth_user_profile (user_id, username, avatar, updated_at)
		VALUES (?, ?, '', ?)
		ON DUPLICATE KEY UPDATE username = VALUES(username), updated_at = VALUES(updated_at)`,
		userID, username, now)
	return err
}
