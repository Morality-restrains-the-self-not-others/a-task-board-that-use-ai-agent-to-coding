package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAliyunSMSIncompleteConfigFails(t *testing.T) {
	prev := os.Getenv("SMS_PROVIDER")
	_ = os.Setenv("SMS_PROVIDER", "aliyun")
	t.Cleanup(func() { _ = os.Setenv("SMS_PROVIDER", prev) })
	for _, k := range []string{
		"SMS_ALIYUN_ACCESS_KEY_ID", "SMS_ALIYUN_ACCESS_KEY_SECRET",
		"SMS_ALIYUN_SIGN_NAME", "SMS_TEMPLATE_CODE_VERIFICATION",
	} {
		v := os.Getenv(k)
		_ = os.Unsetenv(k)
		t.Cleanup(func() { _ = os.Setenv(k, v) })
	}
	// 清除 YAML 回退，确保本测只验证「缺环境变量且无 django sms」时失败
	prevCfg := cfg
	cfg.SMSAliyunAccessKeyID = ""
	cfg.SMSAliyunAccessKeySecret = ""
	cfg.SMSAliyunSignName = ""
	cfg.SMSTemplateCodeVerification = ""
	cfg.SMSAliyunTemplateCodeNotification = ""
	cfg.SMSTemplateCodeNotification = ""
	t.Cleanup(func() { cfg = prevCfg })
	res := sendVerificationSMS(httptest.NewRequest(http.MethodPost, "/", nil).Context(), "+8613900005555", "123456", smsKindVerification)
	if res.Success {
		t.Fatal("expected aliyun incomplete config to fail")
	}
}

func TestPasswordResetPhoneCodeLocal(t *testing.T) {
	setupAuthTestDB(t)
	prev := os.Getenv("SMS_PROVIDER")
	_ = os.Setenv("SMS_PROVIDER", "mock")
	t.Cleanup(func() { _ = os.Setenv("SMS_PROVIDER", prev) })

	phone := "13800006666"
	userID, err := createUserWithPhoneLogin("+86", phone, "oldhash", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = userID

	sendBody, _ := json.Marshal(map[string]string{"phone": phone})
	sendReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/send_password_reset_code/", strings.NewReader(string(sendBody)))
	sendReq.Header.Set("Content-Type", "application/json")
	sendRec := httptest.NewRecorder()
	handleSendPasswordResetCode(sendRec, sendReq)
	if sendRec.Code != http.StatusOK {
		t.Fatalf("send expected 200 got %d %s", sendRec.Code, sendRec.Body.String())
	}
	var code string
	if err := db.QueryRow(`SELECT code FROM auth_sms_verification_code WHERE phone = ? ORDER BY id DESC LIMIT 1`, phone).Scan(&code); err != nil {
		t.Fatalf("code: %v", err)
	}
	resetBody, _ := json.Marshal(map[string]string{"phone": phone, "code": code, "new_password": "newhash"})
	resetReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/reset_password_with_code/", strings.NewReader(string(resetBody)))
	resetReq.Header.Set("Content-Type", "application/json")
	resetRec := httptest.NewRecorder()
	handleResetPasswordWithCode(resetRec, resetReq)
	if resetRec.Code != http.StatusOK {
		t.Fatalf("reset expected 200 got %d %s", resetRec.Code, resetRec.Body.String())
	}
	lm, err := findLoginMethodByPhone("+86", phone)
	if err != nil || lm == nil {
		t.Fatalf("login method missing: lm=%+v err=%v", lm, err)
	}
	if !checkPasswordHash("newhash", lm.PasswordHash) {
		t.Fatalf("password not updated: hash does not match new password, lm=%+v", lm)
	}
}

// handleLogout 双变体 cookie 清除（OPT-20260808）：登出必须同时清 userId/token 的
// 域变体（.daydaymoney.com）+ host-only 变体（071 前旧登录影子）。修复前仅删服务端
// token，30 天 HttpOnly cookie 残留各子域且 JS 无法删除 → reload 后反复 401 跳登录。
func postPhonePasswordLogin(t *testing.T, phone, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"phone": phone, "password": password})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-Id", "login-phone-password-regression")
	rec := httptest.NewRecorder()
	handleLogin(rec, req)
	return rec
}

