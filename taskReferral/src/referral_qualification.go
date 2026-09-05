package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"tracelog"
)

const (
	reviewReasonMinRunes = 8
	reviewReasonMaxRunes = 500
)

var (
	errReferralReasonInvalid    = errors.New("invalid_reason")
	errReferralAppNotRevocable  = errors.New("not_revocable")
	errReferralIdempotencyClash = errors.New("idempotency_conflict")
)

type qualificationAuditEntry struct {
	ID             int64  `json:"id"`
	ApplicationID  int    `json:"application_id"`
	UserID         string `json:"user_id"`
	Action         string `json:"action"`
	Reason         string `json:"reason"`
	OperatorID     string `json:"operator_id"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	TraceID        string `json:"trace_id,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func validateReviewReason(reason string) (string, error) {
	trimmed := strings.TrimSpace(reason)
	n := utf8.RuneCountInString(trimmed)
	if n < reviewReasonMinRunes || n > reviewReasonMaxRunes {
		return "", errReferralReasonInvalid
	}
	return trimmed, nil
}

func qualificationIsActive(r *referralCodeRow) bool {
	if r == nil || r.Status != "approved" {
		return false
	}
	if r.ExpiresAt == nil || strings.TrimSpace(*r.ExpiresAt) == "" {
		return true
	}
	expires, err := time.Parse(time.RFC3339, strings.TrimSpace(*r.ExpiresAt))
	if err != nil {
		return true
	}
	return time.Now().Before(expires)
}

func referralStatusDisplay(r *referralCodeRow) string {
	if r == nil {
		return ""
	}
	switch r.Status {
	case "pending":
		return "待审批"
	case "approved":
		if qualificationIsActive(r) {
			return "已通过"
		}
		return "已过期"
	case "rejected":
		return "已拒绝"
	case "expired":
		return "已过期"
	case "revoked":
		return "已取消资格"
	default:
		return r.Status
	}
}

func decorateReferralApplication(r *referralCodeRow) {
	if r == nil {
		return
	}
	r.IsActive = qualificationIsActive(r)
	r.CanRevoke = r.IsActive
	r.StatusDisplay = referralStatusDisplay(r)
	r.LastActionReason = strings.TrimSpace(r.RejectReason)
}

func newAuditIdempotencyKey(provided string, appID int, action string) string {
	provided = strings.TrimSpace(provided)
	if provided != "" {
		return provided
	}
	return fmt.Sprintf("auto-%s-%d-%d", action, appID, time.Now().UnixNano())
}

