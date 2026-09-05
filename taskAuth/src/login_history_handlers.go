package main

import (
	"database/sql"
	"net/http"
	"strings"
)

func handleListOwnLoginHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	writeLoginHistoryList(w, r, userID)
}

func handleSystemAdminUserLoginHistory(w http.ResponseWriter, r *http.Request, targetUserID string) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := requireSuperuser(w, r); !ok {
		return
	}
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		writeErrorDetail(w, r, http.StatusBadRequest, "user_id required")
		return
	}
	var exists int
	err := db.QueryRow(`SELECT 1 FROM auth_user WHERE id = ?`, targetUserID).Scan(&exists)
	if err == sql.ErrNoRows {
		writeErrorDetail(w, r, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeLoginHistoryList(w, r, targetUserID)
}

func writeLoginHistoryList(w http.ResponseWriter, r *http.Request, userID string) {
	limit := parseIntParam(r, "limit", 20, 100)
	offset := parseIntParam(r, "offset", 0, 100000)
	includeFailures := r.URL.Query().Get("include_failures") == "1"
	rows, total, err := listLoginHistory(userID, limit, offset, includeFailures)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, loginHistoryJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": items,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}
