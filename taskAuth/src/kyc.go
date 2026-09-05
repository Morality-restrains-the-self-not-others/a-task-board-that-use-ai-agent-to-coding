package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"tracelog"
)

const (
	kycTierT0 = "T0_unverified"
	kycTierT1 = "T1_basic"
	kycTierT2 = "T2_enhanced"

	kycStatusNone         = "none"
	kycStatusPending      = "pending"
	kycStatusApproved     = "approved"
	kycStatusRejected     = "rejected"
	kycStatusNeedsReview  = "needs_review"
	kycStatusExpired      = "expired"

	kycTriggerSystem   = "system"
	kycTriggerAdmin    = "admin"
	kycTriggerEvaluate = "evaluate"
	kycTriggerAML      = "aml"

	kycReasonPhoneVerified = "PHONE_VERIFIED"
	kycReasonAdminOverride = "ADMIN_OVERRIDE"

	reasonKycTierBlocked     = "KYC_TIER_BLOCKED"
	reasonKycAmountExceeds   = "KYC_AMOUNT_EXCEEDS_SINGLE"
	reasonKycAMLReview       = "KYC_AML_REVIEW"
	reasonKycOK              = "OK"
)

var (
	errKycUserRequired   = errors.New("user_id required")
	errKycInvalidTier    = errors.New("invalid tier")
	errKycInvalidStatus  = errors.New("invalid status")
	errKycInvalidAmount  = errors.New("invalid amount")
)

type kycProfile struct {
	UserID      string `json:"user_id"`
	Tier        string `json:"tier"`
	Status      string `json:"status"`
	EffectiveAt string `json:"effective_at,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	RiskFlags   string `json:"risk_flags"`
	UpdatedAt   string `json:"updated_at"`
	CreatedAt   string `json:"created_at"`
}

type kycAuditEntry struct {
	ID            int64  `json:"id"`
	UserID        string `json:"user_id"`
	OldTier       string `json:"old_tier"`
	NewTier       string `json:"new_tier"`
	OldStatus     string `json:"old_status"`
	NewStatus     string `json:"new_status"`
	TriggerSource string `json:"trigger_source"`
	ActorID       string `json:"actor_id"`
	ReasonCode    string `json:"reason_code"`
	ReasonDetail  string `json:"reason_detail"`
	RequestID     string `json:"request_id"`
	CreatedAt     string `json:"created_at"`
}

type kycLimitPolicy struct {
	Tier             string `json:"tier"`
	MaxSingleYuan    int    `json:"max_single_yuan"`
	MaxDailyYuan     int    `json:"max_daily_yuan"`
	RechargeAllowed  bool   `json:"recharge_allowed"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type kycRechargeGateResult struct {
	Allowed          bool   `json:"allowed"`
	ReasonCode       string `json:"reason_code"`
	Tier             string `json:"tier"`
	Status           string `json:"status"`
	MaxSingleYuan    int    `json:"max_single_yuan"`
	MaxDailyYuan     int    `json:"max_daily_yuan"`
	DailyUsedYuanHint int   `json:"daily_used_yuan_hint"`
}

func isValidKycTier(tier string) bool {
	switch tier {
	case kycTierT0, kycTierT1, kycTierT2:
		return true
	default:
		return false
	}
}

func isValidKycStatus(status string) bool {
	switch status {
	case kycStatusNone, kycStatusPending, kycStatusApproved,
		kycStatusRejected, kycStatusNeedsReview, kycStatusExpired:
		return true
	default:
		return false
	}
}

func kycTierRank(tier string) int {
	switch tier {
	case kycTierT2:
		return 2
	case kycTierT1:
		return 1
	default:
		return 0
	}
}

func requestIDFromContext(ctx context.Context) string {
	if tid := tracelog.TraceIDFromContext(ctx); tid != "" {
		return tid
	}
	return ""
}

func getOrCreateKycProfile(userID string) (kycProfile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return kycProfile{}, errKycUserRequired
	}
	p, err := loadKycProfile(userID)
	if err != nil {
		return kycProfile{}, err
	}
	if p != nil {
		return *p, nil
	}
	now := timeNowUTC()
	_, err = db.Exec(`
		INSERT INTO auth_kyc_profile (
			user_id, tier, status, effective_at, expires_at, risk_flags, updated_at, created_at
		) VALUES (?, ?, ?, NULL, NULL, '', ?, ?)`,
		userID, kycTierT0, kycStatusNone, now, now,
	)
	if err != nil {
		// concurrent insert
		p2, err2 := loadKycProfile(userID)
		if err2 != nil {
			return kycProfile{}, err
		}
		if p2 != nil {
			return *p2, nil
		}
		return kycProfile{}, err
	}
	return kycProfile{
		UserID:    userID,
		Tier:      kycTierT0,
		Status:    kycStatusNone,
		RiskFlags: "",
		UpdatedAt: now,
		CreatedAt: now,
	}, nil
}

