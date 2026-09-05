package main

import (
	"net/http"
	"testing"

	"taskGitOauth/infrastructure"
)

// febFromRequest 回归测试（v106 安全加固 Phase 0）。
//
// 背景：公网入口计划加 Referrer-Policy: strict-origin-when-cross-origin，
// 同源/子域请求仍带完整 Referer，但跨源跳转只剩 origin（或为空）。
// febFromRequest 负责从 Origin/Referer 解析 OAuth 重定向 FE 来源，
// 若降级错误会导致多入口（公网 www.daydaymoney.com / 内网 10.2.150.68）下
// 重定向回错误前端。本测试锁定「无 Referer/Origin 回退」「allowlist 匹配」
// 「异源拒绝」三类行为，防止 Referrer-Policy 收紧后回归。
//
// 场景与设计文档 §3 Phase 0 第 4 项一致：
//   - 无 Origin 且无 Referer → fallback（浏览器隐私模式/直连书签）
//   - Referer 跨源（域名不在 allowlist）→ fallback
//   - Origin 匹配 allowlist → 优先采用
func newTestApp() *App {
	return &App{Cfg: &infrastructure.Config{
		PublicEntryOrigins: []string{
			"https://www.daydaymoney.com",
			"https://daydaymoney.com",
			"http://10.2.150.68",
		},
		FrontendBase: "https://www.daydaymoney.com",
	}}
}

func reqWithHeaders(headers map[string]string) *http.Request {
	r, err := http.NewRequest("GET", "http://example.test/callback", nil)
	if err != nil {
		panic(err)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestFebFromRequest_OriginAllowed(t *testing.T) {
	a := newTestApp()
	r := reqWithHeaders(map[string]string{"Origin": "https://www.daydaymoney.com"})
	if got := a.febFromRequest(r, "https://fallback.example"); got != "https://www.daydaymoney.com" {
		t.Fatalf("Origin 命中 allowlist 应返回 origin，got %q", got)
	}
}

func TestFebFromRequest_OriginTrailingSlashTrimmed(t *testing.T) {
	a := newTestApp()
	r := reqWithHeaders(map[string]string{"Origin": "https://www.daydaymoney.com/"})
	if got := a.febFromRequest(r, "https://fallback.example"); got != "https://www.daydaymoney.com" {
		t.Fatalf("Origin 尾斜杠应被裁剪，got %q", got)
	}
}

func TestFebFromRequest_RefererAllowed(t *testing.T) {
	a := newTestApp()
	// 同源 Referer 带路径与 query：strict-origin-when-cross-origin 下同源完整保留，
	// 解析后应只取 scheme://host。
	r := reqWithHeaders(map[string]string{"Referer": "https://daydaymoney.com/oauth/start?from=task2app"})
	if got := a.febFromRequest(r, "https://fallback.example"); got != "https://daydaymoney.com" {
		t.Fatalf("Referer 命中 allowlist 应返回 scheme://host，got %q", got)
	}
}

func TestFebFromRequest_NoOriginNoReferer_Fallback(t *testing.T) {
	a := newTestApp()
	r := reqWithHeaders(nil)
	if got := a.febFromRequest(r, "https://www.daydaymoney.com"); got != "https://www.daydaymoney.com" {
		t.Fatalf("无 Origin/Referer 应回退 FrontendBase，got %q", got)
	}
	// 空 fallback（配置缺失场景）也不应 panic，返回空串。
	if got := a.febFromRequest(r, ""); got != "" {
		t.Fatalf("空 fallback 应返回空串，got %q", got)
	}
}

func TestFebFromRequest_OriginNotAllowed_Fallback(t *testing.T) {
	a := newTestApp()
	// 攻击者可控的任意站点 Origin（CSRF 站点）不得被采用。
	r := reqWithHeaders(map[string]string{"Origin": "https://evil.example"})
	if got := a.febFromRequest(r, "https://www.daydaymoney.com"); got != "https://www.daydaymoney.com" {
		t.Fatalf("异源 Origin 应回退，got %q", got)
	}
}

func TestFebFromRequest_RefererCrossOriginNotAllowed_Fallback(t *testing.T) {
	a := newTestApp()
	// 跨源 Referer：strict-origin-when-cross-origin 收紧后 Referer 只带 origin；
	// 若该 origin 不在 allowlist（如第三方广告/联盟页跳入），必须回退。
	r := reqWithHeaders(map[string]string{"Referer": "https://search.example.com/ref"})
	if got := a.febFromRequest(r, "https://www.daydaymoney.com"); got != "https://www.daydaymoney.com" {
		t.Fatalf("异源 Referer 应回退，got %q", got)
	}
}

func TestFebFromRequest_OriginRejectedButRefererAllowed(t *testing.T) {
	a := newTestApp()
	// Origin 非 allowlist 但 Referer 命中：Origin 优先失败后降级 Referer。
	r := reqWithHeaders(map[string]string{
		"Origin":  "null", // 隐私模式/文件协议下 Origin 为字面量 null
		"Referer": "http://10.2.150.68/dashboard",
	})
	if got := a.febFromRequest(r, "https://www.daydaymoney.com"); got != "http://10.2.150.68" {
		t.Fatalf("Origin 拒绝后应降级匹配 Referer，got %q", got)
	}
}

func TestFebFromRequest_MalformedReferer_Fallback(t *testing.T) {
	a := newTestApp()
	cases := map[string]string{
		"裸域名":    "www.daydaymoney.com",        // 无 scheme
		"空 host": "https://",                 // url.Parse 成功但 Host 空
		"非法 url": "://broken%%%",             // url.Parse 失败
		"纯路径":    "/callback?code=xxx",       // 相对引用
		"控制字符":   "https://\x7f.example.com", // 不可打印字符
	}
	for name, referer := range cases {
		r := reqWithHeaders(map[string]string{"Referer": referer})
		if got := a.febFromRequest(r, "https://www.daydaymoney.com"); got != "https://www.daydaymoney.com" {
			t.Fatalf("畸形 Referer（%s）应回退，got %q", name, got)
		}
	}
}

func TestFebFromRequest_SubdomainRefererNotAllowed_Fallback(t *testing.T) {
	a := newTestApp()
	// 子域不在 allowlist：HSTS includeSubDomains 前盘点发现的子域若未登记，
	// 不得静默接受（防子域被攻击者接管后利用 Referer 注入重定向）。
	r := reqWithHeaders(map[string]string{"Referer": "https://static.daydaymoney.com/file.js"})
	if got := a.febFromRequest(r, "https://www.daydaymoney.com"); got != "https://www.daydaymoney.com" {
		t.Fatalf("未登记子域 Referer 应回退，got %q", got)
	}
}

func TestFebFromRequest_FallbackTrailingSlashTrimmed(t *testing.T) {
	a := newTestApp()
	r := reqWithHeaders(nil)
	if got := a.febFromRequest(r, "https://www.daydaymoney.com/"); got != "https://www.daydaymoney.com" {
		t.Fatalf("fallback 尾斜杠应被裁剪，got %q", got)
	}
}
