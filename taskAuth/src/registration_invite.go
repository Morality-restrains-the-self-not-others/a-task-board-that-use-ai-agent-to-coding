package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const registrationInvitePolicyKey = "global"

const inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var (
	errInviteCodeRequired        = errors.New("invite_code_required")
	errInviteCodeInvalid         = errors.New("invite_code_invalid")
	errInviteCodeUsed            = errors.New("invite_code_used")
	errInviteCodeRevoked         = errors.New("invite_code_revoked")
	errInviteFeatureDisabled     = errors.New("invite_feature_disabled")
	errDailyQuotaExhausted       = errors.New("daily_quota_exhausted")
	errAccessCodeInvalid         = errors.New("access_code_invalid")
	errInviteGateUnavailable     = errors.New("invite_gate_unavailable")
)

var shanghaiLocation *time.Location

func init() {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	shanghaiLocation = loc
}

func shanghaiCalendarDay(now time.Time) string {
	return now.In(shanghaiLocation).Format("2006-01-02")
}

type registrationInvitePolicy struct {
	Enabled    bool   `json:"enabled"`
	DailyQuota int    `json:"daily_quota"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	UpdatedBy  string `json:"updated_by,omitempty"`
}

type registrationInviteCodeRow struct {
	ID               string  `json:"id"`
	Code             string  `json:"code"`
	IssuerUserID     string  `json:"issuer_user_id"`
	Status           string  `json:"status"`
	IssuedDay        string  `json:"issued_day"`
	CreatedAt        string  `json:"created_at"`
	RedeemedByUserID *string `json:"redeemed_by_user_id,omitempty"`
	RedeemedAt       *string `json:"redeemed_at,omitempty"`
}

type registrationInviteRelation struct {
	Code             string  `json:"code"`
	IssuerUserID     string  `json:"issuer_user_id"`
	Status           string  `json:"status"`
	IssuedDay        string  `json:"issued_day"`
	CreatedAt        string  `json:"created_at"`
	RedeemedByUserID *string `json:"redeemed_by_user_id,omitempty"`
	RedeemedAt       *string `json:"redeemed_at,omitempty"`
}

func inviteCodeFromBody(body map[string]interface{}) string {
	return strings.ToUpper(strField(body, "invite_code"))
}

func inviteErrorCode(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, errInviteCodeRequired):
		return "invite_code_required"
	case errors.Is(err, errInviteCodeInvalid):
		return "invite_code_invalid"
	case errors.Is(err, errInviteCodeUsed):
		return "invite_code_used"
	case errors.Is(err, errInviteCodeRevoked):
		return "invite_code_revoked"
	case errors.Is(err, errInviteFeatureDisabled):
		return "invite_feature_disabled"
	case errors.Is(err, errDailyQuotaExhausted):
		return "daily_quota_exhausted"
	case errors.Is(err, errAccessCodeInvalid):
		return "access_code_invalid"
	case errors.Is(err, errInviteGateUnavailable):
		return "invite_gate_unavailable"
	default:
		return "internal_error"
	}
}

func getRegistrationInvitePolicy() (registrationInvitePolicy, error) {
	var enabled int
	var dailyQuota int
	var updatedAt string
	var updatedBy sql.NullString
	err := db.QueryRow(`
		SELECT enabled, daily_quota, updated_at, updated_by
		FROM auth_registration_invite_policy
		WHERE singleton_key = ?`, registrationInvitePolicyKey,
	).Scan(&enabled, &dailyQuota, &updatedAt, &updatedBy)
	if err == sql.ErrNoRows {
		return registrationInvitePolicy{Enabled: false, DailyQuota: 0}, nil
	}
	if err != nil {
		return registrationInvitePolicy{}, err
	}
	policy := registrationInvitePolicy{
		Enabled:    enabled != 0,
		DailyQuota: dailyQuota,
		UpdatedAt:  updatedAt,
	}
	if updatedBy.Valid {
		policy.UpdatedBy = updatedBy.String
	}
	return policy, nil
}

func updateRegistrationInvitePolicy(enabled bool, dailyQuota int, updatedBy string) (registrationInvitePolicy, error) {
	if dailyQuota < 0 {
		dailyQuota = 0
	}
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	now := timeNowUTC()
	_, err := db.Exec(`
		UPDATE auth_registration_invite_policy
		SET enabled = ?, daily_quota = ?, updated_at = ?, updated_by = ?
		WHERE singleton_key = ?`,
		enabledInt, dailyQuota, now, nullIfEmpty(updatedBy), registrationInvitePolicyKey,
	)
	if err != nil {
		return registrationInvitePolicy{}, err
	}
	return getRegistrationInvitePolicy()
}

func nullIfEmpty(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func countIssuedToday(day string) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM auth_registration_invite_code
		WHERE issued_day = ?`, day,
	).Scan(&count)
	return count, err
}

