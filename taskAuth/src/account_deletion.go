package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tracelog"
)

const (
	deletionStatusPending   = "pending_cooldown"
	deletionStatusCancelled = "cancelled"
	deletionStatusExecuting = "executing"
	deletionStatusCompleted = "completed"
	deletionStatusBlocked   = "failed_blocked"

	deletionConfirmText = "注销"
)

type deletionBlocker struct {
	Code       string `json:"code"`
	Blocking   bool   `json:"blocking"`
	Message    string `json:"message"`
	ActionURL  string `json:"action_url,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	TenantName string `json:"tenant_name,omitempty"`
}

type accountDeletionRequest struct {
	ID           int64
	UserID       string
	Status       string
	RequestedAt  time.Time
	EffectiveAt  time.Time
	CancelledAt  sql.NullTime
	ExecutedAt   sql.NullTime
	PrecheckJSON string
}

var (
	fetchDeletionBlockersBillFn   = fetchBillDeletionBlockers
	fetchDeletionBlockersTenantFn = fetchTenantDeletionBlockers
	fetchDeletionBlockersCloudFn  = fetchCloudDeletionBlockers
	cleanupTenantInvitesFn        = cleanupTenantInvites
)

func accountDeletionCooldownDays() int {
	if cfg.AccountDeletionCooldownDays > 0 {
		return cfg.AccountDeletionCooldownDays
	}
	return 15
}

func aggregateDeletionBlockers(ctx context.Context, userID string) ([]deletionBlocker, error) {
	var all []deletionBlocker
	for _, fn := range []func(context.Context, string) ([]deletionBlocker, error){
		fetchDeletionBlockersBillFn,
		fetchDeletionBlockersTenantFn,
		fetchDeletionBlockersCloudFn,
	} {
		items, err := fn(ctx, userID)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	return all, nil
}

func deletionPrecheckPayload(ctx context.Context, userID string) (map[string]interface{}, error) {
	blockers, err := aggregateDeletionBlockers(ctx, userID)
	if err != nil {
		return nil, err
	}
	canRequest := true
	for _, b := range blockers {
		if b.Blocking {
			canRequest = false
			break
		}
	}
	return map[string]interface{}{
		"can_request":   canRequest,
		"blockers":      blockers,
		"cooldown_days": accountDeletionCooldownDays(),
	}, nil
}

func getActiveDeletionRequest(userID string) (*accountDeletionRequest, error) {
	row := db.QueryRow(`
		SELECT id, user_id, status, requested_at, effective_at, cancelled_at, executed_at,
		       COALESCE(CAST(last_precheck_snapshot AS CHAR), '')
		FROM auth_account_deletion_request
		WHERE user_id = ? AND status IN (?, ?, ?)
		ORDER BY requested_at DESC LIMIT 1`,
		userID, deletionStatusPending, deletionStatusExecuting, deletionStatusBlocked,
	)
	var req accountDeletionRequest
	var cancelledAt, executedAt sql.NullTime
	err := row.Scan(&req.ID, &req.UserID, &req.Status, &req.RequestedAt, &req.EffectiveAt,
		&cancelledAt, &executedAt, &req.PrecheckJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	req.CancelledAt = cancelledAt
	req.ExecutedAt = executedAt
	return &req, nil
}

func deletionStatusPayload(userID string) (map[string]interface{}, error) {
	req, err := getActiveDeletionRequest(userID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		completed := false
		var completedAt sql.NullTime
		_ = db.QueryRow(`SELECT deletion_completed_at FROM auth_user WHERE id = ?`, userID).Scan(&completedAt)
		if completedAt.Valid {
			completed = true
		}
		out := map[string]interface{}{"status": "none", "cooldown_days": accountDeletionCooldownDays()}
		if completed {
			out["status"] = "completed"
			out["deletion_completed_at"] = completedAt.Time.UTC().Format(time.RFC3339)
		}
		return out, nil
	}
	return map[string]interface{}{
		"status":       req.Status,
		"request_id":   fmt.Sprintf("%d", req.ID),
		"requested_at": req.RequestedAt.UTC().Format(time.RFC3339),
		"effective_at": req.EffectiveAt.UTC().Format(time.RFC3339),
		"cooldown_days": accountDeletionCooldownDays(),
	}, nil
}

func verifyDeletionReauth(userID, password string) bool {
	password = strings.TrimSpace(password)
	if password == "" {
		return false
	}
	rows, err := db.Query(`
		SELECT password_hash FROM auth_login_method
		WHERE object_id = ? AND binding_voided_at IS NULL AND TRIM(COALESCE(password_hash,'')) != ''`,
		userID)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			continue
		}
		if checkPasswordHash(password, hash) {
			return true
		}
	}
	return false
}

func createDeletionRequest(ctx context.Context, userID string, precheck map[string]interface{}) (*accountDeletionRequest, error) {
	if active, err := getActiveDeletionRequest(userID); err != nil {
		return nil, err
	} else if active != nil {
		return nil, fmt.Errorf("deletion already pending")
	}
	var isArchived bool
	if err := db.QueryRow(`SELECT COALESCE(is_archived,0) FROM auth_user WHERE id = ?`, userID).Scan(&isArchived); err != nil {
		return nil, err
	}
	if isArchived {
		return nil, fmt.Errorf("account already archived")
	}
	now := time.Now().UTC()
	effective := now.Add(time.Duration(accountDeletionCooldownDays()) * 24 * time.Hour)
	id := generateSnowflakeID()
	precheckJSON, _ := json.Marshal(precheck)
	idempotencyKey := fmt.Sprintf("user:%s:%d", userID, now.Unix())
	_, err := db.Exec(`
		INSERT INTO auth_account_deletion_request
		(id, user_id, status, requested_at, effective_at, last_precheck_snapshot, idempotency_key)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, userID, deletionStatusPending, now, effective, string(precheckJSON), idempotencyKey)
	if err != nil {
		return nil, err
	}
	insertDeletionAudit(id, userID, "requested", "user submitted account deletion", traceFromCtx(ctx))
	publishAccountDeletionEvent(ctx, "USER_ACCOUNT_DELETION_REQUESTED", userID, id)
	return &accountDeletionRequest{
		ID: id, UserID: userID, Status: deletionStatusPending,
		RequestedAt: now, EffectiveAt: effective, PrecheckJSON: string(precheckJSON),
	}, nil
}

