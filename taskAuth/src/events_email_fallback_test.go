package main

import (
	"strings"
	"testing"
)

func TestBuildFallbackEmailVerificationCodeIncludesOTP(t *testing.T) {
	ctx := map[string]interface{}{"code": "482913"}
	text := buildFallbackEmailText("您的验证码", "verification_code", ctx)
	if !strings.Contains(text, "482913") {
		t.Fatalf("text missing OTP: %s", text)
	}
	html := buildFallbackEmailHTML("您的验证码", "verification_code", ctx)
	if !strings.Contains(html, "482913") {
		t.Fatalf("html missing OTP: %s", html)
	}
}

func TestBuildFallbackEmailVerificationCodeEmptyCodeFallsBackToSubject(t *testing.T) {
	text := buildFallbackEmailText("您的验证码", "verification_code", map[string]interface{}{})
	if text != "您的验证码" {
		t.Fatalf("text=%q", text)
	}
}
