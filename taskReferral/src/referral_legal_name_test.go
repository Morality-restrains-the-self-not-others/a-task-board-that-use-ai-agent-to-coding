package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNormalizeLegalName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		raw     string
		wantErr error
	}{
		{name: "empty", raw: "  ", wantErr: errReferralLegalNameRequired},
		{name: "too short", raw: "张", wantErr: errReferralLegalNameTooShort},
		{name: "digits", raw: "张3", wantErr: errReferralLegalNameInvalid},
		{name: "ok chinese", raw: " 张三 ", wantErr: nil},
		{name: "ok middle dot", raw: "买买提·艾力", wantErr: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeLegalName(tc.raw)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err=%v want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got == "" {
				t.Fatal("empty normalized name")
			}
		})
	}
}

func TestApplyReferralCodeRequiresLegalName(t *testing.T) {
	setupTestReferralDB(t)
	setReferralPolicyForTest(t, "approval", "")

	_, err := applyReferralCode("user-name-empty", testValidPersonalIntro, "", true)
	if !errors.Is(err, errReferralLegalNameRequired) {
		t.Fatalf("empty name err=%v", err)
	}

	result, err := applyReferralCode("user-name-ok", testValidPersonalIntro, testValidLegalName, true)
	if err != nil {
		t.Fatalf("valid name apply: %v", err)
	}
	if result.Status != "pending" {
		t.Fatalf("status=%s", result.Status)
	}
	var stored string
	if err := db.QueryRow(`SELECT legal_name FROM referral_code WHERE user_id = ?`, "user-name-ok").Scan(&stored); err != nil {
		t.Fatalf("query: %v", err)
	}
	if stored != testValidLegalName {
		t.Fatalf("stored=%q", stored)
	}
	status, err := getReferralCodeStatus("user-name-ok")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.LegalName != testValidLegalName {
		t.Fatalf("status name=%q", status.LegalName)
	}
}

func TestHandleReferralCodeApplyRequiresLegalName(t *testing.T) {
	mux := setupTestMux(t)
	body, _ := json.Marshal(map[string]interface{}{
		"personal_intro":        testValidPersonalIntro,
		"identity_bind_consent": true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/referral-codes/apply/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withReferralUser(req, "user-http-name")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var parsed map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v", err)
	}
	if parsed["error"] != "invalid_legal_name" {
		t.Fatalf("error=%s", parsed["error"])
	}
}

// ── OPT-20260823-048: 存量已获资格用户补填个人名称 ──

func newLegalNameUpdateRequest(userID, name string) *http.Request {
	body, _ := json.Marshal(map[string]interface{}{"legal_name": name})
	req := httptest.NewRequest(http.MethodPost, "/api/referral/legal-name/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return withReferralUser(req, userID)
}

func TestHandleReferralLegalNameUpdateUnauthenticated(t *testing.T) {
	mux := setupTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/referral/legal-name/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleReferralLegalNameUpdateNoActiveQualification(t *testing.T) {
	mux := setupTestMux(t)
	req := newLegalNameUpdateRequest("user-no-active", "张三")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var parsed map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v", err)
	}
	if parsed["error"] != "no_active_qualification" {
		t.Fatalf("error=%s", parsed["error"])
	}
}

func TestHandleReferralLegalNameUpdateInvalidName(t *testing.T) {
	mux := setupTestMux(t)
	// 先开通 active 资格，再传非法名称。
	insertApprovedReferral(t, "user-invalid-name", time.Now().UTC().Add(30*24*time.Hour))
	req := newLegalNameUpdateRequest("user-invalid-name", "张3")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var parsed map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v", err)
	}
	if parsed["error"] != "invalid_legal_name" {
		t.Fatalf("error=%s", parsed["error"])
	}
}

func TestHandleReferralLegalNameUpdateSuccessEnsuresReceiver(t *testing.T) {
	mux := setupTestMux(t)
	insertApprovedReferral(t, "user-legal-backfill", time.Now().UTC().Add(30*24*time.Hour))

	var ensured []string
	orig := ensureWechatProfitSharingReceiver
	ensureWechatProfitSharingReceiver = func(uid string) wechatReceiverStatus {
		ensured = append(ensured, uid)
		return wechatReceiverStatus{Status: "registered"}
	}
	t.Cleanup(func() { ensureWechatProfitSharingReceiver = orig })

	req := newLegalNameUpdateRequest("user-legal-backfill", " 王小明 ")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var parsed struct {
		Status                string `json:"status"`
		LegalName             string `json:"legal_name"`
		WechatReceiverStatus  string `json:"wechat_receiver_status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("json: %v", err)
	}
	if parsed.Status != "ok" || parsed.LegalName != "王小明" || parsed.WechatReceiverStatus != "registered" {
		t.Fatalf("parsed=%+v", parsed)
	}

	var stored string
	if err := db.QueryRow(`SELECT legal_name FROM referral_code WHERE user_id = ?`, "user-legal-backfill").Scan(&stored); err != nil {
		t.Fatalf("query: %v", err)
	}
	if stored != "王小明" {
		t.Fatalf("stored=%q", stored)
	}
	if len(ensured) != 1 || ensured[0] != "user-legal-backfill" {
		t.Fatalf("ensured=%v", ensured)
	}

	// 更新后 status 应回显新名称。
	status, err := getReferralCodeStatus("user-legal-backfill")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.LegalName != "王小明" {
		t.Fatalf("status name=%q", status.LegalName)
	}
}
