package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPhoneTakenAndUpsertLoginMethod(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"

	userA, err := createUserWithPhoneLogin("+86", "13700010001", "hash", "")
	if err != nil {
		t.Fatalf("create user A: %v", err)
	}
	userB, err := createUserWithPhoneLogin("+86", "13700010002", "hash", "")
	if err != nil {
		t.Fatalf("create user B: %v", err)
	}

	takenReq := httptest.NewRequest(http.MethodPost, "/api/internal/login-methods/phone-taken/", strings.NewReader(
		`{"country_calling_code":"+86","national_number":"13700010001","exclude_user_id":"`+userB+`"}`,
	))
	takenReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	takenRec := httptest.NewRecorder()
	handlePhoneTaken(takenRec, takenReq)
	if takenRec.Code != http.StatusOK {
		t.Fatalf("phone-taken: %d %s", takenRec.Code, takenRec.Body.String())
	}
	var takenPayload map[string]interface{}
	if err := json.Unmarshal(takenRec.Body.Bytes(), &takenPayload); err != nil {
		t.Fatalf("decode taken: %v", err)
	}
	if takenPayload["taken"] != true {
		t.Fatalf("expected taken true, got %v", takenPayload["taken"])
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/internal/users/"+userB+"/phone-login-method/", strings.NewReader(
		`{"country_calling_code":"+86","national_number":"13900005000"}`,
	))
	patchReq.SetPathValue("user_id", userB)
	patchReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	patchRec := httptest.NewRecorder()
	handleUpsertPhoneLoginMethod(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("upsert phone: %d %s", patchRec.Code, patchRec.Body.String())
	}

	lm, err := findLoginMethodByPhone("+86", "13900005000")
	if err != nil || lm == nil || lm.ObjectID != userB {
		t.Fatalf("expected userB phone binding, lm=%v err=%v", lm, err)
	}
	_ = userA
}

func TestUsernameTakenAndUpsertLoginMethod(t *testing.T) {
	setupAuthTestDB(t)
	cfg.InternalSecret = "test-secret"

	userA, _, err := createUserWithEmailLogin("userA@test.com", "hashA")
	if err != nil {
		t.Fatalf("create user A: %v", err)
	}
	userB, _, err := createUserWithEmailLogin("userB@test.com", "hashB")
	if err != nil {
		t.Fatalf("create user B: %v", err)
	}

	// Seed userA with a username login method
	if err := upsertUsernameLoginMethod(userA, "takenuser"); err != nil {
		t.Fatalf("seed username for userA: %v", err)
	}

	// Scenario 1: username taken by userA → should return taken=true
	takenReq := httptest.NewRequest(http.MethodPost, "/api/internal/login-methods/username-taken/", strings.NewReader(
		`{"identifier":"takenuser","exclude_user_id":"`+userB+`"}`,
	))
	takenReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	takenRec := httptest.NewRecorder()
	handleUsernameTaken(takenRec, takenReq)
	if takenRec.Code != http.StatusOK {
		t.Fatalf("username-taken: expected 200 got %d: %s", takenRec.Code, takenRec.Body.String())
	}
	var takenPayload map[string]interface{}
	if err := json.Unmarshal(takenRec.Body.Bytes(), &takenPayload); err != nil {
		t.Fatalf("decode taken: %v", err)
	}
	if takenPayload["taken"] != true {
		t.Fatalf("expected taken true, got %v", takenPayload["taken"])
	}

	// Scenario 2: userB patches a new username → should return 200
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/internal/users/"+userB+"/username-login-method/", strings.NewReader(
		`{"identifier":"newuser"}`,
	))
	patchReq.SetPathValue("user_id", userB)
	patchReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	patchRec := httptest.NewRecorder()
	handleUpsertUsernameLoginMethod(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("upsert username new: expected 200 got %d: %s", patchRec.Code, patchRec.Body.String())
	}

	// Verify userB now has the username
	lm, err := findLoginMethodByIdentifier("newuser")
	if err != nil || lm == nil || lm.ObjectID != userB {
		t.Fatalf("expected userB username binding, lm=%v err=%v", lm, err)
	}

	// Scenario 3: patch an already-taken username → should return 409 Conflict
	conflictReq := httptest.NewRequest(http.MethodPatch, "/api/internal/users/"+userB+"/username-login-method/", strings.NewReader(
		`{"identifier":"takenuser"}`,
	))
	conflictReq.SetPathValue("user_id", userB)
	conflictReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	conflictRec := httptest.NewRecorder()
	handleUpsertUsernameLoginMethod(conflictRec, conflictReq)
	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("upsert username conflict: expected 409 got %d: %s", conflictRec.Code, conflictRec.Body.String())
	}

	// Scenario 4: delete username by patching empty string → should return 200
	deleteReq := httptest.NewRequest(http.MethodPatch, "/api/internal/users/"+userB+"/username-login-method/", strings.NewReader(
		`{"identifier":""}`,
	))
	deleteReq.SetPathValue("user_id", userB)
	deleteReq.Header.Set("X-TaskAuth-Internal-Secret", "test-secret")
	deleteRec := httptest.NewRecorder()
	handleUpsertUsernameLoginMethod(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("upsert username delete: expected 200 got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}

	// Verify deletion
	deleted, _ := findLoginMethodByIdentifier("newuser")
	if deleted != nil {
		t.Fatalf("expected username to be deleted, got lm=%v", deleted)
	}

	_ = userA
}
