package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	relayPhaseTokenInitSucceeded = "token_init_succeeded"
	relayPhaseStartAccepted      = "start_accepted"
)

// upsertRelayStartupSessionRequest is the Gateway → Cloud internal payload.
type upsertRelayStartupSessionRequest struct {
	WorkflowID       string
	TenantID         string
	WorkspaceID      string
	TaskID           string
	Phase            string
	TokenInitialized bool
	RequestID        string
	ExpiresAt        string
}

func parseUpsertRelayStartupSessionRequest(body map[string]interface{}) (upsertRelayStartupSessionRequest, error) {
	req := upsertRelayStartupSessionRequest{
		WorkflowID:       strField(body, "workflow_id"),
		TenantID:         strField(body, "tenant_id"),
		WorkspaceID:      strField(body, "workspace_id"),
		TaskID:           strField(body, "task_id"),
		Phase:            strField(body, "phase"),
		TokenInitialized: toBool(body["token_initialized"]),
		RequestID:        strField(body, "request_id"),
		ExpiresAt:        strField(body, "expires_at"),
	}
	if req.WorkflowID == "" {
		return req, errBadRequest("workflow_id required")
	}
	if req.TenantID == "" || req.WorkspaceID == "" || req.TaskID == "" {
		return req, errBadRequest("tenant_id, workspace_id, task_id required")
	}
	if req.Phase == "" {
		return req, errBadRequest("phase required")
	}
	return req, nil
}

type badRequestError struct{ msg string }

func (e badRequestError) Error() string { return e.msg }

func errBadRequest(msg string) error { return badRequestError{msg: msg} }

// upsertRelayStartupSession merges or creates a Redis session compatible with
// Django RedisRelayStartupSessionRepository / Cloud converge.
func upsertRelayStartupSession(ctx context.Context, store *relaySessionStore, req upsertRelayStartupSessionRequest, now time.Time) (*RelayStartupSession, error) {
	if store == nil || store.client == nil {
		return nil, errBadRequest("relay session store unavailable")
	}
	existing, err := store.findByWorkflowID(ctx, req.WorkflowID)
	if err != nil {
		return nil, err
	}

	scope := RelayTaskScope{
		TenantID:    req.TenantID,
		WorkspaceID: req.WorkspaceID,
		TaskID:      req.TaskID,
		ExpiresAt:   req.ExpiresAt,
	}
	session := &RelayStartupSession{
		WorkflowID:       req.WorkflowID,
		Scope:            scope,
		Phase:            req.Phase,
		TokenInitialized: req.TokenInitialized,
		FailureReason:    "",
		CreatedAt:        now,
		UpdatedAt:        now,
		CreatedAtRaw:     now.UTC().Format(time.RFC3339Nano),
		UpdatedAtRaw:     now.UTC().Format(time.RFC3339Nano),
	}
	if existing != nil {
		session.CreatedAt = existing.CreatedAt
		session.CreatedAtRaw = existing.CreatedAtRaw
		if session.CreatedAtRaw == "" {
			session.CreatedAtRaw = existing.CreatedAt.UTC().Format(time.RFC3339Nano)
		}
		session.LastStatus = existing.LastStatus
		session.LastStatusSeq = existing.LastStatusSeq
		session.FailureReason = existing.FailureReason
		if scope.ExpiresAt == "" {
			session.Scope.ExpiresAt = existing.Scope.ExpiresAt
		}
		// Preserve request_id unless caller supplies a new one.
		session.RequestID = existing.RequestID
		if !req.TokenInitialized && existing.TokenInitialized {
			session.TokenInitialized = true
		}
	}
	if rid := strings.TrimSpace(req.RequestID); rid != "" {
		session.RequestID = &rid
	}
	if err := store.save(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func handleInternalRelayStartupSessionUpsert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeErrorJSON(w, r, http.StatusForbidden, "forbidden")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	req, err := parseUpsertRelayStartupSessionRequest(body)
	if err != nil {
		writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if relaySessions == nil {
		writeErrorJSON(w, r, http.StatusServiceUnavailable, "relay session store unavailable")
		return
	}
	session, err := upsertRelayStartupSession(r.Context(), relaySessions, req, time.Now().UTC())
	if err != nil {
		if _, ok := err.(badRequestError); ok {
			writeErrorJSON(w, r, http.StatusBadRequest, err.Error())
			return
		}
		log.Printf("[taskCloudService] relay startup session upsert failed: %v", err)
		writeErrorJSON(w, r, http.StatusBadGateway, "relay startup session upsert failed")
		return
	}
	resp := map[string]any{
		"status":            "ok",
		"workflow_id":       session.WorkflowID,
		"phase":             session.Phase,
		"token_initialized": session.TokenInitialized,
	}
	if session.RequestID != nil {
		resp["request_id"] = *session.RequestID
	}
	writeJSON(w, http.StatusOK, resp)
}
