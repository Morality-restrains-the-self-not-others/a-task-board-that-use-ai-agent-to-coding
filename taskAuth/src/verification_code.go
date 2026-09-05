package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"
)

const verificationCodeTTL = 5 * time.Minute

type smsSendResult struct {
	Success      bool
	Provider     string
	RequestID    string
	ErrorCode    string
	ErrorMessage string
}

func generateNumericCode(length int) (string, error) {
	var b strings.Builder
	b.Grow(length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + n.Int64()))
	}
	return b.String(), nil
}

func smsProvider() string {
	p := strings.ToLower(strings.TrimSpace(os.Getenv("SMS_PROVIDER")))
	if p == "" {
		p = strings.ToLower(strings.TrimSpace(os.Getenv("TASKAUTH_SMS_PROVIDER")))
	}
	if p == "" {
		p = strings.ToLower(strings.TrimSpace(cfg.SMSProvider))
	}
	if p == "" {
		return "mock"
	}
	return p
}

func sendVerificationSMS(ctx context.Context, e164Phone, code, kind string) smsSendResult {
	provider := smsProvider()
	if kind == "" {
		kind = smsKindVerification
	}
	switch provider {
	case "none", "mock", "mock-for-tests", "disabled":
		log.Printf("[taskAuth-sms] provider=%s mock send phone=%s kind=%s", provider, e164Phone, kind)
		return smsSendResult{Success: true, Provider: "mock"}
	case "aliyun", "tencent", "tencentcloud":
		return sendCloudSMS(ctx, provider, e164Phone, code, kind)
	default:
		log.Printf("[taskAuth-sms] unsupported SMS_PROVIDER=%s", provider)
		return smsSendResult{
			Success:      false,
			Provider:     provider,
			ErrorCode:    "unsupported_provider",
			ErrorMessage: "不支持的短信渠道",
		}
	}
}

type sendVerificationOutcome struct {
	Message     string
	RequestID   string
	SMSProvider string
}

func sendPhoneVerificationCode(ctx context.Context, rawPhone, kind string) (*sendVerificationOutcome, error) {
	if kind == "" {
		kind = smsKindVerification
	}
	canonical := canonicalPhoneForSMSAndLogin(rawPhone)
	if canonical == "" {
		return nil, fmt.Errorf("手机号无效")
	}
	cc, national := splitCountryCallingCodeAndNational(canonical)
	if cc == "" || national == "" {
		return nil, fmt.Errorf("手机号无效")
	}
	// 检查手机号国家代码是否在允许区域内
	if !isPhoneCountryCodeAllowed(cc) {
		return nil, fmt.Errorf("该地区手机号暂不支持")
	}
	code, err := generateNumericCode(6)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expires := now.Add(verificationCodeTTL)
	id := generateSnowflakeID()
	_, err = db.Exec(`
		INSERT INTO auth_sms_verification_code
			(id, phone, country_calling_code, email, code, created_at, expires_at, is_used)
		VALUES (?, ?, ?, NULL, ?, ?, ?, 0)`,
		id, national, cc, code,
		now.Format("2006-01-02 15:04:05.000000"),
		expires.Format("2006-01-02 15:04:05.000000"),
	)
	if err != nil {
		return nil, fmt.Errorf("store code: %w", err)
	}
	e164 := composeE164(cc, national)
	sms := sendVerificationSMS(ctx, e164, code, kind)
	if !sms.Success {
		_, _ = db.Exec(`DELETE FROM auth_sms_verification_code WHERE id = ?`, id)
		return nil, fmt.Errorf("%s", formatSMSSendUserError(sms))
	}
	log.Printf("[taskAuth] verification code sent phone=%s%s provider=%s kind=%s id=%d", cc, national, sms.Provider, kind, id)
	out := &sendVerificationOutcome{
		Message:     "验证码发送成功",
		SMSProvider: sms.Provider,
		RequestID:   sms.RequestID,
	}
	return out, nil
}