func remainingToday(policy registrationInvitePolicy) (int, error) {
	if policy.DailyQuota <= 0 {
		return 0, nil
	}
	issued, err := countIssuedToday(shanghaiCalendarDay(time.Now()))
	if err != nil {
		return 0, err
	}
	remaining := policy.DailyQuota - issued
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

func generateRegistrationInviteCode() (string, error) {
	buf := make([]byte, 8)
	alphabetLen := big.NewInt(int64(len(inviteCodeAlphabet)))
	for i := range buf {
		n, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		buf[i] = inviteCodeAlphabet[n.Int64()]
	}
	return string(buf), nil
}

func applyRegistrationInviteCode(ctx context.Context, issuerUserID string) (registrationInviteCodeRow, error) {
	policy, err := getRegistrationInvitePolicy()
	if err != nil {
		return registrationInviteCodeRow{}, err
	}
	if !policy.Enabled {
		return registrationInviteCodeRow{}, errInviteFeatureDisabled
	}
	remaining, err := remainingToday(policy)
	if err != nil {
		return registrationInviteCodeRow{}, err
	}
	if remaining <= 0 {
		return registrationInviteCodeRow{}, errDailyQuotaExhausted
	}

	day := shanghaiCalendarDay(time.Now())
	now := timeNowUTC()
	var row registrationInviteCodeRow
	for attempt := 0; attempt < 8; attempt++ {
		code, err := generateRegistrationInviteCode()
		if err != nil {
			return registrationInviteCodeRow{}, err
		}
		id := fmt.Sprintf("%d", generateSnowflakeID())
		_, err = db.Exec(`
			INSERT INTO auth_registration_invite_code (
				id, code, issuer_user_id, status, issued_day, created_at
			) VALUES (?, ?, ?, 'unused', ?, ?)`,
			id, code, issuerUserID, day, now,
		)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				continue
			}
			return registrationInviteCodeRow{}, err
		}
		row = registrationInviteCodeRow{
			ID:           id,
			Code:         code,
			IssuerUserID: issuerUserID,
			Status:       "unused",
			IssuedDay:    day,
			CreatedAt:    now,
		}
		publishInviteEvent(ctx, "REGISTRATION_INVITE_CODE_ISSUED", map[string]interface{}{
			"code_id":        id,
			"code":           code,
			"issuer_user_id": issuerUserID,
			"issued_day":     day,
		}, id)
		return row, nil
	}
	return registrationInviteCodeRow{}, fmt.Errorf("failed to generate unique invite code")
}

func lookupInviteCodeStatus(code string) (string, error) {
	var status string
	err := db.QueryRow(`
		SELECT status FROM auth_registration_invite_code WHERE code = ?`, code,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return "", errInviteCodeInvalid
	}
	if err != nil {
		return "", err
	}
	return status, nil
}

func peekInviteCodeUnused(code string) error {
	status, err := lookupInviteCodeStatus(code)
	if err != nil {
		return err
	}
	switch status {
	case "unused":
		return nil
	case "used":
		return errInviteCodeUsed
	case "revoked":
		return errInviteCodeRevoked
	default:
		return errInviteCodeInvalid
	}
}

// accessCodeInvitationSatisfied 判定分享链接 access_code 是否已满足邀请门禁：
// 返回 (true, nil) = 豁免（无需 invite_code）；(false, nil) = 需走 invite_code 校验；
// err = 校验失败（access_code 无效或门禁服务不可达，fail-closed）。
// 未携带 access_code 时直接返回 (false, nil)，不产生下游调用。
func accessCodeInvitationSatisfied(ctx context.Context, body map[string]interface{}) (bool, error) {
	accessCode := extractAccessCodeFromRegisterBody(body)
	if accessCode == "" {
		return false, nil
	}
	valid, reachable := validateAccessCodeInvitation(ctx, accessCode)
	if !reachable {
		return false, errInviteGateUnavailable
	}
	if valid {
		return true, nil
	}
	return false, errAccessCodeInvalid
}

