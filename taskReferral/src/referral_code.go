package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"
)

// ── Referral code errors ──

var (
	errReferralAlreadyPending              = errors.New("referral_already_pending")
	errReferralAlreadyActive               = errors.New("referral_already_active")
	errReferralCoolDown                    = errors.New("referral_cooldown")
	errReferralAppNotFound                 = errors.New("referral_app_not_found")
	errReferralAppNotPending               = errors.New("referral_app_not_pending")
	errReferralIntroRequired               = errors.New("请填写个人介绍")
	errReferralIntroTooShort               = errors.New("个人介绍至少 20 字")
	errReferralIntroTooLong                = errors.New("个人介绍最多 500 字")
	errReferralLegalNameRequired           = errors.New("请填写个人名称")
	errReferralLegalNameTooShort           = errors.New("个人名称至少 2 个字")
	errReferralLegalNameTooLong            = errors.New("个人名称最多 32 个字")
	errReferralLegalNameInvalid            = errors.New("个人名称须与微信实名一致，仅支持中文、英文字母或间隔号")
	errReferralIdentityBindConsentRequired = errors.New("请勾选同意绑定账号标识与用户身份信息")
	errReferralServiceAccountNotFollowed   = errors.New("请先关注服务号后再申请推荐资格")
)

const (
	personalIntroMinRunes = 20
	personalIntroMaxRunes = 500
)

const referralCodeValidityDays = 180 // 6 months

const referralPolicyKey = "global"

// ── Types ──

type referralCodeRow struct {
	ID                        int     `json:"id"`
	UserID                    string  `json:"user_id"`
	Status                    string  `json:"status"`
	AppliedAt                 string  `json:"applied_at"`
	ApprovedAt                *string `json:"approved_at,omitempty"`
	ExpiresAt                 *string `json:"expires_at,omitempty"`
	RejectedAt                *string `json:"rejected_at,omitempty"`
	RejectReason              string  `json:"reject_reason"`
	ReviewedBy                string  `json:"reviewed_by"`
	PersonalIntro             string  `json:"personal_intro"`
	LegalName                 string  `json:"legal_name"`
	IsActive                  bool    `json:"is_active"`
	CanRevoke                 bool    `json:"can_revoke"`
	StatusDisplay             string  `json:"status_display"`
	LastActionReason          string  `json:"last_action_reason"`
	ProfitSharingRatioDisplay string  `json:"profit_sharing_ratio_display"`
	ReferralRateDisplay       string  `json:"referral_rate_display"`
	ShareCode                 string  `json:"share_code,omitempty"` // OPT-20260824-006: 该用户默认分享码（referral_share_code is_default=1 active）
}

type referralCodeStatusResult struct {
	HasActiveCode        bool    `json:"has_active_code"`
	AccessCode           string  `json:"access_code"`
	ReferralCode         *string `json:"referral_code,omitempty"`
	ExpiresAt            *string `json:"expires_at,omitempty"`
	ExpiresInDays        *int    `json:"expires_in_days,omitempty"`
	ApplicationStatus    *string `json:"application_status,omitempty"`
	PendingSince         *string `json:"pending_since,omitempty"`
	RejectReason         *string `json:"reject_reason,omitempty"`
	RejectedAt           *string `json:"rejected_at,omitempty"`
	CanReapplyAfter      *string `json:"can_reapply_after,omitempty"`
	CanReapply           bool    `json:"can_reapply"`
	CanApply             bool    `json:"can_apply"`
	PolicyMode           string  `json:"policy_mode"`
	PolicyMessage        string  `json:"policy_message"`
	PersonalIntro        string  `json:"personal_intro,omitempty"`
	LegalName            string  `json:"legal_name,omitempty"`
	ReferralRateDisplay  string  `json:"referral_rate_display"`
	WechatReceiverStatus string  `json:"wechat_receiver_status,omitempty"`
	WechatReceiverReason string  `json:"wechat_receiver_reason,omitempty"`
	ServiceAccountBound  bool    `json:"service_account_bound"`
}

type referralPolicy struct {
	Mode      string `json:"mode"`
	Message   string `json:"message"`
	UpdatedAt string `json:"updated_at"`
}

