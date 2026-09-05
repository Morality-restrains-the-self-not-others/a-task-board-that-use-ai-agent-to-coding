package main

import (
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"taskGitOauth/infrastructure"
)

// captureLogOutput reroutes the standard logger for the duration of fn so logWarn
// output can be asserted on.
func captureLogOutput(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := log.Writer()
	log.SetOutput(w)
	defer func() { log.SetOutput(old) }()
	fn()
	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out)
}

// OPT-20260820-012: access-for-user 审计 site 未解析被跳过时，日志必须带 trace_id，
// 否则换票成功但审计被跳过时 Loki 无法用前端 data-traceId 对上。
func TestWriteTokenUseAuditSkipLogCarriesTraceID(t *testing.T) {
	app := testApp(t)
	// gitlab 配置 website 为空 → AuditSiteForProviderKey 返回 "" → skip 路径。
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"gitlab": {{
			Provider:        "gitlab",
			ServiceProvider: "default",
			ProviderKey:     "gitlab:default",
			Website:         "",
		}},
	}
	out := captureLogOutput(t, func() {
		err := app.writeTokenUseAudit("u1", "gitlab:default", map[string]any{"action": "token_issued"}, "tok", "trace-abc-12345")
		if err == nil {
			t.Error("expected error for unresolved audit site")
		}
	})
	if !strings.Contains(out, "trace-abc-12345") {
		t.Fatalf("skip log missing trace_id, got: %q", out)
	}
	if !strings.Contains(out, "access audit skip") {
		t.Fatalf("expected skip log marker, got: %q", out)
	}
}

// OPT-20260820-012: writeTokenUseAudit 成功路径（site 已解析）返回 nil，不误伤正常审计。
func TestWriteTokenUseAuditResolvedSiteReturnsNil(t *testing.T) {
	app := testApp(t)
	app.Cfg.Providers = map[string][]infrastructure.ProviderConfig{
		"github": {{
			Provider:        "github",
			ServiceProvider: "github-official",
			ProviderKey:     "github:github-official",
			Website:         "https://github.com",
		}},
	}
	if err := app.writeTokenUseAudit("u1", "github:github-official", map[string]any{"action": "token_issued"}, "tok", "trace-abc-67890"); err != nil {
		t.Fatalf("resolved site audit should succeed: %v", err)
	}
}
