package main

import (
	"database/sql"
	"net/http"
	"strings"
)

func handleListInbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	rows, err := listInboxMessages(userID, 50)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	items := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		items = append(items, inboxMessageJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": items})
}

func handleMarkInboxRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	messageID := strings.TrimSpace(r.PathValue("id"))
	if messageID == "" {
		writeErrorDetail(w, r, http.StatusNotFound, "not found")
		return
	}
	err := markInboxMessageRead(userID, messageID)
	if err == sql.ErrNoRows {
		writeErrorDetail(w, r, http.StatusNotFound, "message not found")
		return
	}
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}
