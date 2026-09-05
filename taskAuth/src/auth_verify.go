package main

import (
	"net/http"
)

// handleAuthVerify validates Authorization (Token/Bearer) for Go microservice fallback auth.
func handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	userID, ok := resolveUserIDFromRequest(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	isActive, _, _, _, err := loadUserAuthFlags(userID)
	if err != nil || !isActive {
		writeError(w, r, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":   userID,
		"tenant_id": "",
	})
}
