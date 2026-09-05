package templates_test

import (
	"strings"
	"testing"

	"taskEvents/notifications/templates"
)

// 回归 OPT-20260809-031：invitation 模板经 mapContext 渲染时 company_name 缺失
// 须兜底渲染而非报错（旧事件只带 company_id），保证消费者不因缺字段卡「投递中」。
func TestInvitationTemplateMissingCompanyNameFallsBack(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	text, html, err := engine.Render("invitation", map[string]interface{}{
		"invitation_url": "http://localhost/join/?token=t1",
		"email":          "member@example.com",
		"expiration_days": 7,
	})
	if err != nil {
		t.Fatalf("missing company_name must render, got %v", err)
	}
	if !strings.Contains(text, "你的团队") {
		t.Fatalf("expected fallback company name in text: %s", text)
	}
	if !strings.Contains(html, "你的团队") {
		t.Fatalf("expected fallback company name in html: %s", html)
	}
}

func TestInvitationTemplateWithMessage(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	text, html, err := engine.RenderInvitation(templates.InvitationData{
		CompanyName:    "测试公司",
		InvitationURL:  "http://localhost/join",
		ExpirationDays: 7,
		Message:        "欢迎加入我们团队，一起高效协作。",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "欢迎加入我们团队") {
		t.Fatalf("text missing message: %s", text)
	}
	if !strings.Contains(html, "欢迎加入我们团队") {
		t.Fatalf("html missing message: %s", html)
	}
}

func TestInvitationTemplateWithoutMessage(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	text, html, err := engine.RenderInvitation(templates.InvitationData{
		CompanyName:    "测试公司",
		InvitationURL:  "http://localhost/join",
		ExpirationDays: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "邀请人留言") {
		t.Fatalf("unexpected message block in text: %s", text)
	}
	if strings.Contains(html, "邀请人留言") {
		t.Fatalf("unexpected message block in html: %s", html)
	}
}

func TestPasswordResetTemplate(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	resetURL := "http://localhost:4000/auth/reset-password/abc123/"
	text, html, err := engine.Render("password_reset", map[string]interface{}{
		"reset_url": resetURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, resetURL) {
		t.Fatalf("text missing reset_url: %s", text)
	}
	if !strings.Contains(html, resetURL) {
		t.Fatalf("html missing reset_url: %s", html)
	}
}

func TestVerificationCodeTemplate(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	text, html, err := engine.Render("verification_code", map[string]interface{}{
		"code": "123456",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "123456") {
		t.Fatalf("text missing code: %s", text)
	}
	if !strings.Contains(html, "123456") {
		t.Fatalf("html missing code: %s", html)
	}
}

func TestUnknownTemplate(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = engine.Render("nonexistent", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
}

func TestActivationTemplate(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	activationURL := "http://localhost:4000/auth/activate/token/"
	text, html, err := engine.Render("activation", map[string]interface{}{
		"activation_url": activationURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, activationURL) {
		t.Fatalf("text missing activation_url: %s", text)
	}
	if !strings.Contains(html, activationURL) {
		t.Fatalf("html missing activation_url: %s", html)
	}
}

func TestWelcomeTemplate(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	text, html, err := engine.Render("welcome", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "欢迎使用") && !strings.Contains(text, "SaaS平台") {
		t.Fatalf("text missing welcome content: %s", text)
	}
	if !strings.Contains(html, "欢迎使用SaaS平台") {
		t.Fatalf("html missing welcome content: %s", html)
	}
}

func TestPasswordResetMissingContext(t *testing.T) {
	engine, err := templates.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = engine.Render("password_reset", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing reset_url")
	}
}
