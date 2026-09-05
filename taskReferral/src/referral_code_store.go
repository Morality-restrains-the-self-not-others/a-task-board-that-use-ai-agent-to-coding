package main

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"
)

func listReferralApplications(status string, limit, offset int) ([]referralCodeRow, int, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	var rows *sql.Rows
	var err error

	if status != "" {
		rows, err = db.Query(`
			SELECT id, user_id, status, applied_at, approved_at, expires_at,
			       rejected_at, reject_reason, reviewed_by, personal_intro, legal_name
			FROM referral_code
			WHERE status = ?
			ORDER BY applied_at DESC
			LIMIT ? OFFSET ?`, status, limit, offset,
		)
	} else {
		rows, err = db.Query(`
			SELECT id, user_id, status, applied_at, approved_at, expires_at,
			       rejected_at, reject_reason, reviewed_by, personal_intro, legal_name
			FROM referral_code
			ORDER BY applied_at DESC
			LIMIT ? OFFSET ?`, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []referralCodeRow
	for rows.Next() {
		var r referralCodeRow
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.Status, &r.AppliedAt,
			&r.ApprovedAt, &r.ExpiresAt, &r.RejectedAt,
			&r.RejectReason, &r.ReviewedBy, &r.PersonalIntro, &r.LegalName,
		); err != nil {
			return nil, 0, err
		}
		decorateReferralApplication(&r)
		items = append(items, r)
	}
	if items == nil {
		items = []referralCodeRow{}
	}
	attachReferralRatioDisplays(items)

	// OPT-20260824-006: 批量水合默认分享码（referral_share_code is_default=1 active），
	// 供管理端申请列表直接展示「申请者 ↔ 其推荐码」。失败只打日志不阻断列表。
	if len(items) > 0 {
		userIDs := make([]string, 0, len(items))
		for i := range items {
			userIDs = append(userIDs, items[i].UserID)
		}
		codes, err := lookupShareCodesByUsersBatch(userIDs)
		if err != nil {
			log.Printf("[taskReferral] share code hydrate: %v", err)
		} else {
			for i := range items {
				items[i].ShareCode = codes[items[i].UserID]
			}
		}
	}

	var total int
	if status != "" {
		_ = db.QueryRow(`SELECT COUNT(*) FROM referral_code WHERE status = ?`, status).Scan(&total)
	} else {
		_ = db.QueryRow(`SELECT COUNT(*) FROM referral_code`).Scan(&total)
	}

	return items, total, rows.Err()
}

func getActiveOrPendingReferralCode(userID string) (*referralCodeRow, error) {
	var r referralCodeRow
	var approvedAt, expiresAt sql.NullString
	err := db.QueryRow(`
		SELECT id, user_id, status, applied_at, approved_at, expires_at,
		       rejected_at, reject_reason, reviewed_by, personal_intro, legal_name
		FROM referral_code
		WHERE user_id = ? AND status IN ('pending', 'approved')
		ORDER BY applied_at DESC
		LIMIT 1`, userID,
	).Scan(&r.ID, &r.UserID, &r.Status, &r.AppliedAt,
		&approvedAt, &expiresAt, &r.RejectedAt,
		&r.RejectReason, &r.ReviewedBy, &r.PersonalIntro, &r.LegalName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if approvedAt.Valid {
		r.ApprovedAt = &approvedAt.String
	}
	if expiresAt.Valid {
		r.ExpiresAt = &expiresAt.String
	}
	return &r, nil
}

// referrerHasActiveQualification is the bind-time snapshot source for
// commission_eligible: approved and not expired. Share codes remain usable
// without this; only 分成/计提 is gated.
func referrerHasActiveQualification(userID string) bool {
	return getActiveReferralCode(strings.TrimSpace(userID)) != nil
}

const maxQualificationBatch = 200

func referralCodeStillActive(expiresAt *string) bool {
	if expiresAt == nil {
		return true
	}
	expires, err := time.Parse(time.RFC3339, *expiresAt)
	if err == nil && expires.Before(time.Now()) {
		return false
	}
	return true
}

// lookupActiveQualificationsBatch returns user_id → currently active 分账资格.
// Every requested id is present; missing/expired/revoked codes are false.
func lookupActiveQualificationsBatch(userIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(userIDs))
	ids := make([]string, 0, len(userIDs))
	seen := map[string]bool{}
	for _, raw := range userIDs {
		id := strings.TrimSpace(raw)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		out[id] = false
		if len(ids) >= maxQualificationBatch {
			break
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	rows, err := db.Query(`
		SELECT user_id, expires_at
		FROM referral_code
		WHERE user_id IN (`+strings.Join(placeholders, ",")+`) AND status = 'approved'`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var expiresAt sql.NullString
		if err := rows.Scan(&userID, &expiresAt); err != nil {
			continue
		}
		var exp *string
		if expiresAt.Valid {
			exp = &expiresAt.String
		}
		if referralCodeStillActive(exp) {
			out[userID] = true
		}
	}
	return out, rows.Err()
}

func getActiveReferralCode(userID string) *referralCodeRow {
	rows, err := db.Query(`
		SELECT id, user_id, status, applied_at, approved_at, expires_at,
		       rejected_at, reject_reason, reviewed_by, personal_intro, legal_name
		FROM referral_code
		WHERE user_id = ? AND status = 'approved'
		ORDER BY approved_at DESC`, userID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var r referralCodeRow
		var approvedAt, expiresAt, rejectedAt sql.NullString
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.Status, &r.AppliedAt,
			&approvedAt, &expiresAt, &rejectedAt,
			&r.RejectReason, &r.ReviewedBy, &r.PersonalIntro, &r.LegalName,
		); err != nil {
			continue
		}
		if approvedAt.Valid {
			r.ApprovedAt = &approvedAt.String
		}
		if expiresAt.Valid {
			r.ExpiresAt = &expiresAt.String
		}
		if rejectedAt.Valid {
			r.RejectedAt = &rejectedAt.String
		}
		if r.ExpiresAt != nil {
			expires, err := time.Parse(time.RFC3339, *r.ExpiresAt)
			if err == nil && expires.Before(time.Now()) {
				continue
			}
		}
		return &r
	}
	return nil
}

func getPendingReferralCode(userID string) *referralCodeRow {
	var r referralCodeRow
	var approvedAt, expiresAt, rejectedAt sql.NullString
	err := db.QueryRow(`
		SELECT id, user_id, status, applied_at, approved_at, expires_at,
		       rejected_at, reject_reason, reviewed_by, personal_intro, legal_name
		FROM referral_code
		WHERE user_id = ? AND status = 'pending'
		ORDER BY applied_at DESC
		LIMIT 1`, userID,
	).Scan(&r.ID, &r.UserID, &r.Status, &r.AppliedAt,
		&approvedAt, &expiresAt, &rejectedAt,
		&r.RejectReason, &r.ReviewedBy, &r.PersonalIntro, &r.LegalName,
	)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return nil
	}
	return &r
}

func getLastRejectedReferralCode(userID string) *referralCodeRow {
	var r referralCodeRow
	var approvedAt, expiresAt, rejectedAt sql.NullString
	err := db.QueryRow(`
		SELECT id, user_id, status, applied_at, approved_at, expires_at,
		       rejected_at, reject_reason, reviewed_by, personal_intro, legal_name
		FROM referral_code
		WHERE user_id = ? AND status = 'rejected'
		ORDER BY rejected_at DESC
		LIMIT 1`, userID,
	).Scan(&r.ID, &r.UserID, &r.Status, &r.AppliedAt,
		&approvedAt, &expiresAt, &rejectedAt,
		&r.RejectReason, &r.ReviewedBy, &r.PersonalIntro, &r.LegalName,
	)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return nil
	}
	if rejectedAt.Valid {
		r.RejectedAt = &rejectedAt.String
	}
	return &r
}

