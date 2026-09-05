package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	BindPending = "pending"
	BindActive  = "active"
	BindFailed  = "failed"
)

// OauthCredentialBinding is the aggregate root for OAuth app credential binding.
type OauthCredentialBinding struct {
	ProviderKey        string
	Task2appUserID     int64
	GitUserID          string
	GitLogin           string
	Scope              string
	RefreshTokenCipher string
	BindStatus         string
	BindError          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NormalizeBindStatus(raw string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case BindPending, BindActive, BindFailed:
		return s, nil
	default:
		return "", fmt.Errorf("invalid bind_status: %s", raw)
	}
}

func SanitizeBindError(raw string) string {
	s := strings.TrimSpace(raw)
	lower := strings.ToLower(s)
	if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "authorization") {
		return "unsafe_bind_error_redacted"
	}
	if len(s) > 512 {
		return s[:512]
	}
	return s
}

func (b *OauthCredentialBinding) MarkPending(at time.Time) {
	b.BindStatus = BindPending
	b.BindError = ""
	b.UpdatedAt = at
}

func (b *OauthCredentialBinding) MarkBindActive(at time.Time) error {
	if b.BindStatus != BindPending {
		return fmt.Errorf("仅允许从 pending 迁移到 active")
	}
	b.BindStatus = BindActive
	b.BindError = ""
	b.UpdatedAt = at
	return nil
}

func (b *OauthCredentialBinding) MarkBindFailed(reason string, at time.Time) error {
	if b.BindStatus != BindPending {
		return fmt.Errorf("仅允许从 pending 迁移到 failed")
	}
	b.BindStatus = BindFailed
	b.BindError = SanitizeBindError(reason)
	b.UpdatedAt = at
	return nil
}

func (b *OauthCredentialBinding) IsUsableForAccessIssue() bool {
	return b.BindStatus == BindActive && strings.TrimSpace(b.RefreshTokenCipher) != ""
}

func ProviderKey(provider, serviceProvider string) string {
	p := strings.TrimSpace(strings.ToLower(provider))
	if p == "" {
		p = "github"
	}
	sp := strings.TrimSpace(strings.ToLower(serviceProvider))
	if sp == "" {
		sp = "default"
	}
	return p + ":" + sp
}

func ParseProviderKey(providerKey, fallback string) (provider, serviceProvider string) {
	raw := strings.TrimSpace(strings.ToLower(providerKey))
	fb := strings.TrimSpace(strings.ToLower(fallback))
	if fb == "" {
		fb = "github"
	}
	if i := strings.Index(raw, ":"); i >= 0 {
		p := strings.TrimSpace(raw[:i])
		sp := strings.TrimSpace(raw[i+1:])
		if p == "" {
			p = fb
		}
		if sp == "" {
			sp = "default"
		}
		return p, sp
	}
	if raw == "" {
		return fb, "default"
	}
	return raw, "default"
}
