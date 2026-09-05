package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type containerGatewayValidateBody struct {
	Cookie        string `json:"cookie"`
	Authorization string `json:"authorization"`
	TenantID      string `json:"tenant_id"`
	WorkspaceID   string `json:"workspace_id"`
	TaskID        string `json:"task_id"`
	Path          string `json:"path"`
	Method        string `json:"method"`
}

// handleContainerGatewayValidateSession authenticates browser credentials for taskContainerGateway.
// Membership / CloudServerConfig scope checks are owned by tcg + taskCloudService (zero-Django hot path).
func handleContainerGatewayValidateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var body containerGatewayValidateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
		writeErrorDetail(w, r, http.StatusBadRequest, "invalid json body")
		return
	}
	synth, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		writeErrorDetail(w, r, http.StatusInternalServerError, "failed to build auth request")
		return
	}
	if c := strings.TrimSpace(body.Cookie); c != "" {
		synth.Header.Set("Cookie", c)
	}
	if a := strings.TrimSpace(body.Authorization); a != "" {
		synth.Header.Set("Authorization", a)
	}
	userID, err := resolveTokenUserIDFromRequestStrict(synth)
	if err != nil || strings.TrimSpace(userID) == "" {
		writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	isActive, _, _, isArchived, flagErr := loadUserAuthFlags(userID)
	if flagErr != nil || !isActive || isArchived {
		writeErrorDetail(w, r, http.StatusUnauthorized, "authentication required")
		return
	}
	authMethod := detectAuthMethod(body.Cookie, body.Authorization)
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":     userID,
		"auth_method": authMethod,
		"scope_ok":    true,
	})
}

func detectAuthMethod(cookie, authorization string) string {
	auth := strings.TrimSpace(authorization)
	if strings.HasPrefix(auth, "Token ") || strings.HasPrefix(auth, "Bearer ") {
		return "token"
	}
	if strings.Contains(cookie, "token=") {
		return "token"
	}
	return "session"
}
