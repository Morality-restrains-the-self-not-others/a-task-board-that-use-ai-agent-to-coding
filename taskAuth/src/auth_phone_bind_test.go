package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskAuth/domain"
)

// seedPhoneVerificationCode 插入一条有效验证码（is_used=0, expires_at 未来）。
func seedPhoneVerificationCode(t *testing.T, phone, cc, code, userID string) {
	t.Helper()
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	future := time.Now().UTC().Add(5 * time.Minute).Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(`
		INSERT INTO auth_sms_verification_code (phone, country_calling_code, user_id, code, created_at, expires_at, is_used)
		VALUES (?, ?, ?, ?, ?, ?, 0)`, phone, cc, userID, code, now, future)
	if err != nil {
		t.Fatalf("seed sms code: %v", err)
	}
}

func TestBindPhone_RequiresLogin(t *testing.T) {
	setupAuthTestDB(t)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138000","code":"123456"}`))
	rec := httptest.NewRecorder()
	handleBindPhone(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without login, got %d", rec.Code)
	}
}

func TestBindPhone_InvalidCode(t *testing.T) {
	setupAuthTestDB(t)
	userID := mustCreateWechatTestUser(t, "u-bind-test-01")
	seedPhoneVerificationCode(t, "13800138000", "+86", "654321", userID)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138000","code":"000000"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userID))
	rec := httptest.NewRecorder()
	handleBindPhone(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid code, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestBindPhone_SuccessAndTakenConflict(t *testing.T) {
	setupAuthTestDB(t)
	userA := mustCreateWechatTestUser(t, "u-bind-test-02")
	userB := mustCreateWechatTestUser(t, "u-bind-test-03")

	// A 绑定手机号（验证码正确）→ 成功
	seedPhoneVerificationCode(t, "13800138001", "+86", "111111", userA)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138001","code":"111111"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userA))
	rec := httptest.NewRecorder()
	handleBindPhone(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// B 绑定同一手机号（验证码正确）→ 409 已被占用
	seedPhoneVerificationCode(t, "13800138001", "+86", "222222", userB)
	req2 := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138001","code":"222222"}`))
	req2.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userB))
	rec2 := httptest.NewRecorder()
	handleBindPhone(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409 taken, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	var takenBody map[string]interface{}
	if err := json.Unmarshal(rec2.Body.Bytes(), &takenBody); err != nil {
		t.Fatalf("decode 409: %v", err)
	}
	if takenBody["code"] != "phone_taken" {
		t.Fatalf("expected code phone_taken, got %v body=%s", takenBody["code"], rec2.Body.String())
	}
	if takenBody["reclaim_available"] != true {
		t.Fatalf("expected reclaim_available true, got %v", takenBody["reclaim_available"])
	}
}

func TestBindPhone_TakenKeepsCodeForReclaim(t *testing.T) {
	setupAuthTestDB(t)
	owner, err := createUserWithPhoneLogin("+86", "13800138009", "hash-owner", "")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	claimer := mustCreateWechatTestUser(t, "u-bind-reclaim-01")
	seedPhoneVerificationCode(t, "13800138009", "+86", "333333", claimer)

	conflictReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138009","code":"333333"}`))
	conflictReq.Header.Set("Authorization", "Token "+mustIssueTestToken(t, claimer))
	conflictRec := httptest.NewRecorder()
	handleBindPhone(conflictRec, conflictReq)
	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("expected 409 without reclaim, got %d body=%s", conflictRec.Code, conflictRec.Body.String())
	}

	reclaimReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138009","code":"333333","reclaim":true}`))
	reclaimReq.Header.Set("Authorization", "Token "+mustIssueTestToken(t, claimer))
	reclaimRec := httptest.NewRecorder()
	handleBindPhone(reclaimRec, reclaimReq)
	if reclaimRec.Code != http.StatusOK {
		t.Fatalf("expected 200 reclaim, got %d body=%s", reclaimRec.Code, reclaimRec.Body.String())
	}

	claimerLM, err := findLoginMethodByPhone("+86", "13800138009")
	if err != nil || claimerLM == nil || claimerLM.ObjectID != claimer {
		t.Fatalf("expected phone on claimer, lm=%v err=%v", claimerLM, err)
	}
	var ownerVoided sql.NullString
	err = db.QueryRow(`
		SELECT binding_voided_at FROM auth_login_method
		WHERE object_id=? AND method_type='phone' ORDER BY id DESC LIMIT 1`, owner).Scan(&ownerVoided)
	if err != nil {
		t.Fatalf("owner phone row: %v", err)
	}
	if !ownerVoided.Valid || ownerVoided.String == "" {
		t.Fatalf("expected owner phone voided, got %v", ownerVoided)
	}
	_ = owner
}

func mustIssueTestToken(t *testing.T, userID string) string {
	t.Helper()
	token, err := getOrCreateToken(userID, cfg.UserContentTypeID, "")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return token
}

func mustMarkPhoneShareRole(t *testing.T, userID string, staff, super, tester bool) {
	t.Helper()
	toInt := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}
	_, err := db.Exec(`UPDATE auth_user SET is_staff = ?, is_superuser = ?, is_tester = ? WHERE id = ?`,
		toInt(staff), toInt(super), toInt(tester), userID)
	if err != nil {
		t.Fatalf("mark phone-share role: %v", err)
	}
}

