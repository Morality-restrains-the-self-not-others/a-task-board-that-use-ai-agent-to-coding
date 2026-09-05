package main

import "testing"

func TestNormalizeInviteExpirationDays(t *testing.T) {
	got, err := normalizeInviteExpirationDays(0, true)
	if err != nil || got != defaultInviteExpirationDays {
		t.Fatalf("default: got %d err=%v", got, err)
	}
	if defaultInviteExpirationDays != 90 {
		t.Fatalf("default days = %d", defaultInviteExpirationDays)
	}
	got, err = normalizeInviteExpirationDays(365, true)
	if err != nil || got != 365 {
		t.Fatalf("365: got %d err=%v", got, err)
	}
	if _, err = normalizeInviteExpirationDays(366, true); err == nil {
		t.Fatal("expected error for 366")
	}
	if _, err = normalizeInviteExpirationDays(0, false); err == nil {
		t.Fatal("expected error for resend 0")
	}
}
