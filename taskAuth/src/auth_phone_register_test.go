package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPhoneRegisterMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/users/phone_register/", nil)
	rec := httptest.NewRecorder()
	handlePhoneRegister(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func postPhoneRegister(t *testing.T, body map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/phone_register/", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handlePhoneRegister(rec, req)
	return rec
}

func withReferralRecorder(t *testing.T) *referralRecorder {
	t.Helper()
	rec := &referralRecorder{reqs: make(chan map[string]string, 4)}
	srv := httptest.NewServer(rec.handler())
	t.Cleanup(srv.Close)
	prev := cfg.ReferralServiceURL
	cfg.ReferralServiceURL = srv.URL
	t.Cleanup(func() { cfg.ReferralServiceURL = prev })
	return rec
}

func decodeJSONMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	return body
}

// TestPhoneRegisterExistingUserRejectedAndDoesNotBind 已有手机号不得当「注册成功」，
// 即使带 access_code 也不得绑边、不得改密。
func TestPhoneRegisterExistingUserRejectedAndDoesNotBind(t *testing.T) {
	setupAuthTestDB(t)
	rec := withReferralRecorder(t)

	const national = "13800008888"
	const oldPass = "OldPassw0rd!"
	if _, err := createUserWithPhoneLogin("+86", national, oldPass, ""); err != nil {
		t.Fatalf("create existing user: %v", err)
	}
	seedPhoneVerificationCode(t, national, "+86", "654321", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":       "+86" + national,
		"password":    "NewPassw0rd!",
		"code":        "654321",
		"access_code": "DR2AKvP9J9",
	})
	if httpRec.Code != http.StatusBadRequest {
		t.Fatalf("existing phone register status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
	body := decodeJSONMap(t, httpRec)
	if fmt.Sprint(body["error"]) != "该手机号已被注册，请直接登录" {
		t.Fatalf("error=%v", body["error"])
	}
	if body["user_existed"] != true {
		t.Fatalf("user_existed=%v want true", body["user_existed"])
	}
	select {
	case got := <-rec.reqs:
		t.Fatalf("existing user must not bind referral, got %v", got)
	case <-time.After(300 * time.Millisecond):
	}
	lm, err := findLoginMethodByPhone("+86", national)
	if err != nil || lm == nil {
		t.Fatalf("login method: lm=%+v err=%v", lm, err)
	}
	if !checkPasswordHash(oldPass, lm.PasswordHash) {
		t.Fatal("existing user password must not be rewritten")
	}
}

func TestPhoneRegisterExistingUserWithoutAccessCodeStillRejected(t *testing.T) {
	setupAuthTestDB(t)
	rec := withReferralRecorder(t)

	const national = "13800008889"
	if _, err := createUserWithPhoneLogin("+86", national, "OldPassw0rd!", ""); err != nil {
		t.Fatalf("create existing user: %v", err)
	}
	seedPhoneVerificationCode(t, national, "+86", "111111", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":    "+86" + national,
		"password": "NewPassw0rd!",
		"code":     "111111",
	})
	if httpRec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
	select {
	case got := <-rec.reqs:
		t.Fatalf("must not bind, got %v", got)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestPhoneRegisterNewUserBindsReferral(t *testing.T) {
	setupAuthTestDB(t)
	rec := withReferralRecorder(t)

	const national = "13800008890"
	seedPhoneVerificationCode(t, national, "+86", "123456", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":       "+86" + national,
		"password":    "NewPassw0rd!",
		"code":        "123456",
		"access_code": "DR2AKvP9J9",
	})
	if httpRec.Code != http.StatusCreated {
		t.Fatalf("new phone register status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
	referral := waitForReferralCall(t, rec, "DR2AKvP9J9")
	if referral["access_code"] != "DR2AKvP9J9" || referral["referred_user_id"] == "" {
		t.Fatalf("referral bind body unexpected: %v", referral)
	}
}

func TestPhoneRegisterSucceedsWhenOccupantArchived(t *testing.T) {
	setupAuthTestDB(t)
	const national = "13900001111"
	oldID, err := createUserWithPhoneLogin("+86", national, "OldPassw0rd!", "")
	if err != nil {
		t.Fatalf("create occupant: %v", err)
	}
	archiveUser(t, oldID)
	seedPhoneVerificationCode(t, national, "+86", "222222", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":    "+86" + national,
		"password": "NewPassw0rd!",
		"code":     "222222",
	})
	if httpRec.Code != http.StatusCreated {
		t.Fatalf("archived occupant should allow register status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
	var voided sql.NullTime
	if err := db.QueryRow(
		`SELECT binding_voided_at FROM auth_login_method WHERE object_id=? AND method_type='phone'`,
		oldID,
	).Scan(&voided); err != nil {
		t.Fatalf("load old method: %v", err)
	}
	if !voided.Valid {
		t.Fatal("expected stale phone login method to be voided")
	}
	lm, err := findLoginMethodByPhone("+86", national)
	if err != nil || lm == nil {
		t.Fatalf("expected live occupant after register lm=%+v err=%v", lm, err)
	}
	if lm.ObjectID == oldID {
		t.Fatal("live phone occupancy still points at archived user")
	}
}

func TestPhoneRegisterSucceedsWhenOccupantUserMissing(t *testing.T) {
	setupAuthTestDB(t)
	const national = "18965128762"
	oldID, err := createUserWithPhoneLogin("+86", national, "OldPassw0rd!", "")
	if err != nil {
		t.Fatalf("create occupant: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM auth_user WHERE id = ?`, oldID); err != nil {
		t.Fatalf("delete occupant user: %v", err)
	}
	seedPhoneVerificationCode(t, national, "+86", "333333", "")

	httpRec := postPhoneRegister(t, map[string]string{
		"phone":    "+86" + national,
		"password": "NewPassw0rd!",
		"code":     "333333",
	})
	if httpRec.Code != http.StatusCreated {
		t.Fatalf("orphan login method should allow register status=%d body=%s", httpRec.Code, httpRec.Body.String())
	}
	lm, err := findLoginMethodByPhone("+86", national)
	if err != nil || lm == nil || lm.ObjectID == oldID {
		t.Fatalf("expected new live occupant lm=%+v err=%v", lm, err)
	}
}