func getAuditByIdempotencyKey(key string) (*qualificationAuditEntry, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}
	var e qualificationAuditEntry
	var created time.Time
	err := db.QueryRow(`
		SELECT id, application_id, user_id, action, reason, operator_id, idempotency_key, trace_id, created_at
		FROM referral_qualification_audit
		WHERE idempotency_key = ?`, key,
	).Scan(&e.ID, &e.ApplicationID, &e.UserID, &e.Action, &e.Reason, &e.OperatorID, &e.IdempotencyKey, &e.TraceID, &created)
	if err == nil {
		e.CreatedAt = created.UTC().Format(time.RFC3339)
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func insertQualificationAudit(ctx context.Context, e qualificationAuditEntry) (*qualificationAuditEntry, error) {
	if e.IdempotencyKey == "" {
		e.IdempotencyKey = newAuditIdempotencyKey("", e.ApplicationID, e.Action)
	}
	if e.CreatedAt == "" {
		e.CreatedAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	if e.TraceID == "" {
		e.TraceID = tracelog.TraceIDFromContext(ctx)
	}
	res, err := db.Exec(`
		INSERT INTO referral_qualification_audit (
			application_id, user_id, action, reason, operator_id, idempotency_key, trace_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ApplicationID, e.UserID, e.Action, e.Reason, e.OperatorID, e.IdempotencyKey, e.TraceID, e.CreatedAt,
	)
	if err != nil {
		if existing, lookErr := getAuditByIdempotencyKey(e.IdempotencyKey); lookErr == nil && existing != nil {
			if existing.ApplicationID != e.ApplicationID || existing.Action != e.Action {
				return existing, errReferralIdempotencyClash
			}
			return existing, nil
		}
		return nil, err
	}
	id, _ := res.LastInsertId()
	e.ID = id
	return &e, nil
}

func listQualificationAudit(appID, limit, offset int) ([]qualificationAuditEntry, int, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := db.Query(`
		SELECT id, application_id, user_id, action, reason, operator_id, idempotency_key, trace_id, created_at
		FROM referral_qualification_audit
		WHERE application_id = ?
		ORDER BY id DESC
		LIMIT ? OFFSET ?`, appID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []qualificationAuditEntry{}
	for rows.Next() {
		var e qualificationAuditEntry
		var created time.Time
		if err := rows.Scan(&e.ID, &e.ApplicationID, &e.UserID, &e.Action, &e.Reason, &e.OperatorID, &e.IdempotencyKey, &e.TraceID, &created); err != nil {
			return nil, 0, err
		}
		e.CreatedAt = created.UTC().Format(time.RFC3339)
		items = append(items, e)
	}
	var total int
	_ = db.QueryRow(`SELECT COUNT(*) FROM referral_qualification_audit WHERE application_id = ?`, appID).Scan(&total)
	return items, total, rows.Err()
}

func revokeReferralQualification(ctx context.Context, appID int, operatorID, reason, idemKey string) (map[string]interface{}, error) {
	reason, err := validateReviewReason(reason)
	if err != nil {
		return nil, err
	}
	app, err := getReferralApplicationByID(appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errReferralAppNotFound
	}

	idemKey = newAuditIdempotencyKey(idemKey, appID, "revoke")
	if existing, lookErr := getAuditByIdempotencyKey(idemKey); lookErr == nil && existing != nil {
		if existing.ApplicationID == appID && existing.Action == "revoke" {
			return map[string]interface{}{
				"status":  "revoked",
				"message": "already_revoked",
			}, nil
		}
		return nil, errReferralIdempotencyClash
	}

	if app.Status == "revoked" {
		return map[string]interface{}{
			"status":  "revoked",
			"message": "already_revoked",
		}, nil
	}
	if !qualificationIsActive(app) {
		return nil, errReferralAppNotRevocable
	}

	now := timeNowUTC()
	_, err = db.Exec(`
		UPDATE referral_code
		SET status = 'revoked', rejected_at = ?, reject_reason = ?, reviewed_by = ?
		WHERE id = ? AND status = 'approved'`,
		now, reason, operatorID, appID,
	)
	if err != nil {
		return nil, err
	}

	if _, err := insertQualificationAudit(ctx, qualificationAuditEntry{
		ApplicationID:  appID,
		UserID:         app.UserID,
		Action:         "revoke",
		Reason:         reason,
		OperatorID:     operatorID,
		IdempotencyKey: idemKey,
	}); err != nil {
		return nil, err
	}

	publishReferralEvent(ctx, "REFERRAL_QUALIFICATION_REVOKED", map[string]interface{}{
		"user_id":     app.UserID,
		"operator_id": operatorID,
		"app_id":      appID,
		"reason_len":  utf8.RuneCountInString(reason),
	}, fmt.Sprintf("%d", appID))

	if err := disableBillEligibility(app.UserID); err != nil {
		slog.WarnContext(ctx, "referral_revoke_disable_eligibility_failed",
			"level", "warn",
			"user_id", app.UserID,
			"app_id", appID,
			"error", err.Error(),
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
	}

	// OPT-20260822-019: best-effort 删除微信分账接收方，避免商户后台残留已失效接收方。
	// 失败只打 warn，不回滚资格撤销。
	if recv := deleteWechatProfitSharingReceiver(app.UserID); recv.Status != "deleted" {
		slog.WarnContext(ctx, "referral_revoke_delete_receiver_failed",
			"level", "warn",
			"user_id", app.UserID,
			"app_id", appID,
			"receiver_status", recv.Status,
			"reason", recv.Reason,
			"trace_id", tracelog.TraceIDFromContext(ctx),
		)
	}

	slog.InfoContext(ctx, "referral_qualification_revoked",
		"level", "info",
		"user_id", app.UserID,
		"app_id", appID,
		"operator_id", operatorID,
		"reason_len", utf8.RuneCountInString(reason),
		"trace_id", tracelog.TraceIDFromContext(ctx),
	)

	return map[string]interface{}{
		"status":  "revoked",
		"message": "已取消该用户的分账资格",
	}, nil
}
