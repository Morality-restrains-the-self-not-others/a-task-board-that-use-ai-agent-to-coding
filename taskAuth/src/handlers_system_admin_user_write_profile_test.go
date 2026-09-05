package main

import (
	"net/http"
	"testing"
)

func liveProfileUsername(t *testing.T, userID string) string {
	t.Helper()
	var username string
	err := db.QueryRow(`SELECT username FROM auth_user_profile WHERE user_id = ?`, userID).Scan(&username)
	if err != nil {
		return ""
	}
	return username
}

func liveEmailPasswordHash(t *testing.T, userID string) string {
	t.Helper()
	var hash string
	err := db.QueryRow(`
		SELECT password_hash FROM auth_login_method
		WHERE object_id = ? AND method_type = 'email' AND binding_voided_at IS NULL
		LIMIT 1`, userID).Scan(&hash)
	if err != nil {
		t.Fatalf("password hash: %v", err)
	}
	return hash
}

func TestPatchUserAsAdmin_usernameWritesProfile(t *testing.T) {
	setupAuthTestDB(t)

	targetID, _, err := createUserWithEmailLogin("rename-target@test.com", "hash")
	if err != nil {
		t.Fatalf("target: %v", err)
	}

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"username": "运营改名",
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, body)
	}
	if got := liveProfileUsername(t, targetID); got != "运营改名" {
		t.Fatalf("expected profile username 运营改名, got %q", got)
	}
}

func TestPatchUserAsAdmin_passwordUpdatesHash(t *testing.T) {
	setupAuthTestDB(t)

	targetID, _, err := createUserWithEmailLogin("pwd-target@test.com", "old-secret")
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	before := liveEmailPasswordHash(t, targetID)

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"password": "NewSecret12!",
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, body)
	}
	after := liveEmailPasswordHash(t, targetID)
	if after == "" || after == before {
		t.Fatalf("password hash must change, before=%q after=%q", before, after)
	}
	if !checkPasswordHash("NewSecret12!", after) {
		t.Fatalf("new hash must verify NewSecret12!")
	}
}

func TestPatchUserAsAdmin_emptyPasswordDoesNotChangeHash(t *testing.T) {
	setupAuthTestDB(t)

	targetID, _, err := createUserWithEmailLogin("pwd-keep@test.com", "keep-secret")
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	before := liveEmailPasswordHash(t, targetID)

	code, body := putSystemAdminUser(t, targetID, "bootstrap-admin", map[string]interface{}{
		"username": "仍改名",
		"password": "",
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", code, body)
	}
	if got := liveEmailPasswordHash(t, targetID); got != before {
		t.Fatalf("empty password must not change hash")
	}
	if got := liveProfileUsername(t, targetID); got != "仍改名" {
		t.Fatalf("expected profile username 仍改名, got %q", got)
	}
}
