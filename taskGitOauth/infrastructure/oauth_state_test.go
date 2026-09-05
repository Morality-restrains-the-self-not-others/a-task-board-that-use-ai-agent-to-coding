package infrastructure

import (
	"strings"
	"testing"
	"time"
)

func TestOAuthStateRoundTrip(t *testing.T) {
	s := NewSessionStore("test-secret", "")
	enc, err := s.EncodeOAuthState(OAuthBrowserState{
		CSRF:        "csrf-token",
		UID:         "827923618451263488",
		Next:        "/profile/git-site-oauth/",
		Feb:         "https://www.daydaymoney.com",
		RedirectURI: "https://gitoauth.api.daydaymoney.com/api/accounts/github/oauth/callback/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, "v1.") {
		t.Fatalf("prefix: %s", enc)
	}
	got, signed, err := s.DecodeOAuthState(enc)
	if err != nil || !signed || got == nil {
		t.Fatalf("decode: signed=%v err=%v", signed, err)
	}
	if got.UID != "827923618451263488" || got.CSRF != "csrf-token" {
		t.Fatalf("mismatch: %+v", got)
	}
}

func TestOAuthStateGrantFieldsRoundTripAndLegacyDecode(t *testing.T) {
	s := NewSessionStore("test-secret", "")
	enc, err := s.EncodeOAuthState(OAuthBrowserState{
		CSRF: "c", UID: "1", RedirectURI: "https://example/callback/",
		GrantKind: "project", GrantID: "p1", RepoURL: "https://github.com/a/b.git",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, signed, err := s.DecodeOAuthState(enc)
	if err != nil || !signed || got.GrantKind != "project" || got.GrantID != "p1" {
		t.Fatalf("grant fields: %+v err=%v", got, err)
	}
	legacy, err := s.EncodeOAuthState(OAuthBrowserState{
		CSRF: "c", UID: "1", RedirectURI: "https://example/callback/",
	})
	if err != nil {
		t.Fatal(err)
	}
	old, signed, err := s.DecodeOAuthState(legacy)
	if err != nil || !signed || old.GrantKind != "" || old.GrantID != "" {
		t.Fatalf("legacy state must decode without grant fields: %+v err=%v", old, err)
	}
}

func TestOAuthStateLegacyPlainCSRF(t *testing.T) {
	s := NewSessionStore("test-secret", "")
	got, signed, err := s.DecodeOAuthState("Um8LnQ94LEQJgeqHrF8RuwcmuUxQaeus")
	if err != nil || signed || got != nil {
		t.Fatalf("legacy plain csrf should be unsigned: got=%v signed=%v err=%v", got, signed, err)
	}
}

func TestOAuthStateRejectsTamperAndExpiry(t *testing.T) {
	s := NewSessionStore("test-secret", "")
	enc, err := s.EncodeOAuthState(OAuthBrowserState{
		CSRF: "c", UID: "1", RedirectURI: "https://example/callback/",
		Exp: time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.DecodeOAuthState(enc); err == nil {
		t.Fatal("expected expiry error")
	}
	enc2, _ := s.EncodeOAuthState(OAuthBrowserState{
		CSRF: "c", UID: "1", RedirectURI: "https://example/callback/",
	})
	tampered := enc2[:len(enc2)-4] + "xxxx"
	if _, _, err := s.DecodeOAuthState(tampered); err == nil {
		t.Fatal("expected tamper error")
	}
}
