package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const inboxReasonLabel = "理由："

const (
	MinImpersonationReasonRunes = 8
	MaxImpersonationReasonRunes = 500
)

// NestedImpersonationClientMessage is the HTTP 409 detail for nested
// impersonation (a different target while a session is already open).
// Keep in sync with taskFE useImpersonateUser.js isAlreadyImpersonatingDetail.
const NestedImpersonationClientMessage = "已在模拟登录中，请先退出"

var (
	ErrSelfImpersonation     = errors.New("cannot impersonate self")
	ErrNestedImpersonation   = errors.New("already impersonating")
	ErrTargetInactive        = errors.New("target user cannot log in")
	ErrPrivilegeEscalation   = errors.New("cannot impersonate platform admin")
	ErrMissingIdempotencyKey = errors.New("idempotency key required")
	ErrActorOrTargetRequired = errors.New("actor and target required")
	ErrNotImpersonating      = errors.New("not impersonating")
	ErrMissingReason         = errors.New("impersonation reason required")
	ErrReasonTooLong         = errors.New("impersonation reason too long")
	ErrReasonIsPromptText    = errors.New("impersonation reason must not be the modal prompt or placeholder text")
)

// forbiddenImpersonationReasons are the exact strings shown in the reason
// modal (explanation paragraph and placeholder). They meet the 8-char length
// gate but carry no real justification; storing them as the audit/inbox
// reason makes the notice look system-generated. Keep in sync with
// ImpersonateReasonModal.vue exports.
var forbiddenImpersonationReasons = map[string]struct{}{
	"请填写本次模拟登录的理由。该理由会写入审计并通知被模拟用户。": {},
	"例如：排查线上工单 T-12345":              {},
}

// StartImpersonationDecision is the authorization snapshot for starting a session.
type StartImpersonationDecision struct {
	ActorUserID               string
	TargetUserID              string
	ActorHasPlatformManage    bool
	TargetIsPlatformAdmin     bool
	ActorAlreadyImpersonating bool
	TargetActive              bool
	IdempotencyKey            string
	Reason                    string
}

// ImpersonationNoticeBody is the inbox narrative without the audit reason.
// The reason is stored and shown via the dedicated reason field.
func ImpersonationNoticeBody(actorUserID string) string {
	return fmt.Sprintf("系统管理员（用户 ID：%s）以你的身份登录了本平台。", strings.TrimSpace(actorUserID))
}

// StripEmbeddedInboxReason removes a trailing "理由：{reason}" from historical
// bodies that duplicated the dedicated reason column.
func StripEmbeddedInboxReason(body, reason string) string {
	body = strings.TrimRight(body, "\r\n")
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return body
	}
	suffix := inboxReasonLabel + reason
	if !strings.HasSuffix(body, suffix) {
		return body
	}
	return strings.TrimRight(strings.TrimSuffix(body, suffix), "\r\n")
}

// NormalizeImpersonationReason trims surrounding whitespace.
func NormalizeImpersonationReason(reason string) string {
	return strings.TrimSpace(reason)
}

// ValidateImpersonationReason enforces 8–500 Unicode characters after trim
// and rejects strings that merely echo the modal prompt/placeholder.
func ValidateImpersonationReason(reason string) error {
	normalized := NormalizeImpersonationReason(reason)
	if _, forbidden := forbiddenImpersonationReasons[normalized]; forbidden {
		return ErrReasonIsPromptText
	}
	n := utf8.RuneCountInString(normalized)
	if n < MinImpersonationReasonRunes {
		return ErrMissingReason
	}
	if n > MaxImpersonationReasonRunes {
		return ErrReasonTooLong
	}
	return nil
}

// ValidateStart returns a domain error if the impersonation must not start.
func ValidateStart(d StartImpersonationDecision) error {
	if strings.TrimSpace(d.IdempotencyKey) == "" {
		return ErrMissingIdempotencyKey
	}
	actor := strings.TrimSpace(d.ActorUserID)
	target := strings.TrimSpace(d.TargetUserID)
	if actor == "" || target == "" {
		return ErrActorOrTargetRequired
	}
	if err := ValidateImpersonationReason(d.Reason); err != nil {
		return err
	}
	if actor == target {
		return ErrSelfImpersonation
	}
	if d.ActorAlreadyImpersonating {
		return ErrNestedImpersonation
	}
	if !d.TargetActive {
		return ErrTargetInactive
	}
	if d.TargetIsPlatformAdmin && !d.ActorHasPlatformManage {
		return ErrPrivilegeEscalation
	}
	return nil
}
