package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tracelog"
)

func (a *App) handleVendorApplicationPhoneStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	applicant := a.resolveSaasApplicant(r)
	if applicant == nil || applicant.UID <= 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "未认证"})
		return
	}
	uid := strconv.FormatInt(applicant.UID, 10)
	hasPhone, masked, e164 := a.fetchApplicantPhoneInfo(r, uid)
	verified, err := vendorContactVerifiedFn(a, r.Context(), uid)
	if err != nil {
		logWarn(r.Context(), "event=VendorPhoneStatusSmsGateFailed user_id=%d err=%v", applicant.UID, err)
		verified = false
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"has_phone":    hasPhone,
		"phone_masked": masked,
		"phone_e164":   e164,
		"sms_verified": verified,
		"required":     true,
	})
}

func (a *App) handleVendorApplicationSendSMS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	applicant := a.resolveSaasApplicant(r)
	if applicant == nil || applicant.UID <= 0 {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"detail": "未认证"})
		return
	}
	var body struct {
		Phone string `json:"phone"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
		return
	}
	phone := strings.TrimSpace(body.Phone)
	if phone == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "必须提供手机号"})
		return
	}
	status, raw, err := a.taskAuthJSON(r, http.MethodPost, "/api/accounts/users/send_verification_code/", map[string]any{"phone": phone})
	if err != nil {
		logWarn(r.Context(), "event=VendorSendSmsFailed user_id=%d err=%v", applicant.UID, err)
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": "验证码发送服务暂不可用，请稍后重试"})
		return
	}
	if status != http.StatusOK {
		detail := jsonDetailOr(raw, "发送失败")
		writeJSON(w, status, map[string]any{"detail": detail})
		return
	}
	logInfo("event=VendorApplySmsSent user_id=%d", applicant.UID)
	writeJSON(w, http.StatusOK, map[string]any{"message": "验证码已发送"})
}

func (a *App) handleVendorApplicationVerifyPhone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"detail": "method not allowed"})
		return
	}
	uid, _, ok := a.requireVendorApplicant(w, r)
	if !ok {
		return
	}
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "无效 JSON"})
		return
	}
	phone := strings.TrimSpace(body.Phone)
	code := strings.TrimSpace(body.Code)
	if phone == "" || code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "须提供 phone 和 code"})
		return
	}
	uidStr := strconv.FormatInt(uid, 10)
	status, raw, err := a.taskAuthJSON(r, http.MethodPost, "/api/internal/verification-code/verify/", map[string]any{
		"phone": phone, "code": code, "user_id": uidStr,
	})
	if err != nil {
		logWarn(r.Context(), "event=VendorVerifyPhoneFailed user_id=%d err=%v", uid, err)
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": "验证服务暂不可用，请稍后重试"})
		return
	}
	if status != http.StatusOK {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": jsonDetailOr(raw, "验证码错误或已过期")})
		return
	}
	var parsed struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || !parsed.Valid {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "验证码错误或已过期"})
		return
	}
	cc, national := parseApplicantE164(phone)
	if national == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"detail": "手机号格式无效"})
		return
	}
	bindStatus, _, bindErr := a.taskAuthJSON(r, http.MethodPatch, "/api/internal/users/id/"+url.PathEscape(uidStr)+"/phone-login-method/", map[string]any{
		"country_calling_code": cc,
		"national_number":      national,
	})
	if bindErr != nil || (bindStatus != http.StatusOK && bindStatus != http.StatusConflict) {
		logWarn(r.Context(), "event=VendorBindPhoneFailed user_id=%d status=%d err=%v", uid, bindStatus, bindErr)
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": "绑定手机号失败，请稍后重试"})
		return
	}
	gateStatus, _, gateErr := a.taskAuthJSON(r, http.MethodPost, "/api/internal/recharge-sms-gate/", map[string]any{
		"user_id": uidStr, "action": "set_verified",
	})
	if gateErr != nil || gateStatus != http.StatusOK {
		logWarn(r.Context(), "event=VendorSmsGateSetFailed user_id=%d status=%d err=%v", uid, gateStatus, gateErr)
		writeJSON(w, http.StatusBadGateway, map[string]any{"detail": "标记验证状态失败，请稍后重试"})
		return
	}
	logInfo("event=VendorApplyPhoneVerified user_id=%d", uid)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "phone_masked": maskApplicantPhone(phone)})
}

func (a *App) fetchApplicantPhoneInfo(r *http.Request, userID string) (hasPhone bool, masked, e164 string) {
	status, raw, err := a.taskAuthJSON(r, http.MethodGet, "/api/accounts/users/"+url.PathEscape(userID)+"/", nil)
	if err != nil || status != http.StatusOK {
		return false, "", ""
	}
	var user struct {
		LoginMethods []struct {
			MethodType         string `json:"method_type"`
			Identifier         string `json:"identifier"`
			CountryCallingCode string `json:"country_calling_code"`
		} `json:"login_methods"`
	}
	if err := json.Unmarshal(raw, &user); err != nil {
		return false, "", ""
	}
	for _, lm := range user.LoginMethods {
		if lm.MethodType == "phone" && lm.Identifier != "" {
			full := buildApplicantE164(lm.CountryCallingCode, lm.Identifier)
			return true, maskApplicantPhone(full), full
		}
	}
	return false, "", ""
}

func (a *App) taskAuthJSON(r *http.Request, method, path string, body map[string]any) (int, []byte, error) {
	if a == nil || a.Cfg == nil {
		return 0, nil, fmt.Errorf("missing config")
	}
	base := strings.TrimRight(strings.TrimSpace(a.Cfg.TaskAuthBaseURL), "/")
	if base == "" {
		return 0, nil, fmt.Errorf("taskAuth base URL not configured")
	}
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	ctx := r.Context()
	req, err := http.NewRequestWithContext(ctx, method, base+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if secret := strings.TrimSpace(a.Cfg.TaskAuthInternalSecret); secret != "" {
		req.Header.Set("X-TaskAuth-Internal-Secret", secret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, raw, nil
}

func jsonDetailOr(raw []byte, fallback string) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err == nil {
		for _, k := range []string{"detail", "message", "error"} {
			if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return fallback
}

func maskApplicantPhone(phone string) string {
	s := strings.TrimPrefix(strings.TrimSpace(phone), "+")
	if strings.HasPrefix(s, "86") && len(s) > 6 {
		s = s[2:]
	}
	if len(s) < 7 {
		return s
	}
	return s[:3] + "****" + s[len(s)-4:]
}

func buildApplicantE164(countryCode, national string) string {
	cc := strings.TrimSpace(countryCode)
	nat := strings.TrimSpace(national)
	if nat == "" {
		return ""
	}
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

func parseApplicantE164(phone string) (countryCode, national string) {
	s := strings.ReplaceAll(strings.TrimSpace(phone), " ", "")
	if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if strings.HasPrefix(s, "86") && len(s) == 13 {
		return "86", s[2:]
	}
	if len(s) == 11 && strings.HasPrefix(s, "1") {
		return "86", s
	}
	for _, l := range []int{3, 2, 1} {
		if len(s) > l {
			rest := s[l:]
			if len(rest) >= 5 {
				return s[:l], rest
			}
		}
	}
	return "", s
}
