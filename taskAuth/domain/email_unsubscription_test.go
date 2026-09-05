package domain

import "testing"

func TestNormalizeInviteEmail(t *testing.T) {
	got, err := NormalizeInviteEmail("  A@B.C ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "a@b.c" {
		t.Fatalf("got %q", got)
	}
	if _, err := NormalizeInviteEmail("nodomain"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSignParseUnsubscribeTokenRoundTrip(t *testing.T) {
	const secret = "test-unsub-hmac-not-for-prod"
	tok, err := SignUnsubscribeToken(secret, "User@Example.COM")
	if err != nil {
		t.Fatal(err)
	}
	email, err := ParseUnsubscribeToken(secret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if email != "user@example.com" {
		t.Fatalf("email=%q", email)
	}
}

func TestParseUnsubscribeTokenRejectsTamper(t *testing.T) {
	const secret = "test-unsub-hmac-not-for-prod"
	tok, err := SignUnsubscribeToken(secret, "a@b.c")
	if err != nil {
		t.Fatal(err)
	}
	tampered := tok[:len(tok)-2] + "xx"
	if _, err := ParseUnsubscribeToken(secret, tampered); err == nil {
		t.Fatal("expected invalid token")
	}
	if _, err := ParseUnsubscribeToken(secret, ""); err == nil {
		t.Fatal("expected invalid token")
	}
	if _, err := SignUnsubscribeToken("", "a@b.c"); err == nil {
		t.Fatal("empty secret must fail")
	}
}

func TestShouldSkipInviteEmail(t *testing.T) {
	if !ShouldSkipInviteEmail(true) {
		t.Fatal("unsubscribed must skip")
	}
	if ShouldSkipInviteEmail(false) {
		t.Fatal("subscribed must send")
	}
}
