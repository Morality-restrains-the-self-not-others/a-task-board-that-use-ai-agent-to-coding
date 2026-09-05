package main

import (
	"strings"
	"testing"
)

func ensureShareCodeTable(t *testing.T) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS referral_share_code (
		  code VARCHAR(32) NOT NULL,
		  user_id VARCHAR(64) NOT NULL,
		  channel_name VARCHAR(32) NOT NULL DEFAULT '默认',
		  is_default TINYINT(1) NOT NULL DEFAULT 1,
		  status VARCHAR(16) NOT NULL DEFAULT 'active',
		  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		  PRIMARY KEY (code),
		  UNIQUE KEY uk_referral_share_code_user_channel (user_id, channel_name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		t.Fatalf("create referral_share_code: %v", err)
	}
}

func assertOpaqueShareCode(t *testing.T, code, userID string) {
	t.Helper()
	got := strings.TrimSpace(code)
	if got == "" {
		t.Fatalf("expected opaque share code, got empty")
	}
	if got == userID || got == "u"+userID {
		t.Fatalf("share code must not be derived from userID %q, got %q", userID, got)
	}
	if len(got) != shareCodeLen {
		t.Fatalf("expected length %d, got %d code=%q", shareCodeLen, len(got), got)
	}
	for _, r := range got {
		if !strings.ContainsRune(shareCodeAlphabet, r) {
			t.Fatalf("code %q has disallowed rune %q", got, r)
		}
	}
}

func TestGenerateShareCodeAlphabet(t *testing.T) {
	forbidden := "0O1lI"
	for i := 0; i < 40; i++ {
		code, err := generateShareCode()
		if err != nil {
			t.Fatalf("generate: %v", err)
		}
		assertOpaqueShareCode(t, code, "873093522473906176")
		for _, r := range forbidden {
			if strings.ContainsRune(code, r) {
				t.Fatalf("code %q contains forbidden rune %q", code, r)
			}
		}
	}
}

func TestEnsureUserShareCodeNotDerivedFromUserID(t *testing.T) {
	setupTestReferralDB(t)
	const userID = "873093522473906176"
	code, err := ensureUserShareCode(userID)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	assertOpaqueShareCode(t, code, userID)

	again, err := ensureUserShareCode(userID)
	if err != nil {
		t.Fatalf("ensure again: %v", err)
	}
	if again != code {
		t.Fatalf("share code must be stable, first=%q second=%q", code, again)
	}
}

func TestEnsureUserShareCodeDistinctPerUser(t *testing.T) {
	setupTestReferralDB(t)
	a, err := ensureUserShareCode("user-a")
	if err != nil {
		t.Fatalf("user-a: %v", err)
	}
	b, err := ensureUserShareCode("user-b")
	if err != nil {
		t.Fatalf("user-b: %v", err)
	}
	assertOpaqueShareCode(t, a, "user-a")
	assertOpaqueShareCode(t, b, "user-b")
	if a == b {
		t.Fatalf("expected distinct codes, both %q", a)
	}
}

func TestEnsureUserShareCodeRejectsEmpty(t *testing.T) {
	setupTestReferralDB(t)
	if _, err := ensureUserShareCode("  "); err == nil {
		t.Fatal("expected error for empty user_id")
	}
}
