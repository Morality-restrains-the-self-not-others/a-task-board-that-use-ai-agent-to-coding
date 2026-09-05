package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPhoneNormalizeCN(t *testing.T) {
	c := canonicalPhoneForSMSAndLogin("13800138000")
	if c != "13800138000" {
		t.Fatalf("got %q", c)
	}
	cc, nat := splitCountryCallingCodeAndNational(c)
	if cc != "+86" || nat != "13800138000" {
		t.Fatalf("split got %q %q", cc, nat)
	}
	if composeE164(cc, nat) != "+8613800138000" {
		t.Fatalf("e164 %q", composeE164(cc, nat))
	}
}

func TestSendAndVerifyPhoneVerificationCode(t *testing.T) {
	setupAuthTestDB(t)
	prev := os.Getenv("SMS_PROVIDER")
	_ = os.Setenv("SMS_PROVIDER", "mock")
	t.Cleanup(func() { _ = os.Setenv("SMS_PROVIDER", prev) })

	out, err := sendPhoneVerificationCode(httptest.NewRequest(http.MethodPost, "/", nil).Context(), "13900001111", smsKindVerification)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if out.Message == "" {
		t.Fatal("empty message")
	}

	var code string
	err = db.QueryRow(`
		SELECT code FROM auth_sms_verification_code
		WHERE phone = ? AND country_calling_code = ? AND is_used = 0
		ORDER BY id DESC LIMIT 1`, "13900001111", "+86").Scan(&code)
	if err != nil {
		t.Fatalf("read code: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("code len %d", len(code))
	}

	ok, err := verifyPhoneVerificationCode("13900001111", code, "user-1")
	if err != nil || !ok {
		t.Fatalf("verify: ok=%v err=%v", ok, err)
	}
	ok2, err := verifyPhoneVerificationCode("13900001111", code, "")
	if err != nil {
		t.Fatalf("re-verify err: %v", err)
	}
	if ok2 {
		t.Fatal("expected used code invalid")
	}
}

func TestHandleSendVerificationCodeHTTP(t *testing.T) {
	setupAuthTestDB(t)
	prev := os.Getenv("SMS_PROVIDER")
	_ = os.Setenv("SMS_PROVIDER", "mock")
	t.Cleanup(func() { _ = os.Setenv("SMS_PROVIDER", prev) })

	body := `{"phone":"13700002222"}`
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/send_verification_code/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleSendVerificationCode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["message"] == nil {
		t.Fatalf("payload %#v", payload)
	}
}

func TestHandleInternalVerifyVerificationCode(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"
	_ = os.Setenv("SMS_PROVIDER", "mock")

	_, err := sendPhoneVerificationCode(httptest.NewRequest(http.MethodPost, "/", nil).Context(), "13600003333", smsKindVerification)
	if err != nil {
		t.Fatal(err)
	}
	var code string
	_ = db.QueryRow(`SELECT code FROM auth_sms_verification_code WHERE phone = ? ORDER BY id DESC LIMIT 1`, "13600003333").Scan(&code)

	req := httptest.NewRequest(http.MethodPost, "/api/internal/verification-code/verify/", strings.NewReader(
		`{"phone":"13600003333","code":"`+code+`"}`,
	))
	req.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	rec := httptest.NewRecorder()
	handleInternalVerifyVerificationCode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	var payload map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	if payload["valid"] != true {
		t.Fatalf("%#v", payload)
	}
}

func TestSendAndVerifyEmailVerificationCode(t *testing.T) {
	setupAuthTestDB(t)

	prev := os.Getenv("SMS_PROVIDER")
	_ = os.Setenv("SMS_PROVIDER", "mock")
	t.Cleanup(func() { _ = os.Setenv("SMS_PROVIDER", prev) })

	email := "otp-email@example.com"
	_, err := sendEmailVerificationCode(httptest.NewRequest(http.MethodPost, "/", nil).Context(), email, smsKindVerification)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	var code string
	if err := db.QueryRow(`SELECT code FROM auth_sms_verification_code WHERE email = ? ORDER BY id DESC LIMIT 1`, email).Scan(&code); err != nil {
		t.Fatalf("read code: %v", err)
	}
	ok, err := verifyEmailVerificationCode(email, code, "")
	if err != nil || !ok {
		t.Fatalf("verify: ok=%v err=%v", ok, err)
	}
}

func TestSendEmailVerificationCodeReportsDeliveryFailureWhenNotMock(t *testing.T) {
	setupAuthTestDB(t)
	prev := os.Getenv("SMS_PROVIDER")
	_ = os.Setenv("SMS_PROVIDER", "aliyun")
	t.Cleanup(func() { _ = os.Setenv("SMS_PROVIDER", prev) })
	prevKafka := cfg.KafkaBootstrapServers
	cfg.KafkaBootstrapServers = ""
	t.Cleanup(func() { cfg.KafkaBootstrapServers = prevKafka })
	t.Setenv("KAFKA_BOOTSTRAP_SERVERS", "")
	t.Setenv("TASKAUTH_KAFKA_BOOTSTRAP_SERVERS", "")
	t.Setenv("EMAIL_HOST", "127.0.0.1")
	t.Setenv("EMAIL_PORT", "1")
	t.Setenv("EMAIL_HOST_USER", "user@example.com")
	t.Setenv("EMAIL_HOST_PASSWORD", "x")
	t.Setenv("EMAIL_DEFAULT_FROM", "user@example.com")
	t.Setenv("EMAIL_USE_SSL", "false")

	_, err := sendEmailVerificationCode(
		httptest.NewRequest(http.MethodPost, "/", nil).Context(),
		"otp-fail@example.com",
		smsKindVerification,
	)
	if err == nil {
		t.Fatal("expected SMTP delivery error to surface")
	}
	if !strings.Contains(err.Error(), "邮件发送失败") {
		t.Fatalf("err=%v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/send_verification_code/", strings.NewReader(`{"email":"otp-http@example.com"}`))
	rec := httptest.NewRecorder()
	handleSendVerificationCode(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestRechargeSMSGateRoundTrip(t *testing.T) {
	setupAuthTestDB(t)

	uid := "900001"
	if err := setRechargeSMSVerified(uid); err != nil {
		t.Fatalf("set: %v", err)
	}
	ok, err := isRechargeSMSVerified(uid)
	if err != nil || !ok {
		t.Fatalf("verified: ok=%v err=%v", ok, err)
	}
	if err := setRechargePendingPhone(uid, "13900007777"); err != nil {
		t.Fatalf("pending: %v", err)
	}
	phone, err := getRechargePendingPhone(uid)
	if err != nil || phone != "13900007777" {
		t.Fatalf("get pending: phone=%q err=%v", phone, err)
	}
	_ = clearRechargeSMSGate(uid)
	ok, _ = isRechargeSMSVerified(uid)
	if ok {
		t.Fatal("expected cleared")
	}
}