func getReferralApplicationByID(appID int) (*referralCodeRow, error) {
	var r referralCodeRow
	var approvedAt, expiresAt, rejectedAt sql.NullString
	err := db.QueryRow(`
		SELECT id, user_id, status, applied_at, approved_at, expires_at,
		       rejected_at, reject_reason, reviewed_by, personal_intro, legal_name
		FROM referral_code
		WHERE id = ?`, appID,
	).Scan(&r.ID, &r.UserID, &r.Status, &r.AppliedAt,
		&approvedAt, &expiresAt, &rejectedAt,
		&r.RejectReason, &r.ReviewedBy, &r.PersonalIntro, &r.LegalName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if approvedAt.Valid {
		r.ApprovedAt = &approvedAt.String
	}
	if expiresAt.Valid {
		r.ExpiresAt = &expiresAt.String
	}
	if rejectedAt.Valid {
		r.RejectedAt = &rejectedAt.String
	}
	return &r, nil
}

func strPtr(s string) *string {
	return &s
}

func publishReferralEvent(ctx context.Context, eventType string, payload map[string]interface{}, key string) {
	if key == "" {
		return
	}
	log.Printf("[taskReferral] referral event %s key=%s", eventType, key)
}

func expireReferralCodes() {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := db.Exec(`
		UPDATE referral_code
		SET status = 'expired'
		WHERE status = 'approved'
		  AND expires_at IS NOT NULL
		  AND expires_at < ?`, now)
	if err != nil {
		log.Printf("[taskReferral] expireReferralCodes: %v", err)
		return
	}
	n, _ := result.RowsAffected()
	if n > 0 {
		log.Printf("[taskReferral] expireReferralCodes: marked %d records as expired", n)
	}
}
