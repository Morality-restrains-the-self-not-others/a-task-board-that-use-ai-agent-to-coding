package main

import (
	"database/sql"
	"strings"
	"time"
)

const idleNaiveLayout = "2006-01-02 15:04:05"

// parseIdleClockPreferPast reads a naive DATETIME. Digits are first UTC; if that
// instant is in the future, reinterpret as Asia/Shanghai (CSC created_at is often
// session-local CST while last_heartbeat_at is UTC).
func parseIdleClockPreferPast(raw string, now time.Time) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	m := cloudDateTimePrefix.FindStringSubmatch(raw)
	if m == nil {
		t, ok := resolveIdleSinceTimestamp(raw, "")
		return t, ok
	}
	digits := m[1] + " " + m[2]
	now = now.UTC()
	utcT, err := time.ParseInLocation(idleNaiveLayout, digits, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	if !utcT.After(now.Add(2 * time.Minute)) {
		return utcT.UTC(), true
	}
	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		return utcT.UTC(), true
	}
	cstT, err := time.ParseInLocation(idleNaiveLayout, digits, loc)
	if err != nil {
		return utcT.UTC(), true
	}
	return cstT.UTC(), true
}

func loadCSCCreatedAtRaw(id string) string {
	if db == nil || strings.TrimSpace(id) == "" {
		return ""
	}
	var raw sql.NullString
	err := db.QueryRow(`SELECT created_at FROM cloud_server_configs WHERE id=?`, id).Scan(&raw)
	if err != nil || !raw.Valid {
		return ""
	}
	return strings.TrimSpace(raw.String)
}

func neverInstructedIdleRaw(cfg *CloudServerConfig) string {
	if cfg == nil {
		return ""
	}
	if s := strings.TrimSpace(cfg.UserdataRunVerified); s != "" {
		return s
	}
	return loadCSCCreatedAtRaw(cfg.ID)
}
