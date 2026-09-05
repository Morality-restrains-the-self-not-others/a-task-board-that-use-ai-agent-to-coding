package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendPasswordResetLinkRequiresEmail(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/send_password_reset_link/", nil)
	rec := httptest.NewRecorder()
	handleSendPasswordResetLink(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestResetPasswordWithLinkRequiresToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/accounts/users/reset-password-with-link/", nil)
	rec := httptest.NewRecorder()
	handleResetPasswordWithLinkPrefix(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// OPT-20260807-018 回归护栏: 邮件正文链接必须是绝对 URL——
// 裸相对路径在邮件客户端无正确基址，点击解析到邮件服务商域名 → 404。
func TestBuildUserFacingURL(t *testing.T) {
	old := cfg.FrontendBase
	defer func() { cfg.FrontendBase = old }()

	cfg.FrontendBase = "https://www.daydaymoney.com"
	got := buildUserFacingURL("/auth/reset-password/abc123/")
	want := "https://www.daydaymoney.com/auth/reset-password/abc123/"
	if got != want {
		t.Fatalf("buildUserFacingURL = %q, want %q", got, want)
	}

	// FrontendBase 带尾斜杠时不应产生 // 双斜杠
	cfg.FrontendBase = "https://www.daydaymoney.com/"
	got = buildUserFacingURL("/auth/activate/xyz/")
	want = "https://www.daydaymoney.com/auth/activate/xyz/"
	if got != want {
		t.Fatalf("buildUserFacingURL with trailing slash = %q, want %q", got, want)
	}

	// 未配置时回退 localhost
	cfg.FrontendBase = ""
	got = buildUserFacingURL("/auth/reset-password/abc/")
	if !strings.HasPrefix(got, "http://localhost:4000/auth/reset-password/abc/") {
		t.Fatalf("buildUserFacingURL fallback = %q, want localhost prefix", got)
	}
}