// 注册把 phone login identifier 存成国内号（189…），前端登录提交 E.164（+86189…）。
// 若登录仍按 identifier 精确匹配，会在 0ms 内 400「手机号或密码错误」（bcrypt 不会跑）。
func TestHandleLoginPhonePasswordAcceptsE164AfterRegister(t *testing.T) {
	setupAuthTestDB(t)
	const national = "13900001111"
	const password = "TestPassw0rd!"
	userID, err := createUserWithPhoneLogin("+86", national, password, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	stored, err := findLoginMethodByIdentifier("+86" + national)
	if err != nil {
		t.Fatalf("identifier lookup: %v", err)
	}
	if stored != nil {
		t.Fatalf("phone identifier must be national, not E.164; got %+v", stored)
	}

	rec := postPhonePasswordLogin(t, "+86"+national, password)
	if rec.Code != http.StatusOK {
		t.Fatalf("E.164 phone+password login expected 200, got %d %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Fatalf("missing token: %v", resp)
	}
	user, _ := resp["user"].(map[string]interface{})
	if user == nil || fmt.Sprint(user["id"]) != userID {
		t.Fatalf("user id mismatch got=%v want=%s", user, userID)
	}

	var lastLogin sql.NullString
	if err := db.QueryRow(`SELECT last_login FROM auth_user WHERE id = ?`, userID).Scan(&lastLogin); err != nil {
		t.Fatalf("last_login: %v", err)
	}
	if !lastLogin.Valid || lastLogin.String == "" {
		t.Fatal("password login must stamp last_login")
	}

	nationalRec := postPhonePasswordLogin(t, national, password)
	if nationalRec.Code != http.StatusOK {
		t.Fatalf("national phone+password login expected 200, got %d %s", nationalRec.Code, nationalRec.Body.String())
	}

	wrong := postPhonePasswordLogin(t, "+86"+national, "WrongPassw0rd!")
	if wrong.Code != http.StatusBadRequest {
		t.Fatalf("wrong password expected 400, got %d %s", wrong.Code, wrong.Body.String())
	}
}

// OPT-20260819-034 回归：findLoginMethodByCanonicalPhone 与 findLoginMethodForPasswordReset
// 对同一规范化手机号应返回同一 login_method（登录/重置共用查找逻辑）。
func TestFindLoginMethodByCanonicalPhoneSharedLookup(t *testing.T) {
	setupAuthTestDB(t)
	const national = "13900002222"
	const password = "TestPassw0rd!"
	userID, err := createUserWithPhoneLogin("+86", national, password, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	e164 := "+86" + national
	canonical := canonicalPhoneForSMSAndLogin(e164)
	lm1, err := findLoginMethodByCanonicalPhone(canonical)
	if err != nil {
		t.Fatalf("canonical lookup: %v", err)
	}
	if lm1 == nil || lm1.ObjectID != userID {
		t.Fatalf("canonical lookup got=%+v want user=%s", lm1, userID)
	}
	lm2, err := findLoginMethodForPasswordReset(e164, "")
	if err != nil {
		t.Fatalf("reset lookup: %v", err)
	}
	if lm2 == nil || lm2.ID != lm1.ID {
		t.Fatalf("reset lookup got=%+v want same as canonical=%+v", lm2, lm1)
	}
}

func TestHandleLogoutClearsBothCookieVariants(t *testing.T) {
	setupForwardAuthTest(t)
	oldBase := cfg.GatewayPublicBase
	cfg.GatewayPublicBase = "https://www.daydaymoney.com"
	defer func() { cfg.GatewayPublicBase = oldBase }()

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/logout/", nil)
	req.Header.Set("Authorization", "Token some-invalid-token-key")
	rec := httptest.NewRecorder()
	handleLogout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}
	sc := rec.Header().Values("Set-Cookie")
	if len(sc) != 4 {
		t.Fatalf("expected 4 Set-Cookie (userId/token × 域+host-only), got %d: %v", len(sc), sc)
	}
	for _, name := range []string{"userId", "token"} {
		var domainVariant, hostOnlyVariant bool
		for _, c := range sc {
			if !strings.Contains(c, name+"=") || !strings.Contains(c, "Max-Age=0") {
				continue
			}
			if strings.Contains(c, "Domain=") {
				domainVariant = true
			} else {
				hostOnlyVariant = true
			}
		}
		if !domainVariant || !hostOnlyVariant {
			t.Fatalf("expected BOTH domain+host-only clears for %s, got %v", name, sc)
		}
	}
}

// 2026-08-24：手机号+验证码登录已移除（fail-closed）。
// - {phone, code} 无 password → 400 phone_code_login_disabled，不发 token、不自动注册
// - {phone, code, password} 混合请求 → 走密码登录路径（password 存在时以密码为准）
func TestHandleLoginRejectsPhoneCodeLogin(t *testing.T) {
	setupAuthTestDB(t)

	t.Run("phone_code_without_password_rejected", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"phone": "+8613900004444", "code": "123456"})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json: %v body=%s", err, rec.Body.String())
		}
		if resp["error"] != "phone_code_login_disabled" {
			t.Fatalf("expected error=phone_code_login_disabled, got %v", resp)
		}
		if resp["token"] != nil {
			t.Fatalf("disabled login must not issue token: %v", resp)
		}
	})

	t.Run("phone_code_does_not_auto_register", func(t *testing.T) {
		// 未注册手机号 + code：必须拒绝且不得创建账号
		phone := "+8613900005555"
		body, _ := json.Marshal(map[string]string{"phone": phone, "code": "123456"})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
		}
		lm, err := findLoginMethodByPhone("+86", "13900005555")
		if err != nil {
			t.Fatalf("lookup: %v", err)
		}
		if lm != nil {
			t.Fatalf("phone+code login must not auto-register, got login method: %+v", lm)
		}
	})

	t.Run("phone_code_with_password_uses_password_path", func(t *testing.T) {
		// 混合请求：password 存在时按密码登录处理（语义：以密码为准）
		const national = "13900006666"
		if _, err := createUserWithPhoneLogin("+86", national, "hash-of-wrong-pass", ""); err != nil {
			t.Fatalf("create phone user: %v", err)
		}
		body, _ := json.Marshal(map[string]string{
			"phone": "+86" + national, "code": "123456", "password": "wrong-password",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleLogin(rec, req)

		// 密码错误 → 统一「手机号或密码错误」，而非验证码错误
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "手机号或密码错误") {
			t.Fatalf("expected password mismatch message, got %s", rec.Body.String())
		}
	})

	t.Run("rejection_is_audit_logged", func(t *testing.T) {
		// regression：fail-closed 拒绝须留痕（reason=phone_code_login_disabled），
		// 与 not_found/password_mismatch 拒绝路径一致，直连绕过可审计
		var buf bytes.Buffer
		prev := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
		defer slog.SetDefault(prev)

		body, _ := json.Marshal(map[string]string{"phone": "+8613900007777", "code": "123456"})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(buf.String(), "phone_code_login_disabled") {
			t.Fatalf("expected audit log reason=phone_code_login_disabled, got: %q", buf.String())
		}
	})
}
