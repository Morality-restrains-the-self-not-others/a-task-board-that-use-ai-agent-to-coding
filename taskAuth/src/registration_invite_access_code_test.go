package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// inviteGateRecorder mock taskReferral /api/internal/referral/share-code/validate/，
// 响应体与状态码可配置，记录收到的请求体。
type inviteGateRecorder struct {
	valid  bool
	status int
	reqs   chan map[string]string
}

func (r *inviteGateRecorder) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/internal/referral/share-code/validate/" {
			http.NotFound(w, req)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(req.Body).Decode(&body)
		select {
		case r.reqs <- body:
		default:
		}
		w.WriteHeader(r.status)
		if r.status < 400 {
			_, _ = w.Write([]byte(fmt.Sprintf(`{"valid":%t}`, r.valid)))
		}
	})
}

func withInviteGateRecorder(t *testing.T, valid bool, status int) *inviteGateRecorder {
	t.Helper()
	rec := &inviteGateRecorder{valid: valid, status: status, reqs: make(chan map[string]string, 4)}
	srv := httptest.NewServer(rec.handler())
	t.Cleanup(srv.Close)
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })
	return rec
}

func waitForInviteGateCall(t *testing.T, rec *inviteGateRecorder, wantCode string) map[string]string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case body := <-rec.reqs:
			return body
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Fatalf("share-code/validate not called within timeout (want code %s)", wantCode)
	return nil
}

// TestPhoneRegisterAccessCodeExemptsInviteCode 邀请策略开启时，
// 携带有效 access_code 注册不再要求 invite_code（分享链接即邀请凭证）。
func TestPhoneRegisterAccessCodeExemptsInviteCode(t *testing.T) {
	setupAuthTestDB(t)
	withInviteGateRecorder(t, true, http.StatusOK)
	setRegistrationInvitePolicyForTest(t, true, 5)

	const national = "13800009901"
	seedPhoneVerificationCode(t, national, "+86", "654321", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":       "+86" + national,
		"password":    "NewPassw0rd!",
		"code":        "654321",
		"access_code": "w25brikFiT",
	})
	if httpRec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
}

// TestPhoneRegisterAccessCodeInvalidRejected 无效/伪造 access_code 不得豁免，
// 注册被拒（fail-closed）。
func TestPhoneRegisterAccessCodeInvalidRejected(t *testing.T) {
	setupAuthTestDB(t)
	withInviteGateRecorder(t, false, http.StatusOK)
	setRegistrationInvitePolicyForTest(t, true, 5)

	const national = "13800009902"
	seedPhoneVerificationCode(t, national, "+86", "654321", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":       "+86" + national,
		"password":    "NewPassw0rd!",
		"code":        "654321",
		"access_code": "FAKE000000",
	})
	if httpRec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
	body := decodeJSONMap(t, httpRec)
	if body["error"] != "access_code_invalid" {
		t.Fatalf("error=%v want access_code_invalid", body["error"])
	}
	// 用户不应被创建
	lm, err := findLoginMethodByPhone("+86", national)
	if err != nil {
		t.Fatalf("findLoginMethodByPhone: %v", err)
	}
	if lm != nil {
		t.Fatal("invalid access code must not create user")
	}
}

// TestPhoneRegisterAccessCodeGateUnavailableFailClosed 校验服务不可达时
// 注册被拒（fail-closed），不允许绕过邀请闸门。
func TestPhoneRegisterAccessCodeGateUnavailableFailClosed(t *testing.T) {
	setupAuthTestDB(t)
	withInviteGateRecorder(t, true, http.StatusInternalServerError)
	setRegistrationInvitePolicyForTest(t, true, 5)

	const national = "13800009903"
	seedPhoneVerificationCode(t, national, "+86", "654321", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":       "+86" + national,
		"password":    "NewPassw0rd!",
		"code":        "654321",
		"access_code": "w25brikFiT",
	})
	if httpRec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
	body := decodeJSONMap(t, httpRec)
	if body["error"] != "invite_gate_unavailable" {
		t.Fatalf("error=%v want invite_gate_unavailable", body["error"])
	}
}

// TestPhoneRegisterAccessCodeStillRequiresCodeWhenPolicyOn 回归：策略开启、
// 无 access_code 时仍要求 invite_code。
func TestPhoneRegisterAccessCodeStillRequiresCodeWhenPolicyOn(t *testing.T) {
	setupAuthTestDB(t)
	setRegistrationInvitePolicyForTest(t, true, 5)

	const national = "13800009904"
	seedPhoneVerificationCode(t, national, "+86", "654321", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":    "+86" + national,
		"password": "NewPassw0rd!",
		"code":     "654321",
	})
	if httpRec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
	body := decodeJSONMap(t, httpRec)
	if body["error"] != "invite_code_required" {
		t.Fatalf("error=%v want invite_code_required", body["error"])
	}
}