func sendEmailVerificationCode(ctx context.Context, rawEmail, kind string) (*sendVerificationOutcome, error) {
	email := strings.ToLower(strings.TrimSpace(rawEmail))
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("邮箱无效")
	}
	if kind == "" {
		kind = smsKindVerification
	}
	code, err := generateNumericCode(6)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	expires := now.Add(verificationCodeTTL)
	id := generateSnowflakeID()
	_, err = db.Exec(`
		INSERT INTO auth_sms_verification_code
			(id, phone, country_calling_code, email, code, created_at, expires_at, is_used)
		VALUES (?, NULL, NULL, ?, ?, ?, ?, 0)`,
		id, email, code,
		now.Format("2006-01-02 15:04:05.000000"),
		expires.Format("2006-01-02 15:04:05.000000"),
	)
	if err != nil {
		return nil, fmt.Errorf("store code: %w", err)
	}
	subject := "您的验证码"
	template := "verification_code"
	if kind == smsKindPasswordReset {
		subject = "密码重置验证码"
		template = "password_reset"
	}
	method, err := publishEmailSent(ctx, email, subject, template, map[string]interface{}{"code": code})
	if err != nil {
		switch smsProvider() {
		case "none", "mock", "mock-for-tests", "disabled":
			log.Printf("[taskAuth] email mock skip dispatch email=%s err=%v", email, err)
		default:
			log.Printf("[taskAuth] verification code stored but EMAIL_SENT publish failed for email=%s: %v (code preserved in DB, retry later)", email, err)
			return nil, fmt.Errorf("邮件发送失败，请稍后重试")
		}
	}
	log.Printf("[taskAuth] verification email delivered method=%s email=%s kind=%s id=%d", method, email, kind, id)
	return &sendVerificationOutcome{Message: "验证码发送成功"}, nil
}

func findUnusedPhoneVerificationCodeID(rawPhone, code string) (int64, error) {
	canonical := canonicalPhoneForSMSAndLogin(rawPhone)
	if canonical == "" || strings.TrimSpace(code) == "" {
		return 0, nil
	}
	cc, national := splitCountryCallingCodeAndNational(canonical)
	if cc == "" || national == "" {
		return 0, nil
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var id int64
	err := db.QueryRow(`
		SELECT id FROM auth_sms_verification_code
		WHERE phone = ? AND country_calling_code = ? AND code = ?
			AND is_used = 0 AND expires_at > ?
		ORDER BY id DESC LIMIT 1`,
		national, cc, strings.TrimSpace(code), now,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func consumeVerificationCodeID(id int64, userID string) error {
	if id == 0 {
		return nil
	}
	var err error
	if userID != "" {
		_, err = db.Exec(`UPDATE auth_sms_verification_code SET is_used = 1, user_id = ? WHERE id = ?`, userID, id)
	} else {
		_, err = db.Exec(`UPDATE auth_sms_verification_code SET is_used = 1 WHERE id = ?`, id)
	}
	return err
}

func verifyPhoneVerificationCode(rawPhone, code, userID string) (bool, error) {
	id, err := findUnusedPhoneVerificationCodeID(rawPhone, code)
	if err != nil {
		return false, err
	}
	if id == 0 {
		return false, nil
	}
	if err := consumeVerificationCodeID(id, userID); err != nil {
		return false, err
	}
	return true, nil
}

func verifyEmailVerificationCode(rawEmail, code, userID string) (bool, error) {
	email := strings.ToLower(strings.TrimSpace(rawEmail))
	if email == "" || strings.TrimSpace(code) == "" {
		return false, nil
	}
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var id int64
	err := db.QueryRow(`
		SELECT id FROM auth_sms_verification_code
		WHERE email = ? AND code = ?
			AND is_used = 0 AND expires_at > ?
		ORDER BY id DESC LIMIT 1`,
		email, strings.TrimSpace(code), now,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if userID != "" {
		_, err = db.Exec(`UPDATE auth_sms_verification_code SET is_used = 1, user_id = ? WHERE id = ?`, userID, id)
	} else {
		_, err = db.Exec(`UPDATE auth_sms_verification_code SET is_used = 1 WHERE id = ?`, id)
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
