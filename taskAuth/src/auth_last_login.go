package main

import (
	"log/slog"
	"strings"
	"time"
)

// touchLastLogin records the user's last successful authentication (or session
// activation). Failures are logged and never block the login response.
// Do not call this from impersonation start: that stamps the target user.
func touchLastLogin(userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" || db == nil {
		return
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	if _, err := db.Exec(`UPDATE auth_user SET last_login = ? WHERE id = ?`, now, userID); err != nil {
		slog.Warn("last_login_touch_failed", "error", err.Error(), "user_id", userID)
	}
}
