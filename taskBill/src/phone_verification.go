package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"tracelog"
)

// ---- 手机号验证状态查询 ----

// handlePhoneVerificationStatus GET /api/tenant/{id}/billing/phone-verification-status/
func handlePhoneVerificationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	_ = tid // tenant ID available for future use
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "未认证", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	required := isSmsPhoneVerificationRequired()

	// Fetch user login methods from taskAuth to check phone binding
	hasPhone, phoneMasked, phoneE164 := fetchUserPhoneInfo(userID)

	// OPT-20260726-024: Always fetch real SMS verification status, not only when
	// policy is enabled. API contract should reflect actual state regardless of policy.
	smsVerified := fetchSmsVerifiedStatus(userID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"required":     required,
		"has_phone":    hasPhone,
		"phone_masked": phoneMasked,
		"phone_e164":   phoneE164,
		"sms_verified": smsVerified,
	})
}

// ---- 验证码校验并标记支付验证通过 ----

// handleVerifyPhoneCode POST /api/tenant/{id}/billing/verify-phone-code/
func handleVerifyPhoneCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	tid, ok := parseTenantID(r.URL.Path)
	if !ok {
		writeErrorJSON(w, http.StatusBadRequest, "租户信息不存在", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	_ = tid
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		writeErrorJSON(w, http.StatusUnauthorized, "未认证", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	body, err := readJSONBody(r)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	phone := strings.TrimSpace(stringField(body, "phone"))
	code := strings.TrimSpace(stringField(body, "code"))
	if phone == "" || code == "" {
		writeErrorJSON(w, http.StatusBadRequest, "须提供 phone 和 code", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// 1. Verify the code via taskAuth internal API
	valid, err := verifyCodeViaTaskAuth(userID, phone, code)
	if err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "验证服务暂不可用，请稍后重试", tracelog.TraceIDFromContext(r.Context()))
		return
	}
	if !valid {
		writeErrorJSON(w, http.StatusBadRequest, "验证码错误或已过期", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// 2. If phone not yet bound, bind it via taskAuth
	if err := ensurePhoneBound(userID, phone); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "绑定手机号失败，请稍后重试", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	// 3. Mark SMS verified for payment via taskAuth recharge-sms-gate
	if err := setSmsVerifiedViaTaskAuth(userID); err != nil {
		writeErrorJSON(w, http.StatusInternalServerError, "标记验证状态失败，请稍后重试", tracelog.TraceIDFromContext(r.Context()))
		return
	}

	phoneMasked := maskPhoneDisplay(phone)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":           true,
		"phone_masked": phoneMasked,
	})
}

// ---- taskAuth 内部调用 ----

func taskAuthRequest(ctx context.Context, method, path string, body map[string]interface{}) (int, []byte, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskAuthBaseURL), "/")
	if base == "" {
		return 0, nil, fmt.Errorf("taskAuth base URL not configured")
	}
	reqURL := base + path
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(b))
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(cfg.TaskAuthInternalSecret); secret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := smsGateHTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, raw, nil
}

func fetchUserPhoneInfo(userID string) (hasPhone bool, masked string, e164 string) {
	_, raw, err := taskAuthRequest(nil, http.MethodGet, "/api/accounts/users/"+url.PathEscape(userID)+"/", nil)
	if err != nil {
		return false, "", ""
	}
	var user struct {
		LoginMethods []struct {
			MethodType         string `json:"method_type"`
			Identifier         string `json:"identifier"`
			IsVerified         bool   `json:"is_verified"`
			CountryCallingCode string `json:"country_calling_code"`
		} `json:"login_methods"`
	}
	if err := json.Unmarshal(raw, &user); err != nil {
		return false, "", ""
	}
	for _, lm := range user.LoginMethods {
		if lm.MethodType == "phone" && lm.Identifier != "" {
			full := buildE164(lm.CountryCallingCode, lm.Identifier)
			return true, maskPhoneDisplay(full), full
		}
	}
	return false, "", ""
}

func fetchSmsVerifiedStatus(userID string) bool {
	statusCode, raw, err := taskAuthRequest(nil, http.MethodGet,
		"/api/internal/recharge-sms-gate/?user_id="+url.QueryEscape(userID), nil)
	if err != nil || statusCode != http.StatusOK {
		return false
	}
	var gate struct {
		SmsVerified bool `json:"sms_verified"`
	}
	if err := json.Unmarshal(raw, &gate); err != nil {
		return false
	}
	return gate.SmsVerified
}

func verifyCodeViaTaskAuth(userID, phone, code string) (bool, error) {
	statusCode, raw, err := taskAuthRequest(nil, http.MethodPost,
		"/api/internal/verification-code/verify/",
		map[string]interface{}{
			"phone":   phone,
			"code":    code,
			"user_id": userID,
		})
	if err != nil {
		return false, err
	}
	if statusCode != http.StatusOK {
		return false, nil
	}
	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return false, err
	}
	return result.Valid, nil
}

func setSmsVerifiedViaTaskAuth(userID string) error {
	statusCode, _, err := taskAuthRequest(nil, http.MethodPost,
		"/api/internal/recharge-sms-gate/",
		map[string]interface{}{
			"user_id": userID,
			"action":  "set_verified",
		})
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("taskAuth returned %d", statusCode)
	}
	return nil
}

func ensurePhoneBound(userID, phone string) error {
	// Parse E.164 phone to country code + national number
	cc, national := parseE164(phone)
	if national == "" {
		return fmt.Errorf("invalid phone format")
	}
	// Upsert phone login method via taskAuth
	statusCode, _, err := taskAuthRequest(nil, http.MethodPatch,
		"/api/internal/users/id/"+url.PathEscape(userID)+"/phone-login-method/",
		map[string]interface{}{
			"country_calling_code": cc,
			"national_number":      national,
		})
	if err != nil {
		return err
	}
	// 200 = ok, 409 = already bound to another user (unlikely since code was just verified)
	if statusCode != http.StatusOK {
		return fmt.Errorf("taskAuth returned %d", statusCode)
	}
	return nil
}

// ---- 工具函数 ----

func maskPhoneDisplay(phone string) string {
	// Mask middle digits: +8613900000000 → 139****0000
	s := strings.TrimPrefix(phone, "+")
	if strings.HasPrefix(s, "86") && len(s) > 6 {
		s = s[2:] // strip country code prefix for display
	}
	if len(s) < 7 {
		return s
	}
	return s[:3] + "****" + s[len(s)-4:]
}

func buildE164(countryCode, national string) string {
	cc := strings.TrimSpace(countryCode)
	nat := strings.TrimSpace(national)
	if nat == "" {
		return ""
	}
	// Fallback: if no country code but looks like Chinese mobile, assume +86
	if cc == "" {
		if strings.HasPrefix(nat, "1") && len(nat) == 11 {
			cc = "86"
		} else {
			return "+" + nat
		}
	}
	if strings.HasPrefix(cc, "+") {
		return cc + nat
	}
	return "+" + cc + nat
}

func parseE164(phone string) (countryCode, national string) {
	s := strings.TrimSpace(phone)
	if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	// Common country codes: 86 (China, 2-digit), 1 (US, 1-digit), 852 (HK, 3-digit)
	for _, l := range []int{3, 2, 1} {
		if len(s) > l {
			candidate := s[:l]
			rest := s[l:]
			if len(rest) >= 5 {
				return candidate, rest
			}
		}
	}
	return "", s
}
