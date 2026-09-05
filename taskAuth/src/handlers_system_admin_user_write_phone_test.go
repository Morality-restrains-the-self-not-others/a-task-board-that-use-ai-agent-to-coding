package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func putSystemAdminUser(t *testing.T, userID, actorID string, body map[string]interface{}) (int, string) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/system-admin/users/"+userID+"/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", actorID)
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	return rec.Code, rec.Body.String()
}

func livePhoneIdentifier(t *testing.T, userID string) string {
	t.Helper()
	var identifier string
	err := db.QueryRow(`
		SELECT identifier FROM auth_login_method
		WHERE object_id = ? AND method_type = 'phone' AND binding_voided_at IS NULL
		LIMIT 1`, userID).Scan(&identifier)
	if err != nil {
		return ""
	}
	return identifier
}

// Reproduces the system-admin edit form: PUT a phone already bound to another
// active user must 409 with phone_taken and must not rewrite the target's phone.
func TestPatchUserAsAdmin_phoneTakenReturns409(t *testing.T) {
	setupAuthTestDB(t)

	holderID, _, err := createUserWithEmailLogin("phone-holder@test.com", "hash")
	if err != nil {
		t.Fatalf("holder: %v", err)
	}
	if err := upsertPhoneLoginMethod(holderID, "+86", "13900001111"); err != nil {
		t.Fatalf("bind holder phone: %v", err)
	}

	targetID, _, err := createUserWithEmailLogin("phone-target@test.com", "hash")
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	if err := upsertPhoneLoginMethod(targetID, "+86", "18959264502"); err != nil {
		t.Fatalf("bind target phone: %v", err)
	}

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"username":     "软刀",
		"email":        "phone-target@test.com",
		"phone":        "13900001111",
		"is_superuser": false,
		"is_staff":     false,
		"is_tenant":    true,
		"is_tester":    false,
	})
	if code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", code, body)
	}
	if !strings.Contains(body, "phone_taken") {
		t.Fatalf("expected code phone_taken, got %s", body)
	}
	if !strings.Contains(body, "手机号") {
		t.Fatalf("conflict message should mention 手机号, got %s", body)
	}
	if got := livePhoneIdentifier(t, targetID); got != "18959264502" {
		t.Fatalf("target phone must stay 18959264502, got %q", got)
	}
}

func TestPatchUserAsAdmin_emptyPhoneVoidsLoginMethod(t *testing.T) {
	setupAuthTestDB(t)

	targetID, _, err := createUserWithEmailLogin("phone-unbind@test.com", "hash")
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	if err := upsertPhoneLoginMethod(targetID, "+86", "13900003333"); err != nil {
		t.Fatalf("bind target phone: %v", err)
	}

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"username":     "解绑用户",
		"email":        "phone-unbind@test.com",
		"phone":        "",
		"is_superuser": false,
		"is_staff":     false,
		"is_tenant":    true,
		"is_tester":    false,
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, body)
	}
	if got := livePhoneIdentifier(t, targetID); got != "" {
		t.Fatalf("expected phone unbound, still %q", got)
	}
}

func TestPatchUserAsAdmin_omitPhoneKeepsLoginMethod(t *testing.T) {
	setupAuthTestDB(t)

	targetID, _, err := createUserWithEmailLogin("phone-keep@test.com", "hash")
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	if err := upsertPhoneLoginMethod(targetID, "+86", "13900004444"); err != nil {
		t.Fatalf("bind target phone: %v", err)
	}

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"is_staff": true,
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, body)
	}
	if got := livePhoneIdentifier(t, targetID); got != "13900004444" {
		t.Fatalf("omitting phone must keep 13900004444, got %q", got)
	}
}

func TestPatchUserAsAdmin_emptyPhoneWhenAlreadyUnboundIsOK(t *testing.T) {
	setupAuthTestDB(t)

	targetID, _, err := createUserWithEmailLogin("phone-already-free@test.com", "hash")
	if err != nil {
		t.Fatalf("target: %v", err)
	}

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"phone": "",
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, body)
	}
	if got := livePhoneIdentifier(t, targetID); got != "" {
		t.Fatalf("expected still unbound, got %q", got)
	}
}

