package main

import (
	"net/http"
)

// handleTranslateBranchTitle is retired: public traffic is owned by task-project-service.
// Kept as an explicit 501 so accidental TTS routing is not silently mishandled.
func handleTranslateBranchTitle(w http.ResponseWriter, r *http.Request, tenantID string) {
	_ = tenantID
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeErrorMap(w, r, http.StatusNotImplemented, map[string]interface{}{
		"error":   "translate-branch-title moved to task-project-service",
		"service": "task-project-service",
	})
}
