package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

const (
	amlResultPending = "pending"
	amlResultClear   = "clear"
	amlResultReview  = "review"
	amlResultHit     = "hit"
)

var (
	errAmlInvalidResult = errors.New("invalid aml result")
)

type amlScreeningRecord struct {
	ID           int64  `json:"id"`
	UserID       string `json:"user_id"`
	Provider     string `json:"provider"`
	ScreeningRef string `json:"screening_ref"`
	Result       string `json:"result"`
	CheckedAt    string `json:"checked_at"`
	ActorID      string `json:"actor_id"`
	Notes        string `json:"notes"`
	CreatedAt    string `json:"created_at"`
}

func isValidAmlResult(result string) bool {
	switch result {
	case amlResultPending, amlResultClear, amlResultReview, amlResultHit:
		return true
	default:
		return false
	}
}

// OPT-20260722-025: list AML screenings for admin UI.
func listAmlScreenings(userID string, limit int) ([]amlScreeningRecord, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := db.Query(`
		SELECT id, user_id, provider, screening_ref, result, checked_at, actor_id, notes, created_at
		FROM auth_aml_screening_record
		WHERE user_id = ?
		ORDER BY checked_at DESC, id DESC
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []amlScreeningRecord
	for rows.Next() {
		var rec amlScreeningRecord
		if err := rows.Scan(&rec.ID, &rec.UserID, &rec.Provider, &rec.ScreeningRef,
			&rec.Result, &rec.CheckedAt, &rec.ActorID, &rec.Notes, &rec.CreatedAt); err != nil {
			continue
		}
		out = append(out, rec)
	}
	if out == nil {
		out = []amlScreeningRecord{}
	}
	return out, rows.Err()
}

func latestAmlScreeningResult(userID string) (string, error) {
	rec, err := latestAmlScreeningRecord(userID)
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", nil
	}
	return rec.Result, nil
}

func latestAmlScreeningRecord(userID string) (*amlScreeningRecord, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	var rec amlScreeningRecord
	err := db.QueryRow(`
		SELECT id, user_id, provider, screening_ref, result, checked_at, actor_id, notes, created_at
		FROM auth_aml_screening_record
		WHERE user_id = ?
		ORDER BY checked_at DESC, id DESC
		LIMIT 1`, userID,
	).Scan(
		&rec.ID, &rec.UserID, &rec.Provider, &rec.ScreeningRef, &rec.Result,
		&rec.CheckedAt, &rec.ActorID, &rec.Notes, &rec.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func recordAmlScreening(
	ctx context.Context,
	userID, result, provider, notes, actorID, screeningRef string,
) (amlScreeningRecord, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return amlScreeningRecord{}, errKycUserRequired
	}
	result = strings.TrimSpace(result)
	if !isValidAmlResult(result) {
		return amlScreeningRecord{}, errAmlInvalidResult
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "manual"
	}
	notes = sanitizeKycReasonDetail(notes)
	screeningRef = strings.TrimSpace(screeningRef)
	actorID = strings.TrimSpace(actorID)
	now := timeNowUTC()
	id := generateSnowflakeID()

	// Ensure profile exists for audit linkage.
	if _, err := getOrCreateKycProfile(userID); err != nil {
		return amlScreeningRecord{}, err
	}

	tx, err := db.Begin()
	if err != nil {
		return amlScreeningRecord{}, err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO auth_aml_screening_record (
			id, user_id, provider, screening_ref, result, checked_at, actor_id, notes, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, userID, provider, screeningRef, result, now, actorID, notes, now,
	)
	if err != nil {
		return amlScreeningRecord{}, err
	}

	var oldTier, oldStatus string
	err = tx.QueryRow(`
		SELECT tier, status FROM auth_kyc_profile WHERE user_id = ?`, userID,
	).Scan(&oldTier, &oldStatus)
	if err != nil {
		return amlScreeningRecord{}, err
	}
	statusChanged := false
	// AML review/hit may mark profile needs_review without changing tier.
	if (result == amlResultReview || result == amlResultHit) && oldStatus != kycStatusNeedsReview {
		_, err = tx.Exec(`
			UPDATE auth_kyc_profile SET status = ?, updated_at = ? WHERE user_id = ?`,
			kycStatusNeedsReview, now, userID,
		)
		if err != nil {
			return amlScreeningRecord{}, err
		}
		if err := writeKycAuditTx(tx, kycAuditEntry{
			UserID:        userID,
			OldTier:       oldTier,
			NewTier:       oldTier,
			OldStatus:     oldStatus,
			NewStatus:     kycStatusNeedsReview,
			TriggerSource: kycTriggerAML,
			ActorID:       actorID,
			ReasonCode:    reasonKycAMLReview,
			ReasonDetail:  "aml result=" + result,
			RequestID:     requestIDFromContext(ctx),
			CreatedAt:     now,
		}); err != nil {
			return amlScreeningRecord{}, err
		}
		statusChanged = true
	}

	if err := tx.Commit(); err != nil {
		return amlScreeningRecord{}, err
	}

	rec := amlScreeningRecord{
		ID:           id,
		UserID:       userID,
		Provider:     provider,
		ScreeningRef: screeningRef,
		Result:       result,
		CheckedAt:    now,
		ActorID:      actorID,
		Notes:        notes,
		CreatedAt:    now,
	}
	reqID := requestIDFromContext(ctx)
	publishKycEvent(ctx, "AML_SCREENING_RECORDED", map[string]interface{}{
		"user_id":       userID,
		"result":        result,
		"provider":      provider,
		"screening_ref": screeningRef,
		"actor_id":      actorID,
		"request_id":    reqID,
		"checked_at":    now,
	}, userID)
	if statusChanged {
		emitKycChangeEvents(ctx,
			kycProfile{UserID: userID, Tier: oldTier, Status: oldStatus},
			kycProfile{UserID: userID, Tier: oldTier, Status: kycStatusNeedsReview, UpdatedAt: now},
			kycTriggerAML, actorID, reasonKycAMLReview, reqID,
		)
	}
	return rec, nil
}
