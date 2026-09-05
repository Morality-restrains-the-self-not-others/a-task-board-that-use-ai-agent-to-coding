package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"tracelog"
)

const (
	reasonSmsNotVerified        = "SMS_NOT_VERIFIED"
	reasonSmsServiceUnavailable = "SMS_SERVICE_UNAVAILABLE"
	reasonSmsNotRequired        = "SMS_NOT_REQUIRED"
)

var smsGateHTTP = tracelog.DirectClient(10 * time.Second)

type smsGateResult struct {
	SmsVerified  bool   `json:"sms_verified"`
	PendingPhone string `json:"pending_phone,omitempty"`
}

type smsGateError struct {
	Message    string
	ReasonCode string
}

func (e *smsGateError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.ReasonCode
}

func smsGateDenied(reasonCode, message string) *smsGateError {
	if message == "" {
		message = smsGateUserMessage(reasonCode)
	}
	return &smsGateError{Message: message, ReasonCode: reasonCode}
}

func smsGateUserMessage(reasonCode string) string {
	switch reasonCode {
	case reasonSmsNotVerified:
		return "请先完成手机号短信验证"
	case reasonSmsServiceUnavailable:
		return "手机验证服务暂不可用，请稍后重试"
	default:
		return "手机验证未通过"
	}
}

func writeSmsGateDenied(w http.ResponseWriter, err error) {
	se, ok := err.(*smsGateError)
	if !ok || se == nil {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"error":       smsGateUserMessage(reasonSmsServiceUnavailable),
			"reason_code": reasonSmsServiceUnavailable,
		})
		return
	}
	writeJSON(w, http.StatusForbidden, map[string]interface{}{
		"error":       se.Message,
		"reason_code": se.ReasonCode,
	})
}

// isSmsPhoneVerificationRequired returns whether the system feature policy
// requires SMS phone verification before payment creation.
// Checks config first (env TASKBILL_SMS_GATE: live|off), then consults Django
// auth_system_feature_policy.enable_recharge_phone_verification as a second layer.
// If Django says disabled, the gate is skipped regardless of config.
// OPT-20260726-015: Added Django feature policy query for tenant-level grayscale control.
func isSmsPhoneVerificationRequired() bool {
	mode := strings.ToLower(strings.TrimSpace(cfg.SmsGateMode))
	if mode == "" {
		mode = "live"
	}
	if mode == "off" {
		return false
	}

	// Second layer: query Django auth_system_feature_policy (cached 60s)
	policy, err := fetchSmsFeaturePolicyCached()
	if err == nil && policy != nil && !policy.EnableRechargePhoneVerification {
		return false
	}
	return true
}

// smsFeaturePolicy holds the relevant subset of the system feature policy
// for SMS gate decisions (phone verification + region whitelist).
type smsFeaturePolicy struct {
	EnableRechargePhoneVerification bool     `json:"enable_recharge_phone_verification"`
	AllowedPhoneCountryCodes        []string `json:"allowed_phone_country_codes"`
}

var (
	smsFeaturePolicyCache    *smsFeaturePolicy
	smsFeaturePolicyCachedAt time.Time
	smsFeaturePolicyMu       sync.Mutex
	smsFeaturePolicyTTL      = 60 * time.Second
)

func fetchSmsFeaturePolicyCached() (*smsFeaturePolicy, error) {
	smsFeaturePolicyMu.Lock()
	defer smsFeaturePolicyMu.Unlock()
	if smsFeaturePolicyCache != nil && time.Since(smsFeaturePolicyCachedAt) < smsFeaturePolicyTTL {
		return smsFeaturePolicyCache, nil
	}
	policy, err := fetchSmsFeaturePolicy(nil)
	if policy != nil {
		smsFeaturePolicyCache = policy
		smsFeaturePolicyCachedAt = time.Now()
	}
	return policy, err
}

// checkSmsPaymentGate validates that the user has completed SMS phone
// verification before allowing payment creation. Calls taskAuth's internal
// recharge-sms-gate endpoint.
// OPT-20260726-025: Logs region whitelist status for audit visibility.
func checkSmsPaymentGate(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return smsGateDenied(reasonSmsNotVerified, "未登录")
	}

	if !isSmsPhoneVerificationRequired() {
		return nil
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.SmsGateMode))
	if mode == "" {
		mode = "live"
	}
	if mode == "off" {
		return nil
	}

	// Log region whitelist state for audit (non-blocking)
	if policy, err := fetchSmsFeaturePolicyCached(); err == nil && policy != nil && len(policy.AllowedPhoneCountryCodes) > 0 {
		log.Printf("[taskBill] sms-gate: region whitelist active, allowed_country_codes=%v, user=%s",
			policy.AllowedPhoneCountryCodes, userID)
	}

	gate, err := fetchSmsPaymentGate(ctx, mode, userID)
	if err != nil {
		return err
	}
	if !gate.SmsVerified {
		return smsGateDenied(reasonSmsNotVerified, "")
	}
	return nil
}

func fetchSmsPaymentGate(ctx context.Context, mode, userID string) (smsGateResult, error) {
	if mode == "mock" {
		// Default mock: allow — keeps existing tests working without taskAuth.
		return smsGateResult{SmsVerified: true}, nil
	}

	base := strings.TrimRight(strings.TrimSpace(cfg.TaskAuthBaseURL), "/")
	if base == "" {
		return smsGateResult{}, smsGateDenied(reasonSmsServiceUnavailable, "")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	q := url.Values{}
	q.Set("user_id", userID)
	reqURL := base + "/api/internal/recharge-sms-gate/?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return smsGateResult{}, smsGateDenied(reasonSmsServiceUnavailable, "")
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(cfg.TaskAuthInternalSecret); secret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", secret)
	}
	resp, err := smsGateHTTP.Do(req)
	if err != nil {
		return smsGateResult{}, smsGateDenied(reasonSmsServiceUnavailable, "")
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return smsGateResult{}, smsGateDenied(reasonSmsServiceUnavailable, "")
	}
	var gate smsGateResult
	if err := json.Unmarshal(raw, &gate); err != nil {
		return smsGateResult{}, smsGateDenied(reasonSmsServiceUnavailable, "")
	}
	return gate, nil
}

// Fetch system feature policy from Django to check if SMS phone verification
// is enabled. Used as a fallback when the taskAuth SMS gate is unavailable.
// Returns the full policy subset including allowed_phone_country_codes for
// region-aware logging (OPT-20260726-025).
// fetchSmsFeaturePolicy previously fetched /api/public/system-feature-policy/ from Django.
// OPT-052: Django saas-backend retired 2026-07-30. Returns nil to use safe defaults
// (isSmsPhoneVerificationRequired returns true → verification required).
func fetchSmsFeaturePolicy(ctx context.Context) (*smsFeaturePolicy, error) {
	_ = ctx
	return nil, nil
}