func cancelDeletionRequest(ctx context.Context, userID string) error {
	req, err := getActiveDeletionRequest(userID)
	if err != nil {
		return err
	}
	if req == nil || req.Status != deletionStatusPending {
		return fmt.Errorf("no pending deletion")
	}
	now := time.Now().UTC()
	res, err := db.Exec(`
		UPDATE auth_account_deletion_request SET status = ?, cancelled_at = ?, updated_at = ?
		WHERE id = ? AND status = ?`, deletionStatusCancelled, now, now, req.ID, deletionStatusPending)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("no pending deletion")
	}
	insertDeletionAudit(req.ID, userID, "cancelled", "user cancelled deletion", traceFromCtx(ctx))
	publishAccountDeletionEvent(ctx, "USER_ACCOUNT_DELETION_CANCELLED", userID, req.ID)
	return nil
}

func insertDeletionAudit(requestID int64, userID, step, detail, traceID string) {
	_, _ = db.Exec(`
		INSERT INTO auth_account_deletion_audit (request_id, user_id, step, detail_redacted, trace_id)
		VALUES (?, ?, ?, ?, ?)`, requestID, userID, step, detail, traceID)
}

func traceFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return strings.TrimSpace(tracelog.TraceIDFromContext(ctx))
}

func executeDueDeletionRequests(ctx context.Context, limit int) (processed int, err error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT id, user_id, status, requested_at, effective_at
		FROM auth_account_deletion_request
		WHERE status = ? AND effective_at <= ?
		ORDER BY effective_at ASC LIMIT ?`,
		deletionStatusPending, time.Now().UTC(), limit)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var req accountDeletionRequest
		if err := rows.Scan(&req.ID, &req.UserID, &req.Status, &req.RequestedAt, &req.EffectiveAt); err != nil {
			continue
		}
		if err := executeOneDeletion(ctx, &req); err != nil {
			slog.WarnContext(ctx, "account_deletion_execute_failed", "user_id", req.UserID, "error", err.Error())
			continue
		}
		processed++
	}
	return processed, nil
}

func executeOneDeletion(ctx context.Context, req *accountDeletionRequest) error {
	res, err := db.Exec(`
		UPDATE auth_account_deletion_request SET status = ?, updated_at = ?
		WHERE id = ? AND status = ?`, deletionStatusExecuting, time.Now().UTC(), req.ID, deletionStatusPending)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	publishAccountDeletionEvent(ctx, "USER_ACCOUNT_DELETION_EXECUTION_STARTED", req.UserID, req.ID)

	precheck, err := deletionPrecheckPayload(ctx, req.UserID)
	if err != nil {
		return markDeletionBlocked(ctx, req, "precheck_error")
	}
	canRequest, _ := precheck["can_request"].(bool)
	if !canRequest {
		return markDeletionBlocked(ctx, req, "precheck_blocked")
	}

	if err := eraseUserPII(req.UserID); err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = db.Exec(`
		UPDATE auth_user SET is_archived = 1, is_active = 0, deletion_completed_at = ? WHERE id = ?`,
		now, req.UserID)
	if err != nil {
		return err
	}
	// OPT-20260820-019: after archiving the last member, invalidate pending invites of
	// tenants that have no remaining active members. Best-effort — failures are logged
	// but must not fail the deletion execution.
	if err := cleanupTenantInvitesFn(ctx, req.UserID); err != nil {
		slog.WarnContext(ctx, "account_deletion_invite_cleanup_failed", "user_id", req.UserID, "error", err.Error())
	}
	_, err = db.Exec(`
		UPDATE auth_account_deletion_request SET status = ?, executed_at = ?, updated_at = ?
		WHERE id = ?`, deletionStatusCompleted, now, now, req.ID)
	if err != nil {
		return err
	}
	insertDeletionAudit(req.ID, req.UserID, "completed", "account archived and PII erased", "")
	publishAccountDeletionEvent(ctx, "USER_ACCOUNT_DELETION_COMPLETED", req.UserID, req.ID)
	return nil
}

// cleanupTenantInvites asks taskTenantService to invalidate pending invitations in
// tenants where the deleted user was the last active member (OPT-20260820-019).
func cleanupTenantInvites(ctx context.Context, userID string) error {
	base := tenantServiceBaseURL()
	if base == "" {
		return nil
	}
	payload, _ := json.Marshal(map[string]string{"user_id": userID})
	url := base + "/api/internal/tenant/account-deletion/cleanup-invites/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cleanup-invites status=%d", resp.StatusCode)
	}
	return nil
}

func markDeletionBlocked(ctx context.Context, req *accountDeletionRequest, reason string) error {
	now := time.Now().UTC()
	_, _ = db.Exec(`
		UPDATE auth_account_deletion_request SET status = ?, updated_at = ? WHERE id = ?`,
		deletionStatusBlocked, now, req.ID)
	insertDeletionAudit(req.ID, req.UserID, "blocked", reason, "")
	publishAccountDeletionEvent(ctx, "USER_ACCOUNT_DELETION_BLOCKED", req.UserID, req.ID)
	return fmt.Errorf("deletion blocked: %s", reason)
}

func eraseUserPII(userID string) error {
	deletedTag := "deleted_" + hashUserID(userID)
	_, _ = db.Exec(`UPDATE auth_user_profile SET username = '', avatar = '' WHERE user_id = ?`, userID)
	_, _ = db.Exec(`
		UPDATE auth_login_method SET identifier = ?, password_hash = '', binding_voided_at = NOW(), updated_at = NOW()
		WHERE object_id = ? AND binding_voided_at IS NULL`, deletedTag, userID)
	_, _ = db.Exec(`DELETE FROM auth_customtoken WHERE object_id = ?`, userID)
	_, _ = db.Exec(`UPDATE auth_user_access_token SET is_revoked = 1 WHERE user_id = ?`, userID)
	return nil
}

func hashUserID(userID string) string {
	sum := sha256.Sum256([]byte(userID + ":deleted"))
	return hex.EncodeToString(sum[:8])
}

func publishAccountDeletionEvent(ctx context.Context, eventType, userID string, requestID int64) {
	data := map[string]interface{}{
		"user_id":    userID,
		"request_id": fmt.Sprintf("%d", requestID),
	}
	_ = publishDomainEventKafka(ctx, eventType, data, userID)
}

func decodeBlockersResponse(raw []byte) ([]deletionBlocker, error) {
	var wrapper struct {
		Blockers []deletionBlocker `json:"blockers"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Blockers, nil
}

func internalServiceGet(ctx context.Context, url, secretHeader, secretValue string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	if secretValue != "" {
		req.Header.Set(secretHeader, secretValue)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}
