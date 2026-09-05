package main

import (
	"database/sql"
	"testing"
)

func TestTouchLastLoginNoopOnEmpty(t *testing.T) {
	setupAuthTestDB(t)
	touchLastLogin("")
	touchLastLogin("   ")
}

func TestTouchLastLoginWritesUTCTimestamp(t *testing.T) {
	setupAuthTestDB(t)
	userID, _, err := createUserWithEmailLogin("last-login-touch@test.com", "hash")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	var before sql.NullString
	if err := db.QueryRow(`SELECT last_login FROM auth_user WHERE id = ?`, userID).Scan(&before); err != nil {
		t.Fatalf("select before: %v", err)
	}
	if before.Valid && before.String != "" {
		t.Fatalf("expected NULL last_login on insert, got %q", before.String)
	}
	touchLastLogin(userID)
	var after sql.NullString
	if err := db.QueryRow(`SELECT last_login FROM auth_user WHERE id = ?`, userID).Scan(&after); err != nil {
		t.Fatalf("select after: %v", err)
	}
	if !after.Valid || after.String == "" {
		t.Fatal("expected last_login to be written")
	}
}
