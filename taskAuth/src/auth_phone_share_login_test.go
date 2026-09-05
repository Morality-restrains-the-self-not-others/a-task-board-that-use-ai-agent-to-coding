package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleLoginPhonePasswordDisambiguatesDistinctPasswords(t *testing.T) {
	setupAuthTestDB(t)
	userA, err := createUserWithPhoneLogin("+86", "13800138108", "Passw0rdA!", "")
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	userB, err := createUserWithPhoneLogin("+86", "13800138108", "Passw0rdB!", "")
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	// Customer /api/auth/ rejects is_staff; testers still share phones without that gate.
	mustMarkPhoneShareRole(t, userA, false, false, true)
	mustMarkPhoneShareRole(t, userB, false, false, true)

	recA := postPhonePasswordLogin(t, "+8613800138108", "Passw0rdA!")
	if recA.Code != http.StatusOK {
		t.Fatalf("login A expected 200, got %d %s", recA.Code, recA.Body.String())
	}
	var respA map[string]interface{}
	if err := json.Unmarshal(recA.Body.Bytes(), &respA); err != nil {
		t.Fatalf("json A: %v", err)
	}
	userMap, _ := respA["user"].(map[string]interface{})
	if userMap == nil || userMap["id"] != userA {
		t.Fatalf("expected user A %s, got %v", userA, userMap)
	}

	recB := postPhonePasswordLogin(t, "+8613800138108", "Passw0rdB!")
	if recB.Code != http.StatusOK {
		t.Fatalf("login B expected 200, got %d %s", recB.Code, recB.Body.String())
	}
	var respB map[string]interface{}
	if err := json.Unmarshal(recB.Body.Bytes(), &respB); err != nil {
		t.Fatalf("json B: %v", err)
	}
	userMapB, _ := respB["user"].(map[string]interface{})
	if userMapB == nil || userMapB["id"] != userB {
		t.Fatalf("expected user B %s, got %v", userB, userMapB)
	}
}

func TestHandleLoginPhonePasswordAmbiguousWhenSamePassword(t *testing.T) {
	setupAuthTestDB(t)
	const sharedPass = "SamePassw0rd!"
	userA, err := createUserWithPhoneLogin("+86", "13800138109", sharedPass, "")
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	userB, err := createUserWithPhoneLogin("+86", "13800138109", sharedPass, "")
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	// Same as TestHandleLoginPhonePasswordDisambiguatesDistinctPasswords: tester, not staff.
	mustMarkPhoneShareRole(t, userA, false, false, true)
	mustMarkPhoneShareRole(t, userB, false, false, true)

	rec := postPhonePasswordLogin(t, "+8613800138109", sharedPass)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 ambiguous, got %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	if body["code"] != "phone_ambiguous" {
		t.Fatalf("expected phone_ambiguous, got %v body=%s", body["code"], rec.Body.String())
	}
}

func TestInternalPhoneTakenFalseWhenStaffCanShare(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"
	userA := mustCreateWechatTestUser(t, "u-int-share-a")
	userB := mustCreateWechatTestUser(t, "u-int-share-b")
	mustMarkPhoneShareRole(t, userA, true, false, false)
	mustMarkPhoneShareRole(t, userB, true, false, false)
	if err := upsertPhoneLoginMethod(userA, "+86", "13700018888"); err != nil {
		t.Fatalf("a: %v", err)
	}

	takenReq := httptest.NewRequest(http.MethodPost, "/api/internal/login-methods/phone-taken/", strings.NewReader(
		`{"country_calling_code":"+86","national_number":"13700018888","exclude_user_id":"`+userB+`"}`,
	))
	takenReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	takenRec := httptest.NewRecorder()
	handlePhoneTaken(takenRec, takenReq)
	if takenRec.Code != http.StatusOK {
		t.Fatalf("phone-taken: %d %s", takenRec.Code, takenRec.Body.String())
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(takenRec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["taken"] != false {
		t.Fatalf("staff share should not be taken, got %v", payload["taken"])
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/internal/users/"+userB+"/phone-login-method/", strings.NewReader(
		`{"country_calling_code":"+86","national_number":"13700018888"}`,
	))
	patchReq.SetPathValue("user_id", userB)
	patchReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	patchRec := httptest.NewRecorder()
	handleUpsertPhoneLoginMethod(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("upsert share expected 200, got %d %s", patchRec.Code, patchRec.Body.String())
	}
}

func TestSystemAdminUnarchivePrivilegedSharedPhoneAllowed(t *testing.T) {
	setupAuthTestDB(t)
	userA, err := createUserWithPhoneLogin("+86", "13800138110", "password123", "")
	if err != nil {
		t.Fatalf("userA: %v", err)
	}
	mustMarkPhoneShareRole(t, userA, true, false, false)
	archiveUserByID(t, userA)

	userB, err := createUserWithPhoneLogin("+86", "13800138110", "password456", "")
	if err != nil {
		t.Fatalf("userB: %v", err)
	}
	mustMarkPhoneShareRole(t, userB, true, false, false)

	code, body := unarchiveSystemAdminUser(t, userA)
	if code != http.StatusOK {
		t.Fatalf("privileged share unarchive expected 200, got %d body=%s", code, body)
	}
}