func TestPatchUserAsAdmin_phoneAvailableUpdatesLoginMethod(t *testing.T) {
	setupAuthTestDB(t)

	targetID, _, err := createUserWithEmailLogin("phone-free@test.com", "hash")
	if err != nil {
		t.Fatalf("target: %v", err)
	}

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"email":        "phone-free@test.com",
		"phone":        "13900001111",
		"is_superuser": false,
		"is_staff":     false,
		"is_tenant":    true,
		"is_tester":    false,
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, body)
	}
	if got := livePhoneIdentifier(t, targetID); got != "13900001111" {
		t.Fatalf("expected phone 13900001111, got %q", got)
	}
}

func postSystemAdminCreateUser(t *testing.T, actorID string, body map[string]interface{}) (int, string) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/system-admin/users/create/", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", actorID)
	req.Header.Set("X-User-Roles", "super_admin")
	rec := httptest.NewRecorder()
	handleSystemAdminUsers(rec, req)
	return rec.Code, rec.Body.String()
}

func countUsersByEmail(t *testing.T, email string) int {
	t.Helper()
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM auth_user u
		INNER JOIN auth_login_method lm ON lm.object_id = u.id
		WHERE lm.method_type = 'email' AND LOWER(lm.identifier) = LOWER(?)`, email).Scan(&n)
	if err != nil {
		t.Fatalf("count email %s: %v", email, err)
	}
	return n
}

func TestCreateUserAsAdmin_phoneTakenReturns409AndDoesNotCreate(t *testing.T) {
	setupAuthTestDB(t)

	holderID, _, err := createUserWithEmailLogin("create-phone-holder@test.com", "hash")
	if err != nil {
		t.Fatalf("holder: %v", err)
	}
	if err := upsertPhoneLoginMethod(holderID, "+86", "13900001111"); err != nil {
		t.Fatalf("bind holder phone: %v", err)
	}

	newEmail := "create-phone-new@test.com"
	code, body := postSystemAdminCreateUser(t, "bootstrap-admin", map[string]interface{}{
		"username": "新人",
		"email":    newEmail,
		"phone":    "13900001111",
		"password": "secret12",
	})
	if code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", code, body)
	}
	if !strings.Contains(body, "phone_taken") {
		t.Fatalf("expected code phone_taken, got %s", body)
	}
	if countUsersByEmail(t, newEmail) != 0 {
		t.Fatalf("must not leave a half-created user for %s", newEmail)
	}
}

func TestCreateUserAsAdmin_phoneAvailableBindsLoginMethod(t *testing.T) {
	setupAuthTestDB(t)

	newEmail := "create-phone-free@test.com"
	code, body := postSystemAdminCreateUser(t, "bootstrap-admin", map[string]interface{}{
		"email":    newEmail,
		"phone":    "13900002222",
		"password": "secret12",
	})
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", code, body)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	userID, _ := decoded["id"].(string)
	if userID == "" {
		t.Fatalf("expected id in body %s", body)
	}
	if got := livePhoneIdentifier(t, userID); got != "13900002222" {
		t.Fatalf("expected phone 13900002222, got %q", got)
	}
}

func TestPatchUserAsAdmin_emailTakenReturns409(t *testing.T) {
	setupAuthTestDB(t)

	_, _, err := createUserWithEmailLogin("email-holder@test.com", "hash")
	if err != nil {
		t.Fatalf("holder: %v", err)
	}
	targetID, _, err := createUserWithEmailLogin("email-target@test.com", "hash")
	if err != nil {
		t.Fatalf("target: %v", err)
	}

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"email":        "email-holder@test.com",
		"is_superuser": false,
		"is_staff":     false,
		"is_tenant":    true,
		"is_tester":    false,
	})
	if code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", code, body)
	}
	if !strings.Contains(body, "email_taken") {
		t.Fatalf("expected code email_taken, got %s", body)
	}
}
