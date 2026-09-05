package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"tracelog"
)

const (
	// DocumentKindPaymentTermsCanonical is SSOT for 支付服务条款.
	// Historical alias recharge_points is accepted when reading.
	DocumentKindPaymentTermsCanonical = "recharge_cents"
	DocumentKindPaymentTermsAlias     = "recharge_points"

	reasonConsentRequired           = "PAYMENT_TERMS_CONSENT_REQUIRED"
	reasonConsentInvalid            = "PAYMENT_TERMS_CONSENT_INVALID"
	reasonConsentServiceUnavailable = "PAYMENT_TERMS_CONSENT_UNAVAILABLE"
)

type consentGateError struct {
	Message    string
	ReasonCode string
}

func (e *consentGateError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.ReasonCode
}

func consentGateDenied(reasonCode, message string) *consentGateError {
	if message == "" {
		message = consentGateUserMessage(reasonCode)
	}
	return &consentGateError{Message: message, ReasonCode: reasonCode}
}

func consentGateUserMessage(reasonCode string) string {
	switch reasonCode {
	case reasonConsentRequired:
		return "请先阅读并同意支付服务条款后再支付"
	case reasonConsentInvalid:
		return "支付服务条款签署无效或已过期，请重新同意"
	case reasonConsentServiceUnavailable:
		return "支付条款校验暂不可用，请稍后重试"
	default:
		return "支付条款校验未通过"
	}
}

func writeConsentGateDenied(w http.ResponseWriter, err error) {
	ce, ok := err.(*consentGateError)
	if !ok || ce == nil {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"error":       consentGateUserMessage(reasonConsentServiceUnavailable),
			"reason_code": reasonConsentServiceUnavailable,
		})
		return
	}
	writeJSON(w, http.StatusForbidden, map[string]interface{}{
		"error":       ce.Message,
		"reason_code": ce.ReasonCode,
	})
}

func normalizePaymentTermsKind(kind string) string {
	k := strings.TrimSpace(kind)
	if k == DocumentKindPaymentTermsAlias {
		return DocumentKindPaymentTermsCanonical
	}
	return k
}

func isPaymentTermsKind(kind string) bool {
	return normalizePaymentTermsKind(kind) == DocumentKindPaymentTermsCanonical
}

// activePaymentTermsAgreementID returns the active payment-terms agreement id, or "" if none published.
func activePaymentTermsAgreementID(ctx context.Context) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `
		SELECT id FROM billing_license_agreements
		WHERE is_active = 1 AND document_kind IN (?, ?)
		ORDER BY CASE document_kind WHEN ? THEN 0 ELSE 1 END, created_at DESC
		LIMIT 1`,
		DocumentKindPaymentTermsCanonical, DocumentKindPaymentTermsAlias, DocumentKindPaymentTermsCanonical,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(id), nil
}

// userHasPaymentTermsConsent reports whether user has consented to the given agreement.
func userHasPaymentTermsConsent(ctx context.Context, userID, agreementID string) (consentID string, ok bool, err error) {
	userID = strings.TrimSpace(userID)
	agreementID = strings.TrimSpace(agreementID)
	if userID == "" || agreementID == "" {
		return "", false, nil
	}
	err = db.QueryRowContext(ctx, `
		SELECT id FROM billing_user_license_agreement_consents
		WHERE user_id = ? AND license_agreement_id = ?
		ORDER BY consented_at DESC LIMIT 1`, userID, agreementID,
	).Scan(&consentID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(consentID), consentID != "", nil
}

// validatePaymentConsentID ensures consentID belongs to userID and targets the active payment terms.
func validatePaymentConsentID(ctx context.Context, userID, consentID, activeAgreementID string) error {
	consentID = strings.TrimSpace(consentID)
	userID = strings.TrimSpace(userID)
	if consentID == "" {
		return consentGateDenied(reasonConsentRequired, "")
	}
	var gotUser, gotAgreement string
	err := db.QueryRowContext(ctx, `
		SELECT user_id, license_agreement_id FROM billing_user_license_agreement_consents WHERE id = ?`,
		consentID,
	).Scan(&gotUser, &gotAgreement)
	if err == sql.ErrNoRows {
		return consentGateDenied(reasonConsentInvalid, "")
	}
	if err != nil {
		return consentGateDenied(reasonConsentServiceUnavailable, err.Error())
	}
	if strings.TrimSpace(gotUser) != userID {
		return consentGateDenied(reasonConsentInvalid, "")
	}
	if activeAgreementID != "" && strings.TrimSpace(gotAgreement) != activeAgreementID {
		return consentGateDenied(reasonConsentInvalid, "签署版本已更新，请重新同意支付服务条款")
	}
	return nil
}

// checkPaymentTermsConsentGate enforces that the payer has consented to the active 支付服务条款.
// If no payment-terms document is published, the gate is a no-op (fail-open with structured log).
// Prefer an explicit consent_id from the client; otherwise accept any existing consent for the active agreement.
func checkPaymentTermsConsentGate(ctx context.Context, userID, consentID string) (resolvedConsentID string, err error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return "", consentGateDenied(reasonConsentRequired, "未登录")
	}
	activeID, aerr := activePaymentTermsAgreementID(ctx)
	if aerr != nil {
		tracelog.LogForwardStage(ctx, "payment_terms_consent_lookup_failed", map[string]any{
			"error": aerr.Error(),
		})
		return "", consentGateDenied(reasonConsentServiceUnavailable, "")
	}
	if activeID == "" {
		tracelog.LogForwardStage(ctx, "payment_terms_consent_skipped_no_active", map[string]any{
			"user_id": userID,
		})
		return "", nil
	}
	consentID = strings.TrimSpace(consentID)
	if consentID != "" {
		if verr := validatePaymentConsentID(ctx, userID, consentID, activeID); verr != nil {
			return "", verr
		}
		return consentID, nil
	}
	existing, ok, qerr := userHasPaymentTermsConsent(ctx, userID, activeID)
	if qerr != nil {
		return "", consentGateDenied(reasonConsentServiceUnavailable, qerr.Error())
	}
	if !ok {
		return "", consentGateDenied(reasonConsentRequired, "")
	}
	return existing, nil
}

// attachConsentToOrder persists consent_id on the resource order for admin audit.
func attachConsentToOrder(ctx context.Context, orderID int64, consentID string) error {
	consentID = strings.TrimSpace(consentID)
	if orderID <= 0 || consentID == "" {
		return nil
	}
	_, err := db.ExecContext(ctx, `
		UPDATE billing_resource_order SET consent_id = ? WHERE id = ? AND (consent_id IS NULL OR consent_id = '')`,
		consentID, orderID)
	if err != nil {
		return fmt.Errorf("attach consent_id to order: %w", err)
	}
	return nil
}
