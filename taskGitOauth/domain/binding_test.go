package domain

import (
	"testing"
	"time"
)

func TestBindStateMachine(t *testing.T) {
	now := time.Now().UTC()
	b := &OauthCredentialBinding{
		ProviderKey:        "github",
		Task2appUserID:     1,
		GitUserID:          "42",
		RefreshTokenCipher: "cipher",
		BindStatus:         BindPending,
	}
	if err := b.MarkBindActive(now); err != nil {
		t.Fatal(err)
	}
	if !b.IsUsableForAccessIssue() {
		t.Fatal("expected usable")
	}
	b2 := &OauthCredentialBinding{BindStatus: BindPending}
	if err := b2.MarkBindFailed("token leaked", now); err != nil {
		t.Fatal(err)
	}
	if b2.BindError != "unsafe_bind_error_redacted" {
		t.Fatalf("got %q", b2.BindError)
	}
}

func TestParseProviderKey(t *testing.T) {
	p, sp := ParseProviderKey("gitlab:daydaymoney", "github")
	if p != "gitlab" || sp != "daydaymoney" {
		t.Fatalf("%s %s", p, sp)
	}
}