func TestBindPhone_TwoStaffShareSameNumber(t *testing.T) {
	setupAuthTestDB(t)
	userA := mustCreateWechatTestUser(t, "u-share-staff-a")
	userB := mustCreateWechatTestUser(t, "u-share-staff-b")
	mustMarkPhoneShareRole(t, userA, true, false, false)
	mustMarkPhoneShareRole(t, userB, true, false, false)

	seedPhoneVerificationCode(t, "13800138101", "+86", "111111", userA)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138101","code":"111111"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userA))
	rec := httptest.NewRecorder()
	handleBindPhone(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("staff A bind expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	seedPhoneVerificationCode(t, "13800138101", "+86", "222222", userB)
	req2 := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138101","code":"222222"}`))
	req2.Header.Set("Authorization", "Token "+mustIssueTestToken(t, userB))
	rec2 := httptest.NewRecorder()
	handleBindPhone(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("staff B share bind expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}

	methods, err := listLiveLoginMethodsByPhone("+86", "13800138101")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(methods) != 2 {
		t.Fatalf("expected 2 live bindings, got %d", len(methods))
	}
}

func TestBindPhone_SixthPrivilegedRejectedWithLimit(t *testing.T) {
	setupAuthTestDB(t)
	const national = "13800138102"
	for i := 0; i < 5; i++ {
		u := mustCreateWechatTestUser(t, fmt.Sprintf("u-share-limit-%d", i))
		mustMarkPhoneShareRole(t, u, true, false, false)
		if err := upsertPhoneLoginMethod(u, "+86", national); err != nil {
			t.Fatalf("seed bind %d: %v", i, err)
		}
	}
	sixth := mustCreateWechatTestUser(t, "u-share-limit-6")
	mustMarkPhoneShareRole(t, sixth, true, false, false)
	seedPhoneVerificationCode(t, national, "+86", "666666", sixth)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+86`+national+`","code":"666666"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, sixth))
	rec := httptest.NewRecorder()
	handleBindPhone(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 limit, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["code"] != "phone_bind_limit" {
		t.Fatalf("expected phone_bind_limit, got %v body=%s", body["code"], rec.Body.String())
	}
	if body["reclaim_available"] != false {
		t.Fatalf("limit must not advertise reclaim, got %v", body["reclaim_available"])
	}
}

func TestBindPhone_CustomerCannotShareStaffNumber(t *testing.T) {
	setupAuthTestDB(t)
	staff := mustCreateWechatTestUser(t, "u-share-staff-hold")
	mustMarkPhoneShareRole(t, staff, true, false, false)
	if err := upsertPhoneLoginMethod(staff, "+86", "13800138103"); err != nil {
		t.Fatalf("staff bind: %v", err)
	}
	customer := mustCreateWechatTestUser(t, "u-share-cust")
	seedPhoneVerificationCode(t, "13800138103", "+86", "777777", customer)
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138103","code":"777777"}`))
	req.Header.Set("Authorization", "Token "+mustIssueTestToken(t, customer))
	rec := httptest.NewRecorder()
	handleBindPhone(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("customer expected 409 taken, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["code"] != "phone_taken" {
		t.Fatalf("expected phone_taken, got %v", body["code"])
	}
}

func TestBindPhone_StaffVersusCustomerRequiresReclaim(t *testing.T) {
	setupAuthTestDB(t)
	customer := mustCreateWechatTestUser(t, "u-share-cust-hold")
	if err := upsertPhoneLoginMethod(customer, "+86", "13800138104"); err != nil {
		t.Fatalf("customer bind: %v", err)
	}
	staff := mustCreateWechatTestUser(t, "u-share-staff-reclaim")
	mustMarkPhoneShareRole(t, staff, true, false, false)
	seedPhoneVerificationCode(t, "13800138104", "+86", "888888", staff)

	conflict := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138104","code":"888888"}`))
	conflict.Header.Set("Authorization", "Token "+mustIssueTestToken(t, staff))
	conflictRec := httptest.NewRecorder()
	handleBindPhone(conflictRec, conflict)
	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("staff vs customer expected 409, got %d %s", conflictRec.Code, conflictRec.Body.String())
	}

	reclaim := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/",
		bytes.NewBufferString(`{"phone":"+8613800138104","code":"888888","reclaim":true}`))
	reclaim.Header.Set("Authorization", "Token "+mustIssueTestToken(t, staff))
	reclaimRec := httptest.NewRecorder()
	handleBindPhone(reclaimRec, reclaim)
	if reclaimRec.Code != http.StatusOK {
		t.Fatalf("reclaim expected 200, got %d %s", reclaimRec.Code, reclaimRec.Body.String())
	}
	methods, err := listLiveLoginMethodsByPhone("+86", "13800138104")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(methods) != 1 || methods[0].ObjectID != staff {
		t.Fatalf("expected sole staff binding, got %+v", methods)
	}
}

func TestBindPhone_TesterAndSuperuserCanShare(t *testing.T) {
	setupAuthTestDB(t)
	tester := mustCreateWechatTestUser(t, "u-share-tester")
	super := mustCreateWechatTestUser(t, "u-share-super")
	mustMarkPhoneShareRole(t, tester, false, false, true)
	mustMarkPhoneShareRole(t, super, false, true, false)
	if err := upsertPhoneLoginMethod(tester, "+86", "13800138105"); err != nil {
		t.Fatalf("tester: %v", err)
	}
	if err := upsertPhoneLoginMethod(super, "+86", "13800138105"); err != nil {
		t.Fatalf("super share: %v", err)
	}
	methods, err := listLiveLoginMethodsByPhone("+86", "13800138105")
	if err != nil || len(methods) != 2 {
		t.Fatalf("expected 2 bindings, got %d err=%v", len(methods), err)
	}
}

func TestPhoneRegisterRejectedWhenPrivilegedAlreadyHoldsNumber(t *testing.T) {
	setupAuthTestDB(t)
	staff := mustCreateWechatTestUser(t, "u-share-reg-staff")
	mustMarkPhoneShareRole(t, staff, true, false, false)
	if err := upsertPhoneLoginMethod(staff, "+86", "13800138106"); err != nil {
		t.Fatalf("staff: %v", err)
	}
	seedPhoneVerificationCode(t, "13800138106", "+86", "121212", "")
	rec := postPhoneRegister(t, map[string]string{
		"phone":    "+8613800138106",
		"password": "NewPassw0rd!",
		"code":     "121212",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("register on shared staff phone expected 400, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestPhoneResetCodeRejectedWhenShared(t *testing.T) {
	setupAuthTestDB(t)
	a, err := createUserWithPhoneLogin("+86", "13800138107", "Passw0rdA!", "")
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	b, err := createUserWithPhoneLogin("+86", "13800138107", "Passw0rdB!", "")
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	mustMarkPhoneShareRole(t, a, true, false, false)
	mustMarkPhoneShareRole(t, b, true, false, false)

	sendBody, _ := json.Marshal(map[string]string{"phone": "+8613800138107"})
	sendReq := httptest.NewRequest(http.MethodPost, "/api/accounts/users/send_password_reset_code/", strings.NewReader(string(sendBody)))
	sendReq.Header.Set("Content-Type", "application/json")
	sendRec := httptest.NewRecorder()
	handleSendPasswordResetCode(sendRec, sendReq)
	if sendRec.Code != http.StatusBadRequest {
		t.Fatalf("shared phone reset send expected 400, got %d %s", sendRec.Code, sendRec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(sendRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["code"] != "phone_ambiguous" {
		t.Fatalf("expected phone_ambiguous, got %v", body["code"])
	}
}

func TestWritePhoneBindConflict_LimitAndTakenBodies(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/bind_phone/", nil)

	limitRec := httptest.NewRecorder()
	writePhoneBindConflict(limitRec, req, domain.PhoneShareLimit)
	if limitRec.Code != http.StatusConflict {
		t.Fatalf("limit status %d", limitRec.Code)
	}
	var limitBody map[string]interface{}
	if err := json.Unmarshal(limitRec.Body.Bytes(), &limitBody); err != nil {
		t.Fatalf("limit json: %v", err)
	}
	if limitBody["code"] != "phone_bind_limit" {
		t.Fatalf("limit code %v", limitBody["code"])
	}
	if limitBody["reclaim_available"] != false {
		t.Fatalf("limit reclaim_available %v", limitBody["reclaim_available"])
	}
	if limitBody["limit"] != float64(domain.MaxSharedPhoneBindings) {
		t.Fatalf("limit field %v", limitBody["limit"])
	}

	takenRec := httptest.NewRecorder()
	writePhoneBindConflict(takenRec, req, domain.PhoneShareTaken)
	if takenRec.Code != http.StatusConflict {
		t.Fatalf("taken status %d", takenRec.Code)
	}
	var takenBody map[string]interface{}
	if err := json.Unmarshal(takenRec.Body.Bytes(), &takenBody); err != nil {
		t.Fatalf("taken json: %v", err)
	}
	if takenBody["code"] != "phone_taken" {
		t.Fatalf("taken code %v", takenBody["code"])
	}
	if takenBody["reclaim_available"] != true {
		t.Fatalf("taken reclaim_available %v", takenBody["reclaim_available"])
	}
}

var _ = json.Valid // keep encoding/json import if unused elsewhere
