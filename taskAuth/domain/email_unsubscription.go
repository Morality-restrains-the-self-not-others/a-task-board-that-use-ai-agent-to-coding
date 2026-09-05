package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	UnsubscribeTokenVersion = "v1"
	unsubscribeMACPrefix    = "email-unsub-v1|"
	UnsubscribeSourceInvite = "invite_email"
	UnsubscribeSourceList   = "list_unsubscribe_post"
	EmailUnsubscribedHint   = "该邮箱已退订邮件邀请，请手动复制邀请链接给对方"
)

// NormalizeInviteEmail trims and lowercases an address. It must contain "@".
func NormalizeInviteEmail(email string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(email))
	if n == "" || !strings.Contains(n, "@") {
		return "", fmt.Errorf("invalid email")
	}
	return n, nil
}

// SignUnsubscribeToken returns v1.{base64url(email)}.{base64url(mac)}.
func SignUnsubscribeToken(secret, email string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("unsubscribe hmac secret empty")
	}
	norm, err := NormalizeInviteEmail(email)
	if err != nil {
		return "", err
	}
	mac := unsubscribeMAC(secret, norm)
	return UnsubscribeTokenVersion + "." +
		base64.RawURLEncoding.EncodeToString([]byte(norm)) + "." +
		base64.RawURLEncoding.EncodeToString(mac), nil
}

// ParseUnsubscribeToken verifies HMAC and returns the normalized email.
func ParseUnsubscribeToken(secret, token string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("unsubscribe hmac secret empty")
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != UnsubscribeTokenVersion {
		return "", fmt.Errorf("invalid unsubscribe token")
	}
	rawEmail, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid unsubscribe token")
	}
	gotMAC, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid unsubscribe token")
	}
	norm, err := NormalizeInviteEmail(string(rawEmail))
	if err != nil {
		return "", err
	}
	want := unsubscribeMAC(secret, norm)
	if !hmac.Equal(gotMAC, want) {
		return "", fmt.Errorf("invalid unsubscribe token")
	}
	return norm, nil
}

func unsubscribeMAC(secret, normalizedEmail string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(unsubscribeMACPrefix + normalizedEmail))
	return mac.Sum(nil)
}

// ShouldSkipInviteEmail is true when the recipient has unsubscribed from invite mail.
func ShouldSkipInviteEmail(unsubscribed bool) bool {
	return unsubscribed
}
