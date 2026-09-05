package infrastructure

import "testing"

// TestLoadConfigGithubOutboundProxyEmpty — 回归 2026-08-07：本机开发代理
// socks5://127.0.0.1:1080 曾作为 outbound_proxy 固化在 provider 配置中，生产未运行
// 该代理导致 GitHub 换票必失败（dial tcp 127.0.0.1:1080: connection refused）。
// 配置已移除：断言默认加载后出站代理为空（直连），防止死代理配置回归。
func TestLoadConfigGithubOutboundProxyEmpty(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GithubOutboundProxy != "" {
		t.Fatalf("GithubOutboundProxy=%q, want empty (dead local dev proxy must not be a production egress)", cfg.GithubOutboundProxy)
	}
}
