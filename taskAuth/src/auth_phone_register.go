package main

import (
	"log"
	"log/slog"
	"net/http"
)

func handlePhoneRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}

	phone := strField(body, "phone")
	password := strField(body, "password")
	code := strField(body, "code")
	if phone == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"phone": []string{"手机号和密码是必填项"}})
		return
	}
	if len(password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"password": []string{"密码至少 8 位"}})
		return
	}

	// Validate verification code
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"code": "验证码是必填项"})
		return
	}
	ok, err := verifyPhoneVerificationCode(phone, code, "")
	if err != nil {
		log.Printf("[taskAuth] phone reg verify code: %v", err)
		writeError(w, r, http.StatusInternalServerError, "验证码校验失败")
		return
	}
	if !ok {
		writeError(w, r, http.StatusBadRequest, "验证码无效或已过期")
		return
	}

	// Normalize phone: split country calling code + national number
	canonical := canonicalPhoneForSMSAndLogin(phone)
	cc, nat := splitCountryCallingCodeAndNational(canonical)
	if cc == "" || nat == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"phone": "手机号无效"})
		return
	}

	// Region whitelist check
	if !isPhoneCountryCodeAllowed(cc) {
		writeErrorMap(w, r, http.StatusForbidden, map[string]interface{}{
			"error":  "phone_region_not_allowed",
			"detail": "该地区手机号暂不支持注册",
		})
		return
	}

	lm, err := findLoginMethodByPhone(cc, nat)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if lm != nil {
		active := true
		if lm.ObjectID != "" {
			active, _ = userIsActive(lm.ObjectID)
		}
		slog.InfoContext(r.Context(), "phone_register_rejected_existing",
			"user_id", lm.ObjectID, "is_active", active)
		writeErrorMap(w, r, http.StatusBadRequest, map[string]interface{}{
			"error":        "该手机号已被注册，请直接登录",
			"user_existed": true,
			"is_active":    active,
		})
		return
	}

	n, err := voidStalePhoneLoginMethods(cc, nat)
	if err != nil {
		slog.ErrorContext(r.Context(), "phone_register_void_stale_failed", "err", err)
		writeError(w, r, http.StatusInternalServerError, "db error")
		return
	}
	if n > 0 {
		slog.InfoContext(r.Context(), "phone_register_reclaimed_stale_identifier", "voided_count", n)
	}

	if err := validateInviteBeforeRegister(r.Context(), body); err != nil {
		writeInviteRegisterError(w, r, err)
		return
	}
	userID, err := createUserWithPhoneLogin(cc, nat, password, "")
	if err != nil {
		log.Printf("[taskAuth] phone register create: %v", err)
		writeError(w, r, http.StatusInternalServerError, "register failed")
		return
	}

	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, resolveClientIP(r))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "token error")
		return
	}

	publishUserCreatedAsync(userID, phone, "", "")
	accessCode := extractAccessCodeFromRegisterBody(body)
	if accessCode != "" {
		slog.Info("phone_register_referral_bind", "user_id", userID, "is_new_user", true)
		bindReferralAfterRegisterAsync(userID, accessCode)
	}

	if err := redeemInviteAfterRegister(r.Context(), body, userID); err != nil {
		registerInviteRollback(r.Context(), userID, body, err)
		writeInviteRegisterError(w, r, err)
		return
	}

	touchLastLogin(userID)
	recordSuccessfulLoginFromRequest(r, userID, "", "phone", "register", "")
	resp := buildLoginResponse(userID, token)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"token":   token,
		"user":    resp["user"],
		"message": "注册成功",
	})
}
