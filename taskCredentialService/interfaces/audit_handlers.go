package interfaces

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"taskCredentialService/domain"
)

func (h *Handlers) handleAuditAppend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, nil)
		return
	}
	var req struct {
		TenantID        string `json:"tenant_id"`
		WorkspaceID     string `json:"workspace_id"`
		TaskID          string `json:"task_id"`
		EventType       string `json:"event_type"`
		SourceComponent string `json:"source_component"`
		AccessToken     string `json:"access_token"`
		TraceID         string `json:"trace_id"`
		ErrorCode       string `json:"error_code"`
		ErrorDetail     string `json:"error_detail"`
		Seq             int    `json:"seq"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "invalid json"})
		return
	}
	taskID := strings.TrimSpace(req.TaskID)
	eventType := strings.TrimSpace(req.EventType)
	if taskID == "" || eventType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "task_id and event_type required"})
		return
	}
	source := strings.TrimSpace(req.SourceComponent)
	if source == "" {
		source = "task-events"
	}
	event := &domain.TokenAuditEvent{
		ID:              generateAuditID(),
		TaskID:          taskID,
		EventType:       eventType,
		SourceComponent: source,
		TraceID:         strings.TrimSpace(req.TraceID),
		ErrorCode:       strings.TrimSpace(req.ErrorCode),
		ErrorDetail:     truncateDetail(req.ErrorDetail),
		Seq:             req.Seq,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
	}
	if tok := strings.TrimSpace(req.AccessToken); tok != "" {
		sum := sha256.Sum256([]byte(tok))
		event.AccessTokenSHA256 = hex.EncodeToString(sum[:])
	}
	if err := h.svc.Token.AppendAuditEvent(event); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func generateAuditID() string {
	return "audit_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}

func truncateDetail(raw string) string {
	const maxLen = 500
	s := strings.TrimSpace(raw)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
