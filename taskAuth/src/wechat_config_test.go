package main

import (
	"strings"
	"testing"

	confload "confload"
)

// TestLoadWeChatConfigResolvesRedirectURITemplate 端到端回归测试：
// 从真实 monorepo 配置树加载微信配置，断言 redirectUri 中的
// ${scheme} / ${subdomains.xxx} 模板被正确展开。
//
// 背景：conf/auth/task-auth/config.yaml 中
//
//	redirectUri: "${scheme}://${subdomains.base}/api/auth/wechat/callback/"
//
// 曾因 confload 反射解析器不支持 map 结构（wechat.apps 是 map）而原样透传，
// 导致微信 qrconnect 跳转携带未解析占位符（%24%7Bscheme%7D...）。
func TestLoadWeChatConfigResolvesRedirectURITemplate(t *testing.T) {
	root := repoRoot()

	// 基线：base.yaml 解析结果（与 loadWeChatConfig 内部解析一致）
	subs := confload.ResolveBaseYaml(root)
	wantRedirect := confload.ResolveTemplate("${scheme}://${subdomains.base}/api/auth/wechat/callback/", subs)
	if strings.Contains(wantRedirect, "${") {
		t.Fatalf("test baseline unresolved: %q", wantRedirect)
	}

	wechatApps = nil
	loadWeChatConfig(root)
	if !wechatEnabled() {
		t.Skip("wechat config absent from this checkout — nothing to verify")
	}
	web := wechatApps["web"]
	if web == nil {
		t.Skip("wechat app 'web' not configured — nothing to verify")
	}
	if strings.Contains(web.RedirectURI, "${") {
		t.Errorf("redirectUri contains unresolved template: %q", web.RedirectURI)
	}
	if web.RedirectURI != wantRedirect {
		t.Errorf("redirectUri: want %q, got %q", wantRedirect, web.RedirectURI)
	}
	if strings.TrimSpace(web.AppID) == "" || strings.TrimSpace(web.AppSecret) == "" {
		t.Errorf("app web missing appId/appSecret: %+v", web)
	}
}

func TestLoadWeChatConfigIncludesMPApp(t *testing.T) {
	wechatApps = nil
	loadWeChatConfig(repoRoot())
	mp := wechatApps["mp"]
	if mp == nil {
		t.Fatal("wechat.apps.mp missing")
	}
	if mp.Type != "mp" {
		t.Fatalf("mp type %q", mp.Type)
	}
	if strings.TrimSpace(mp.AppID) == "" {
		t.Fatal("mp appId empty")
	}
	if wechatMPApp() != nil && strings.TrimSpace(mp.Token) == "" {
		t.Fatal("wechatMPApp should be nil without token")
	}
	if k := strings.TrimSpace(mp.EncodingAESKey); k != "" {
		if _, err := wechatMPAESKey(k); err != nil {
			t.Fatalf("mp encodingAESKey from local overlay is not 32-byte AES: %v", err)
		}
	}
}

func TestApplyWeChatMPEnvOverrides(t *testing.T) {
	prev := wechatApps
	t.Cleanup(func() { wechatApps = prev })
	wechatApps = map[string]*WeChatAppConfig{
		"mp": {Key: "mp", AppID: "wx31273ca77c89dffe", Type: "mp"},
	}
	t.Setenv("WECHAT_MP_TOKEN", "env-token")
	t.Setenv("WECHAT_MP_APP_SECRET", "env-secret")
	t.Setenv("WECHAT_MP_ENCODING_AES_KEY", "env-aes-key")
	applyWeChatMPEnvOverrides()
	got := wechatMPApp()
	if got == nil || got.Token != "env-token" || got.AppSecret != "env-secret" || got.EncodingAESKey != "env-aes-key" {
		t.Fatalf("mp after env token_set=%t secret_set=%t aes_key_set=%t",
			got != nil && got.Token == "env-token",
			got != nil && got.AppSecret == "env-secret",
			got != nil && got.EncodingAESKey == "env-aes-key")
	}
}