func validateInviteBeforeRegister(ctx context.Context, body map[string]interface{}) error {
	policy, err := getRegistrationInvitePolicy()
	if err != nil {
		return err
	}
	if !policy.Enabled {
		return nil
	}
	satisfied, err := accessCodeInvitationSatisfied(ctx, body)
	if err != nil {
		return err
	}
	if satisfied {
		return nil
	}
	code := inviteCodeFromBody(body)
	if code == "" {
		return errInviteCodeRequired
	}
	return peekInviteCodeUnused(code)
}

func redeemInviteCode(ctx context.Context, code, newUserID string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return errInviteCodeRequired
	}
	now := timeNowUTC()
	res, err := db.Exec(`
		UPDATE auth_registration_invite_code
		SET status = 'used', redeemed_by_user_id = ?, redeemed_at = ?
		WHERE code = ? AND status = 'unused'`,
		newUserID, now, code,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 1 {
		publishInviteEvent(ctx, "REGISTRATION_INVITE_CODE_REDEEMED", map[string]interface{}{
			"code":                code,
			"redeemed_by_user_id": newUserID,
			"redeemed_at":         now,
		}, code)
		return nil
	}
	return peekInviteCodeUnused(code)
}

func redeemInviteAfterRegister(ctx context.Context, body map[string]interface{}, userID string) error {
	policy, err := getRegistrationInvitePolicy()
	if err != nil {
		return err
	}
	if !policy.Enabled {
		return nil
	}
	// access_code 豁免路径无 invite_code 可核销，跳过 redeem。
	satisfied, err := accessCodeInvitationSatisfied(ctx, body)
	if err != nil {
		return err
	}
	if satisfied {
		return nil
	}
	code := inviteCodeFromBody(body)
	return redeemInviteCode(ctx, code, userID)
}

func listRegistrationInviteCodesForIssuer(issuerUserID string) ([]registrationInviteCodeRow, error) {
	rows, err := db.Query(`
		SELECT id, code, issuer_user_id, status, issued_day, created_at,
		       redeemed_by_user_id, redeemed_at
		FROM auth_registration_invite_code
		WHERE issuer_user_id = ?
		ORDER BY created_at DESC`, issuerUserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []registrationInviteCodeRow{}
	for rows.Next() {
		row, err := scanRegistrationInviteCodeRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func listRegistrationInviteRelations(page, pageSize int) ([]registrationInviteRelation, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM auth_registration_invite_code`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(`
		SELECT code, issuer_user_id, status, issued_day, created_at,
		       redeemed_by_user_id, redeemed_at
		FROM auth_registration_invite_code
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []registrationInviteRelation{}
	for rows.Next() {
		var rel registrationInviteRelation
		var redeemedBy sql.NullString
		var redeemedAt sql.NullString
		if err := rows.Scan(
			&rel.Code, &rel.IssuerUserID, &rel.Status, &rel.IssuedDay, &rel.CreatedAt,
			&redeemedBy, &redeemedAt,
		); err != nil {
			return nil, 0, err
		}
		if redeemedBy.Valid {
			v := redeemedBy.String
			rel.RedeemedByUserID = &v
		}
		if redeemedAt.Valid {
			v := redeemedAt.String
			rel.RedeemedAt = &v
		}
		out = append(out, rel)
	}
	return out, total, rows.Err()
}

func scanRegistrationInviteCodeRow(rows *sql.Rows) (registrationInviteCodeRow, error) {
	var row registrationInviteCodeRow
	var redeemedBy sql.NullString
	var redeemedAt sql.NullString
	if err := rows.Scan(
		&row.ID, &row.Code, &row.IssuerUserID, &row.Status, &row.IssuedDay, &row.CreatedAt,
		&redeemedBy, &redeemedAt,
	); err != nil {
		return registrationInviteCodeRow{}, err
	}
	if redeemedBy.Valid {
		v := redeemedBy.String
		row.RedeemedByUserID = &v
	}
	if redeemedAt.Valid {
		v := redeemedAt.String
		row.RedeemedAt = &v
	}
	return row, nil
}