func loadKycProfile(userID string) (*kycProfile, error) {
	var p kycProfile
	var effectiveAt, expiresAt sql.NullString
	err := db.QueryRow(`
		SELECT user_id, tier, status, effective_at, expires_at, COALESCE(risk_flags,''), updated_at, created_at
		FROM auth_kyc_profile WHERE user_id = ?`, userID,
	).Scan(&p.UserID, &p.Tier, &p.Status, &effectiveAt, &expiresAt, &p.RiskFlags, &p.UpdatedAt, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if effectiveAt.Valid {
		p.EffectiveAt = effectiveAt.String
	}
	if expiresAt.Valid {
		p.ExpiresAt = expiresAt.String
	}
	return &p, nil
}

func hasVerifiedPhoneLoginMethod(userID string) (bool, error) {
	var marker int
	err := db.QueryRow(`
		SELECT 1 FROM auth_login_method
		WHERE object_id = ? AND method_type = 'phone' AND is_verified = 1
		  AND binding_voided_at IS NULL
		LIMIT 1`, userID).Scan(&marker)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func writeKycAuditTx(tx *sql.Tx, entry kycAuditEntry) error {
	id := entry.ID
	if id == 0 {
		id = generateSnowflakeID()
	}
	_, err := tx.Exec(`
		INSERT INTO auth_kyc_audit_log (
			id, user_id, old_tier, new_tier, old_status, new_status,
			trigger_source, actor_id, reason_code, reason_detail, request_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, entry.UserID, entry.OldTier, entry.NewTier, entry.OldStatus, entry.NewStatus,
		entry.TriggerSource, entry.ActorID, entry.ReasonCode, entry.ReasonDetail,
		entry.RequestID, entry.CreatedAt,
	)
	return err
}

func applyKycChange(
	ctx context.Context,
	userID, newTier, newStatus, triggerSource, actorID, reasonCode, reasonDetail string,
) (kycProfile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return kycProfile{}, errKycUserRequired
	}
	if !isValidKycTier(newTier) {
		return kycProfile{}, errKycInvalidTier
	}
	if !isValidKycStatus(newStatus) {
		return kycProfile{}, errKycInvalidStatus
	}
	old, err := getOrCreateKycProfile(userID)
	if err != nil {
		return kycProfile{}, err
	}
	if old.Tier == newTier && old.Status == newStatus {
		return old, nil
	}
	now := timeNowUTC()
	reqID := requestIDFromContext(ctx)
	reasonDetail = sanitizeKycReasonDetail(reasonDetail)

	tx, err := db.Begin()
	if err != nil {
		return kycProfile{}, err
	}
	defer tx.Rollback()

	effectiveAt := old.EffectiveAt
	if newTier != old.Tier || newStatus == kycStatusApproved {
		effectiveAt = now
	}
	_, err = tx.Exec(`
		UPDATE auth_kyc_profile
		SET tier = ?, status = ?, effective_at = ?, updated_at = ?
		WHERE user_id = ?`,
		newTier, newStatus, nullIfEmpty(effectiveAt), now, userID,
	)
	if err != nil {
		return kycProfile{}, err
	}
	if err := writeKycAuditTx(tx, kycAuditEntry{
		UserID:        userID,
		OldTier:       old.Tier,
		NewTier:       newTier,
		OldStatus:     old.Status,
		NewStatus:     newStatus,
		TriggerSource: triggerSource,
		ActorID:       strings.TrimSpace(actorID),
		ReasonCode:    reasonCode,
		ReasonDetail:  reasonDetail,
		RequestID:     reqID,
		CreatedAt:     now,
	}); err != nil {
		return kycProfile{}, err
	}
	if err := tx.Commit(); err != nil {
		return kycProfile{}, err
	}

	updated := old
	updated.Tier = newTier
	updated.Status = newStatus
	updated.EffectiveAt = effectiveAt
	updated.UpdatedAt = now

	emitKycChangeEvents(ctx, old, updated, triggerSource, actorID, reasonCode, reqID)
	return updated, nil
}

func sanitizeKycReasonDetail(detail string) string {
	detail = strings.TrimSpace(detail)
	if len(detail) > 500 {
		return detail[:500]
	}
	return detail
}

func evaluateKyc(ctx context.Context, userID string) (kycProfile, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return kycProfile{}, errKycUserRequired
	}
	profile, err := getOrCreateKycProfile(userID)
	if err != nil {
		return kycProfile{}, err
	}
	verified, err := hasVerifiedPhoneLoginMethod(userID)
	if err != nil {
		return kycProfile{}, err
	}
	if !verified {
		return profile, nil
	}
	// Do not downgrade higher tiers; only uplift toward T1 if below.
	if kycTierRank(profile.Tier) >= kycTierRank(kycTierT1) {
		if profile.Status == kycStatusApproved {
			return profile, nil
		}
		// Keep tier, ensure approved when phone verified and not admin-rejected.
		if profile.Status == kycStatusRejected || profile.Status == kycStatusNeedsReview {
			return profile, nil
		}
	}
	targetTier := profile.Tier
	if kycTierRank(targetTier) < kycTierRank(kycTierT1) {
		targetTier = kycTierT1
	}
	return applyKycChange(
		ctx, userID, targetTier, kycStatusApproved,
		kycTriggerEvaluate, "system", kycReasonPhoneVerified, "verified phone login_method",
	)
}

func adminOverrideKyc(
	ctx context.Context,
	userID, tier, status, reasonCode, reasonDetail, actorID string,
) (kycProfile, error) {
	if strings.TrimSpace(reasonCode) == "" {
		reasonCode = kycReasonAdminOverride
	}
	return applyKycChange(
		ctx, userID, tier, status,
		kycTriggerAdmin, actorID, reasonCode, reasonDetail,
	)
}

func listKycAudit(userID string, limit int) ([]kycAuditEntry, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errKycUserRequired
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := db.Query(`
		SELECT id, user_id, old_tier, new_tier, old_status, new_status,
		       trigger_source, actor_id, reason_code, reason_detail, request_id, created_at
		FROM auth_kyc_audit_log
		WHERE user_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []kycAuditEntry
	for rows.Next() {
		var e kycAuditEntry
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.OldTier, &e.NewTier, &e.OldStatus, &e.NewStatus,
			&e.TriggerSource, &e.ActorID, &e.ReasonCode, &e.ReasonDetail, &e.RequestID, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func maybeEvaluateKycAfterPhoneVerified(ctx context.Context, userID string) {
	if strings.TrimSpace(userID) == "" {
		return
	}
	if _, err := evaluateKyc(ctx, userID); err != nil {
		log.Printf("[taskAuth] kyc evaluate after phone verify user_id=%s err=%v", userID, err)
	}
}