type referralApplyResult struct {
	Status        string `json:"status"`
	Message       string `json:"message"`
	ReferralCode  string `json:"referral_code,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	ExpiresInDays int    `json:"expires_in_days,omitempty"`
}

// ── Policy ──

func getReferralPolicy() (referralPolicy, error) {
	var mode, message, updatedAt string
	err := db.QueryRow(`
		SELECT mode, message, updated_at
		FROM referral_policy
		WHERE singleton_key = ?`, referralPolicyKey,
	).Scan(&mode, &message, &updatedAt)
	if err == sql.ErrNoRows {
		return referralPolicy{Mode: "approval", Message: "", UpdatedAt: ""}, nil
	}
	if err != nil {
		return referralPolicy{}, err
	}
	return referralPolicy{Mode: mode, Message: message, UpdatedAt: updatedAt}, nil
}

func updateReferralPolicy(mode, message string) (referralPolicy, error) {
	if mode != "open" && mode != "approval" {
		return referralPolicy{}, fmt.Errorf("invalid mode: %s", mode)
	}
	now := timeNowUTC()
	_, err := db.Exec(`
		UPDATE referral_policy
		SET mode = ?, message = ?, updated_at = ?
		WHERE singleton_key = ?`,
		mode, message, now, referralPolicyKey,
	)
	if err != nil {
		return referralPolicy{}, err
	}
	return getReferralPolicy()
}

func normalizePersonalIntro(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(s)
	if n == 0 {
		return "", errReferralIntroRequired
	}
	if n < personalIntroMinRunes {
		return "", errReferralIntroTooShort
	}
	if n > personalIntroMaxRunes {
		return "", errReferralIntroTooLong
	}
	return s, nil
}

// ── Application logic ──

func applyReferralCode(userID, personalIntro, legalName string, identityBindConsent bool) (referralApplyResult, error) {
	intro, err := normalizePersonalIntro(personalIntro)
	if err != nil {
		return referralApplyResult{}, err
	}
	name, err := normalizeLegalName(legalName)
	if err != nil {
		return referralApplyResult{}, err
	}
	if !identityBindConsent {
		return referralApplyResult{}, errReferralIdentityBindConsentRequired
	}
	if !userHasServiceAccountBound(userID) {
		return referralApplyResult{}, errReferralServiceAccountNotFollowed
	}

	now := timeNowUTC()

	existing, _ := getActiveOrPendingReferralCode(userID)
	if existing != nil {
		if existing.Status == "pending" {
			return referralApplyResult{}, errReferralAlreadyPending
		}
		if existing.Status == "approved" {
			if existing.ExpiresAt != nil {
				expires, err := time.Parse(time.RFC3339, *existing.ExpiresAt)
				if err == nil && expires.After(time.Now()) {
					days := int(expires.Sub(time.Now()).Hours() / 24)
					if days < 1 {
						days = 1
					}
					return referralApplyResult{}, fmt.Errorf("您的推荐资格仍在有效期内（剩余约 %d 天），无需重复申请", days)
				}
			} else {
				return referralApplyResult{}, errReferralAlreadyActive
			}
		}
	}

	lastRejected := getLastRejectedReferralCode(userID)
	if lastRejected != nil && lastRejected.RejectedAt != nil {
		rejectedTime, err := time.Parse(time.RFC3339, *lastRejected.RejectedAt)
		if err == nil {
			cooldownEnd := rejectedTime.Add(7 * 24 * time.Hour)
			if time.Now().Before(cooldownEnd) {
				days := int(cooldownEnd.Sub(time.Now()).Hours() / 24)
				if days < 1 {
					days = 1
				}
				return referralApplyResult{}, fmt.Errorf("您的上次推荐资格申请被拒绝，请在 %d 天后重新申请", days)
			}
		}
	}

	policy, err := getReferralPolicy()
	if err != nil {
		return referralApplyResult{}, err
	}

	if policy.Mode == "open" {
		expires := time.Now().AddDate(0, 6, 0)
		expiresStr := expires.Format(time.RFC3339)
		_, err := db.Exec(`
			INSERT INTO referral_code (
				user_id, status, applied_at, approved_at, expires_at,
				reject_reason, reviewed_by, personal_intro, legal_name, identity_bind_consented_at
			) VALUES (?, 'approved', ?, ?, ?, '', '', ?, ?, ?)`,
			userID, now, now, expiresStr, intro, name, now,
		)
		if err != nil {
			return referralApplyResult{}, err
		}
		shareCode, err := ensureUserShareCode(userID)
		if err != nil {
			return referralApplyResult{}, err
		}
		ensureWechatProfitSharingReceiver(userID)
		return referralApplyResult{
			Status:        "approved",
			Message:       "推荐资格已自动开通（限额放开模式）",
			ReferralCode:  shareCode,
			ExpiresAt:     expiresStr,
			ExpiresInDays: referralCodeValidityDays,
		}, nil
	}

	_, err = db.Exec(`
		INSERT INTO referral_code (
			user_id, status, applied_at, reject_reason, reviewed_by, personal_intro,
			legal_name, identity_bind_consented_at
		) VALUES (?, 'pending', ?, '', '', ?, ?, ?)`,
		userID, now, intro, name, now,
	)
	if err != nil {
		return referralApplyResult{}, err
	}
	return referralApplyResult{
		Status:  "pending",
		Message: "推荐资格申请已提交，等待管理员审批",
	}, nil
}

func getReferralCodeStatus(userID string) (referralCodeStatusResult, error) {
	shareCode, err := ensureUserShareCode(userID)
	if err != nil {
		return referralCodeStatusResult{}, err
	}
	now := time.Now()
	result := referralCodeStatusResult{
		CanApply:            true,
		AccessCode:          shareCode,
		ReferralRateDisplay: configuredReferralRateDisplay(),
		ServiceAccountBound: userHasServiceAccountBound(userID),
	}

	policy, err := getReferralPolicy()
	if err != nil {
		log.Printf("[taskReferral] referral status policy: %v", err)
	}
	result.PolicyMode = policy.Mode
	result.PolicyMessage = policy.Message

	active := getActiveReferralCode(userID)
	if active != nil {
		result.HasActiveCode = true
		result.ReferralCode = &shareCode
		result.ApplicationStatus = strPtr("approved")
		result.PersonalIntro = active.PersonalIntro
		result.LegalName = active.LegalName
		result.CanApply = false
		if active.ExpiresAt != nil {
			result.ExpiresAt = active.ExpiresAt
			expires, err := time.Parse(time.RFC3339, *active.ExpiresAt)
			if err == nil && expires.After(now) {
				days := int(expires.Sub(now).Hours() / 24)
				if days < 1 {
					days = 1
				}
				result.ExpiresInDays = &days
			}
		}
		recv := ensureWechatProfitSharingReceiver(userID)
		result.WechatReceiverStatus = recv.Status
		result.WechatReceiverReason = recv.Reason
		return result, nil
	}

	pending := getPendingReferralCode(userID)
	if pending != nil {
		st := "pending"
		result.ApplicationStatus = &st
		result.PersonalIntro = pending.PersonalIntro
		result.LegalName = pending.LegalName
		result.CanApply = false
		if pending.AppliedAt != "" {
			result.PendingSince = &pending.AppliedAt
		}
		return result, nil
	}

	lastRejected := getLastRejectedReferralCode(userID)
	if lastRejected != nil {
		st := "rejected"
		result.ApplicationStatus = &st
		result.PersonalIntro = lastRejected.PersonalIntro
		result.LegalName = lastRejected.LegalName
		if lastRejected.RejectReason != "" {
			result.RejectReason = &lastRejected.RejectReason
		}
		result.RejectedAt = lastRejected.RejectedAt
		if lastRejected.RejectedAt != nil {
			rejectedTime, err := time.Parse(time.RFC3339, *lastRejected.RejectedAt)
			if err == nil {
				cooldownEnd := rejectedTime.Add(7 * 24 * time.Hour)
				if now.Before(cooldownEnd) {
					iso := cooldownEnd.Format(time.RFC3339)
					result.CanReapplyAfter = &iso
				} else {
					result.CanReapply = true
				}
			}
		}
		return result, nil
	}

	return result, nil
}

func approveReferralApplication(ctx context.Context, appID int, reviewerID, reason, idemKey string) (map[string]interface{}, error) {
	app, err := getReferralApplicationByID(appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errReferralAppNotFound
	}
	// 与 revoke 同口径：同一 Idempotency-Key 重放返回已处理结果，不重复写审计。
	idemKey = newAuditIdempotencyKey(idemKey, appID, "approve")
	if existing, lookErr := getAuditByIdempotencyKey(idemKey); lookErr == nil && existing != nil {
		if existing.ApplicationID == appID && existing.Action == "approve" {
			return map[string]interface{}{
				"status":  "approved",
				"message": "already_approved",
			}, nil
		}
		return nil, errReferralIdempotencyClash
	}
	if app.Status != "pending" {
		return nil, errReferralAppNotPending
	}
	reason, err = validateReviewReason(reason)
	if err != nil {
		return nil, err
	}

	now := timeNowUTC()
	expires := time.Now().AddDate(0, 6, 0)
	expiresStr := expires.Format(time.RFC3339)

	_, err = db.Exec(`
		UPDATE referral_code
		SET status = 'approved', approved_at = ?, expires_at = ?, reviewed_by = ?, reject_reason = ?
		WHERE id = ?`, now, expiresStr, reviewerID, reason, appID,
	)
	if err != nil {
		return nil, err
	}
	if _, err := insertQualificationAudit(ctx, qualificationAuditEntry{
		ApplicationID:  appID,
		UserID:         app.UserID,
		Action:         "approve",
		Reason:         reason,
		OperatorID:     reviewerID,
		IdempotencyKey: idemKey,
	}); err != nil {
		return nil, err
	}

	publishReferralEvent(ctx, "REFERRAL_APPROVED", map[string]interface{}{
		"user_id":     app.UserID,
		"reviewed_by": reviewerID,
		"app_id":      appID,
		"expires_at":  expiresStr,
		"reason_len":  utf8.RuneCountInString(reason),
	}, app.UserID)

	shareCode, err := ensureUserShareCode(app.UserID)
	if err != nil {
		return nil, err
	}
	ensureWechatProfitSharingReceiver(app.UserID)
	return map[string]interface{}{
		"status":          "approved",
		"message":         "推荐资格已审批通过",
		"referral_code":   shareCode,
		"expires_at":      expiresStr,
		"expires_in_days": referralCodeValidityDays,
	}, nil
}

func rejectReferralApplication(ctx context.Context, appID int, reviewerID, reason, idemKey string) (map[string]interface{}, error) {
	app, err := getReferralApplicationByID(appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errReferralAppNotFound
	}
	// 与 revoke 同口径：同一 Idempotency-Key 重放返回已处理结果，不重复写审计。
	idemKey = newAuditIdempotencyKey(idemKey, appID, "reject")
	if existing, lookErr := getAuditByIdempotencyKey(idemKey); lookErr == nil && existing != nil {
		if existing.ApplicationID == appID && existing.Action == "reject" {
			return map[string]interface{}{
				"status":  "rejected",
				"message": "already_rejected",
			}, nil
		}
		return nil, errReferralIdempotencyClash
	}
	if app.Status != "pending" {
		return nil, errReferralAppNotPending
	}
	reason, err = validateReviewReason(reason)
	if err != nil {
		return nil, err
	}

	now := timeNowUTC()
	_, err = db.Exec(`
		UPDATE referral_code
		SET status = 'rejected', rejected_at = ?, reject_reason = ?, reviewed_by = ?
		WHERE id = ?`, now, reason, reviewerID, appID,
	)
	if err != nil {
		return nil, err
	}
	if _, err := insertQualificationAudit(ctx, qualificationAuditEntry{
		ApplicationID:  appID,
		UserID:         app.UserID,
		Action:         "reject",
		Reason:         reason,
		OperatorID:     reviewerID,
		IdempotencyKey: idemKey,
	}); err != nil {
		return nil, err
	}

	publishReferralEvent(ctx, "REFERRAL_REJECTED", map[string]interface{}{
		"user_id":     app.UserID,
		"reviewed_by": reviewerID,
		"app_id":      appID,
		"reason_len":  utf8.RuneCountInString(reason),
	}, app.UserID)

	return map[string]interface{}{
		"status":  "rejected",
		"message": "已拒绝该推荐资格申请",
	}, nil
}
