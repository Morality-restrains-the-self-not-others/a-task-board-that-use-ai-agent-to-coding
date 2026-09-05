package confload

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestBaseYaml(t *testing.T, dir, content string) string {
	t.Helper()
	confDir := filepath.Join(dir, "conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir conf: %v", err)
	}
	path := filepath.Join(confDir, "base.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write base.yaml: %v", err)
	}
	return dir
}

// ── Domain addressing ────────────────────────────────────────────────────

func TestResolveBaseYamlIgnoresLegacyLocal(t *testing.T) {
	dir := t.TempDir()
	writeTestBaseYaml(t, dir, `
scheme: https
baseDomain: example.com
subdomains:
  gateway: ${scheme}://api.${baseDomain}
  gitlab: gitlab.${baseDomain}
  api: api.${baseDomain}
  www: www.${baseDomain}
local:
  gateway: http://10.0.0.1:18081
  gitlab: 10.0.0.1:8012
`)

	subs := ResolveBaseYaml(dir)
	if got := subs["scheme"]; got != "https" {
		t.Errorf("scheme: want https, got %q", got)
	}
	if got := subs["subdomains.gateway"]; got != "https://api.daydaymoney.com" {
		t.Errorf("gateway: want https://api.daydaymoney.com, got %q", got)
	}
	if got := subs["subdomains.gitlab"]; got != "gitlab.daydaymoney.com" {
		t.Errorf("gitlab: want gitlab.daydaymoney.com, got %q", got)
	}
}

func TestResolveBaseYamlBaseDomainEnvDefault(t *testing.T) {
	dir := t.TempDir()
	writeTestBaseYaml(t, dir, `
scheme: ${PUBLIC_SCHEME:-https}
baseDomain: ${BASE_DOMAIN:-daydaymoney.com}
subdomains:
  gateway: ${scheme}://api.${baseDomain}
  gitlab: gitlab.${baseDomain}
`)

	t.Setenv("BASE_DOMAIN", "")
	_ = os.Unsetenv("BASE_DOMAIN")
	_ = os.Unsetenv("PUBLIC_SCHEME")
	subs := ResolveBaseYaml(dir)
	if got := subs["scheme"]; got != "https" {
		t.Errorf("scheme default: want https, got %q", got)
	}
	if got := subs["subdomains.gateway"]; got != "https://api.daydaymoney.com" {
		t.Errorf("gateway default: want https://api.daydaymoney.com, got %q", got)
	}
}

func TestResolveBaseYamlSchemeEnvOverride(t *testing.T) {
	dir := t.TempDir()
	writeTestBaseYaml(t, dir, `
scheme: ${PUBLIC_SCHEME:-https}
baseDomain: example.com
subdomains:
  gateway: ${scheme}://api.${baseDomain}
`)

	t.Setenv("PUBLIC_SCHEME", "http")
	subs := ResolveBaseYaml(dir)
	if got := subs["scheme"]; got != "http" {
		t.Errorf("scheme: want http, got %q", got)
	}
	if got := subs["subdomains.gateway"]; got != "http://api.daydaymoney.com" {
		t.Errorf("gateway: want http://api.daydaymoney.com, got %q", got)
	}
}

// ── Domain mode ──────────────────────────────────────────────────────────

func TestResolveBaseYamlDomainMode(t *testing.T) {
	dir := t.TempDir()
	writeTestBaseYaml(t, dir, `
baseDomain: example.com
subdomains:
  gateway: https://api.${baseDomain}
  gitlab: gitlab.daydaymoney.com
  api: api.daydaymoney.com
  www: www.daydaymoney.com
`)

	subs := ResolveBaseYaml(dir)
	if len(subs) == 0 {
		t.Fatal("expected non-empty map")
	}

	tests := []struct{ key, want string }{
		{"subdomains.gateway", "https://api.daydaymoney.com"},
		{"subdomains.gitlab", "gitlab.daydaymoney.com"},
		{"subdomains.api", "api.daydaymoney.com"},
		{"subdomains.www", "www.daydaymoney.com"},
		{"baseDomain", "example.com"},
	}
	for _, tc := range tests {
		if got := subs[tc.key]; got != tc.want {
			t.Errorf("%s: want %q, got %q", tc.key, tc.want, got)
		}
	}
}

func TestResolveBaseYamlDomainWithEnvBaseDomain(t *testing.T) {
	dir := t.TempDir()
	writeTestBaseYaml(t, dir, `
baseDomain: ${BASE_DOMAIN:-default.com}
subdomains:
  gateway: https://api.${baseDomain}
  gitlab: gitlab.${baseDomain}
`)

	t.Setenv("BASE_DOMAIN", "custom.io")
	subs := ResolveBaseYaml(dir)
	if got := subs["subdomains.gateway"]; got != "https://api.custom.io" {
		t.Errorf("gateway: want https://api.custom.io, got %q", got)
	}
	if got := subs["subdomains.gitlab"]; got != "gitlab.custom.io" {
		t.Errorf("gitlab: want gitlab.custom.io, got %q", got)
	}
}

// ── ResolveTemplate ──────────────────────────────────────────────────────

func TestResolveTemplate(t *testing.T) {
	subs := map[string]string{
		"subdomains.gateway": "http://10.0.0.1:18081",
		"subdomains.gitlab":  "gitlab.daydaymoney.com",
		"baseDomain":         "example.com",
	}

	tests := []struct{ input, want string }{
		{"${subdomains.gateway}", "http://10.0.0.1:18081"},
		{"http://${subdomains.gitlab}/callback", "http://gitlab.daydaymoney.com/callback"},
		{"${subdomains.gateway}/api/oidc/authorize", "http://10.0.0.1:18081/api/oidc/authorize"},
		{"no template here", "no template here"},
		{"", ""},
		{"${unknown.key}", "${unknown.key}"}, // unresolved patterns left as-is
	}

	for _, tc := range tests {
		got := ResolveTemplate(tc.input, subs)
		if got != tc.want {
			t.Errorf("ResolveTemplate(%q): want %q, got %q", tc.input, tc.want, got)
		}
	}
}

func TestResolveTemplateEmptySubs(t *testing.T) {
	if got := ResolveTemplate("${subdomains.gateway}", nil); got != "${subdomains.gateway}" {
		t.Errorf("nil subs: want unchanged, got %q", got)
	}
	if got := ResolveTemplate("${subdomains.gateway}", map[string]string{}); got != "${subdomains.gateway}" {
		t.Errorf("empty subs: want unchanged, got %q", got)
	}
}

// ── ReadAppConfigResolved ────────────────────────────────────────────────

func TestReadAppConfigResolved(t *testing.T) {
	dir := t.TempDir()

	// Set up base.yaml
	writeTestBaseYaml(t, dir, `
baseDomain: example.com
subdomains:
  gateway: https://api.${baseDomain}
  gitlab: gitlab.${baseDomain}
`)

	// Set up a config.yaml
	confDir := filepath.Join(dir, "conf", "myapp")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configYAML := `
publicBase: ${subdomains.gateway}
name: myapp
`
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	var dest struct {
		PublicBase string `yaml:"publicBase"`
		Name       string `yaml:"name"`
	}
	if err := ReadAppConfigResolved(dir, "myapp", &dest); err != nil {
		t.Fatalf("ReadAppConfigResolved: %v", err)
	}

	if dest.PublicBase != "https://api.daydaymoney.com" {
		t.Errorf("PublicBase: want https://api.daydaymoney.com, got %q", dest.PublicBase)
	}
	if dest.Name != "myapp" {
		t.Errorf("Name: want myapp, got %q", dest.Name)
	}
}

func TestReadAppConfigResolvedNestedStruct(t *testing.T) {
	dir := t.TempDir()

	writeTestBaseYaml(t, dir, `
baseDomain: example.com
subdomains:
  gateway: https://api.${baseDomain}
  gitlab: gitlab.${baseDomain}
`)

	confDir := filepath.Join(dir, "conf", "myapp")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configYAML := `
oidc:
  issuer: ${subdomains.gateway}
  redirectUri: ${scheme}://${subdomains.gitlab}/callback
`
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	var dest struct {
		Oidc *struct {
			Issuer      string `yaml:"issuer"`
			RedirectURI string `yaml:"redirectUri"`
		} `yaml:"oidc"`
	}
	if err := ReadAppConfigResolved(dir, "myapp", &dest); err != nil {
		t.Fatalf("ReadAppConfigResolved: %v", err)
	}

	if dest.Oidc == nil {
		t.Fatal("Oidc is nil")
	}
	if dest.Oidc.Issuer != "https://api.daydaymoney.com" {
		t.Errorf("Issuer: want https://api.daydaymoney.com, got %q", dest.Oidc.Issuer)
	}
	if dest.Oidc.RedirectURI != "https://gitlab.daydaymoney.com/callback" {
		t.Errorf("RedirectURI: want https://gitlab.daydaymoney.com/callback, got %q", dest.Oidc.RedirectURI)
	}
}

func TestReadAppConfigResolvedSliceOfStructs(t *testing.T) {
	dir := t.TempDir()

	writeTestBaseYaml(t, dir, `
baseDomain: example.com
subdomains:
  gateway: https://api.${baseDomain}
  gitlab: gitlab.${baseDomain}
`)

	confDir := filepath.Join(dir, "conf", "myapp")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configYAML := `
clients:
  - id: client1
    url: ${subdomains.gateway}
  - id: client2
    url: http://${subdomains.gitlab}/cb
`
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	var dest struct {
		Clients []struct {
			ID  string `yaml:"id"`
			URL string `yaml:"url"`
		} `yaml:"clients"`
	}
	if err := ReadAppConfigResolved(dir, "myapp", &dest); err != nil {
		t.Fatalf("ReadAppConfigResolved: %v", err)
	}

	if len(dest.Clients) != 2 {
		t.Fatalf("want 2 clients, got %d", len(dest.Clients))
	}
	if dest.Clients[0].URL != "https://api.daydaymoney.com" {
		t.Errorf("client0: want https://api.daydaymoney.com, got %q", dest.Clients[0].URL)
	}
	if dest.Clients[1].URL != "http://gitlab.daydaymoney.com/cb" {
		t.Errorf("client1: want http://gitlab.daydaymoney.com/cb, got %q", dest.Clients[1].URL)
	}
}

func TestReadAppConfigResolvedStringSlice(t *testing.T) {
	dir := t.TempDir()

	writeTestBaseYaml(t, dir, `
scheme: https
baseDomain: example.com
subdomains:
  www: www.${baseDomain}
  gateway: ${scheme}://api.${baseDomain}
`)

	confDir := filepath.Join(dir, "conf", "task-auth")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configYAML := `
oidc:
  issuer: ${subdomains.gateway}
  postLogoutRedirectOrigins:
    - ${scheme}://${subdomains.www}
    - ${scheme}://${baseDomain}
    - http://127.0.0.1:4000
`
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	var dest struct {
		Oidc *struct {
			Issuer                    string   `yaml:"issuer"`
			PostLogoutRedirectOrigins []string `yaml:"postLogoutRedirectOrigins"`
		} `yaml:"oidc"`
	}
	if err := ReadAppConfigResolved(dir, "task-auth", &dest); err != nil {
		t.Fatalf("ReadAppConfigResolved: %v", err)
	}
	if dest.Oidc == nil {
		t.Fatal("Oidc is nil")
	}
	if dest.Oidc.Issuer != "https://api.daydaymoney.com" {
		t.Errorf("Issuer: want https://api.daydaymoney.com, got %q", dest.Oidc.Issuer)
	}
	wantOrigins := []string{
		"https://www.daydaymoney.com",
		"https://example.com",
		"http://127.0.0.1:4000",
	}
	if len(dest.Oidc.PostLogoutRedirectOrigins) != len(wantOrigins) {
		t.Fatalf("origins len: want %d, got %d (%q)", len(wantOrigins), len(dest.Oidc.PostLogoutRedirectOrigins), dest.Oidc.PostLogoutRedirectOrigins)
	}
	for i, want := range wantOrigins {
		if got := dest.Oidc.PostLogoutRedirectOrigins[i]; got != want {
			t.Errorf("origins[%d]: want %q, got %q", i, want, got)
		}
	}
}

// TestReadAppConfigResolvedMapOfStructs mirrors the wechat.apps.<key>.redirectUri
// config shape (map[string]struct) — regression test for the bug where map-backed
// config blocks never had their ${subdomains.xxx} templates resolved.
func TestReadAppConfigResolvedMapOfStructs(t *testing.T) {
	dir := t.TempDir()

	writeTestBaseYaml(t, dir, `
scheme: https
baseDomain: example.com
subdomains:
  base: ${baseDomain}
  gateway: ${scheme}://api.${baseDomain}
`)

	confDir := filepath.Join(dir, "conf", "auth", "task-auth")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configYAML := `
wechat:
  apps:
    web:
      appId: wx123
      redirectUri: "${scheme}://${subdomains.base}/api/auth/wechat/callback/"
    inapp:
      redirectUri: ${subdomains.gateway}/cb
`
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	var dest struct {
		WeChat *struct {
			Apps map[string]struct {
				AppID       string `yaml:"appId"`
				RedirectURI string `yaml:"redirectUri"`
			} `yaml:"apps"`
		} `yaml:"wechat"`
	}
	if err := ReadAppConfigResolved(dir, "auth/task-auth", &dest); err != nil {
		t.Fatalf("ReadAppConfigResolved: %v", err)
	}
	if dest.WeChat == nil {
		t.Fatal("WeChat is nil")
	}
	web, ok := dest.WeChat.Apps["web"]
	if !ok {
		t.Fatalf("missing app web: %+v", dest.WeChat.Apps)
	}
	if web.RedirectURI != "https://example.com/api/auth/wechat/callback/" {
		t.Errorf("web.redirectUri: want https://example.com/api/auth/wechat/callback/, got %q", web.RedirectURI)
	}
	if web.AppID != "wx123" {
		t.Errorf("web.appId: want wx123, got %q", web.AppID)
	}
	if inapp, ok := dest.WeChat.Apps["inapp"]; !ok || inapp.RedirectURI != "https://api.daydaymoney.com/cb" {
		t.Errorf("inapp.redirectUri: want https://api.daydaymoney.com/cb, got %+v", dest.WeChat.Apps["inapp"])
	}
}

func TestReadAppConfigResolvedMapOfStrings(t *testing.T) {
	dir := t.TempDir()

	writeTestBaseYaml(t, dir, `
scheme: https
baseDomain: example.com
subdomains:
  base: ${baseDomain}
`)

	confDir := filepath.Join(dir, "conf", "myapp")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configYAML := `
headers:
  origin: ${scheme}://${subdomains.base}
  keep: plain-value
`
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	var dest struct {
		Headers map[string]string `yaml:"headers"`
	}
	if err := ReadAppConfigResolved(dir, "myapp", &dest); err != nil {
		t.Fatalf("ReadAppConfigResolved: %v", err)
	}
	if got := dest.Headers["origin"]; got != "https://example.com" {
		t.Errorf("headers.origin: want https://example.com, got %q", got)
	}
	if got := dest.Headers["keep"]; got != "plain-value" {
		t.Errorf("headers.keep: want plain-value, got %q", got)
	}
}

func TestReadAppConfigResolvedNoBaseYaml(t *testing.T) {
	dir := t.TempDir()

	confDir := filepath.Join(dir, "conf", "myapp")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configYAML := `name: testapp`
	if err := os.WriteFile(filepath.Join(confDir, "config.yaml"), []byte(configYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	var dest struct {
		Name string `yaml:"name"`
	}
	// Should NOT crash when base.yaml is missing — just skip resolution.
	if err := ReadAppConfigResolved(dir, "myapp", &dest); err != nil {
		t.Fatalf("ReadAppConfigResolved: %v", err)
	}
	if dest.Name != "testapp" {
		t.Errorf("Name: want testapp, got %q", dest.Name)
	}
}

// ── resolveEnvVars ───────────────────────────────────────────────────────

func TestResolveEnvVars(t *testing.T) {
	t.Setenv("MY_VAR", "hello")
	t.Setenv("OTHER_VAR", "world")

	tests := []struct{ input, want string }{
		{"${MY_VAR:-default}", "hello"},
		{"${OTHER_VAR:-fallback}", "world"},
		{"${UNSET:-the-default}", "the-default"},
		{"${UNSET}", ""},
		{"plain text", "plain text"},
		{"prefix_${MY_VAR:-x}_suffix", "prefix_hello_suffix"},
	}

	for _, tc := range tests {
		got := resolveEnvVars(tc.input)
		if got != tc.want {
			t.Errorf("resolveEnvVars(%q): want %q, got %q", tc.input, tc.want, got)
		}
	}
}

// ── Edge cases ───────────────────────────────────────────────────────────

func TestResolveBaseYamlMissingFile(t *testing.T) {
	dir := t.TempDir()
	subs := ResolveBaseYaml(dir)
	if len(subs) != 0 {
		t.Errorf("expected empty map for missing base.yaml, got %d entries", len(subs))
	}
}

func TestResolveTemplateUnresolvedLeftAsIs(t *testing.T) {
	// Patterns that don't match any key should be left unchanged.
	subs := map[string]string{"subdomains.gateway": "http://gw"}
	input := "${subdomains.gateway}/api and ${subdomains.unknown}/x"
	want := "http://gw/api and ${subdomains.unknown}/x"
	if got := ResolveTemplate(input, subs); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestResolveBaseYamlIgnoresDeployModeEnv(t *testing.T) {
	dir := t.TempDir()
	writeTestBaseYaml(t, dir, `
baseDomain: ${BASE_DOMAIN:-override.io}
subdomains:
  gateway: https://api.${baseDomain}
local:
  gateway: http://127.0.0.1:18081
`)

	t.Setenv("DEPLOY_MODE", "local")
	subs := ResolveBaseYaml(dir)
	if got := subs["subdomains.gateway"]; got != "https://api.override.io" {
		t.Errorf("domain addressing must ignore DEPLOY_MODE=local: want https://api.override.io, got %q", got)
	}
}

// Test that strings NOT containing ${} are unchanged by ResolveTemplate.
func TestResolveTemplateNoPlaceholders(t *testing.T) {
	subs := map[string]string{"subdomains.gateway": "http://gw"}
	input := "gitlab-git-service"
	if got := ResolveTemplate(input, subs); got != input {
		t.Errorf("no-placeholder string changed: %q → %q", input, got)
	}
}

// Test that _base key is NOT used as a template replacement.
func TestResolveTemplateSkipsBaseKey(t *testing.T) {
	subs := map[string]string{"_base": "10.0.0.1", "subdomains.gw": "http://gw"}
	// If _base were incorrectly applied, it would replace "${_base}" in the string.
	if got := ResolveTemplate("${_base}", subs); got != "${_base}" {
		t.Errorf("_base should be skipped, got %q", got)
	}
	// But subdomains.gw should still be resolved.
	if got := ResolveTemplate("${subdomains.gw}", subs); got != "http://gw" {
		t.Errorf("subdomains.gw should resolve, got %q", got)
	}
}

func TestReadAppFragmentOwnDirOnly(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "conf", "auth", "task-auth")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("host: 0.0.0.0\nport: 8003\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "sms.yaml"), []byte("sms:\n  provider: aliyun\n  aliyun:\n    sign_name: test-sign\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var wrap struct {
		SMS *struct {
			Provider string `yaml:"provider"`
			Aliyun   *struct {
				SignName string `yaml:"sign_name"`
			} `yaml:"aliyun"`
		} `yaml:"sms"`
	}
	if err := ReadAppFragment(dir, "auth/task-auth", "sms.yaml", &wrap); err != nil {
		t.Fatalf("ReadAppFragment: %v", err)
	}
	if wrap.SMS == nil || wrap.SMS.Provider != "aliyun" {
		t.Fatalf("sms=%#v", wrap.SMS)
	}
	if wrap.SMS.Aliyun == nil || wrap.SMS.Aliyun.SignName != "test-sign" {
		t.Fatalf("aliyun=%#v", wrap.SMS.Aliyun)
	}
	if err := ReadAppFragment(dir, "auth/task-auth", "../django/config.yaml", &wrap); err == nil {
		t.Fatal("expected path traversal filename to fail")
	}
}

func TestReadAppFragmentIgnoresSiblingLocalYaml(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "conf", "auth", "task-auth")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("host: 0.0.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "sms.yaml"), []byte("sms:\n  provider: aliyun\n  aliyun:\n    access_key_id: \"\"\n    sign_name: public-sign\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "sms.local.yaml"), []byte("sms:\n  aliyun:\n    access_key_id: stale-local-key\n"), 0644); err != nil {
		t.Fatal(err)
	}
	localDir := filepath.Join(dir, "conf-local", "auth", "task-auth")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "sms.yaml"), []byte("sms:\n  aliyun:\n    access_key_id: conf-local-id\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var wrap struct {
		SMS *struct {
			Provider string `yaml:"provider"`
			Aliyun   *struct {
				AccessKeyID string `yaml:"access_key_id"`
				SignName    string `yaml:"sign_name"`
			} `yaml:"aliyun"`
		} `yaml:"sms"`
	}
	if err := ReadAppFragment(dir, "auth/task-auth", "sms.yaml", &wrap); err != nil {
		t.Fatalf("ReadAppFragment: %v", err)
	}
	if wrap.SMS == nil || wrap.SMS.Aliyun == nil {
		t.Fatalf("sms=%#v", wrap.SMS)
	}
	if wrap.SMS.Aliyun.AccessKeyID != "conf-local-id" {
		t.Fatalf("access_key_id=%q, want conf-local (sms.local.yaml must not win)", wrap.SMS.Aliyun.AccessKeyID)
	}
	if wrap.SMS.Aliyun.SignName != "public-sign" {
		t.Fatalf("sign_name=%q, want value from committed fragment", wrap.SMS.Aliyun.SignName)
	}
}

// ── FindMonorepoRoot ──────────────────────────────────────────────────────

func TestFindMonorepoRootFromCwd(t *testing.T) {
	unsetDeliveryRootEnv(t)
	dir := t.TempDir()

	// OPT-20260806-057: marker 从 conf/core/django/config.yaml 改为 conf/base.yaml
	markerDir := filepath.Join(dir, "conf")
	if err := os.MkdirAll(markerDir, 0755); err != nil {
		t.Fatalf("mkdir marker: %v", err)
	}
	if err := os.WriteFile(filepath.Join(markerDir, "base.yaml"), []byte("scheme: https\nbaseDomain: example.com\n"), 0644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	// Change into a subdirectory to test upward traversal.
	subDir := filepath.Join(dir, "taskCloudService", "bin")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	origWd, _ := os.Getwd()
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWd)

	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatalf("FindMonorepoRoot: %v", err)
	}
	if root != dir {
		t.Errorf("want %q, got %q", dir, root)
	}
}

func TestFindMonorepoRootViaDataMigrateMarker(t *testing.T) {
	unsetDeliveryRootEnv(t)
	dir := t.TempDir()

	// Only dataMigrate/ directory exists (no conf/).
	if err := os.MkdirAll(filepath.Join(dir, "dataMigrate"), 0755); err != nil {
		t.Fatalf("mkdir dataMigrate: %v", err)
	}

	origWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWd)

	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatalf("FindMonorepoRoot with dataMigrate marker: %v", err)
	}
	if root != dir {
		t.Errorf("want %q, got %q", dir, root)
	}
}

func TestFindMonorepoRootViaGitmodulesMarker(t *testing.T) {
	unsetDeliveryRootEnv(t)
	dir := t.TempDir()

	// Only .gitmodules exists.
	if err := os.WriteFile(filepath.Join(dir, ".gitmodules"), []byte{}, 0644); err != nil {
		t.Fatalf("write .gitmodules: %v", err)
	}

	origWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWd)

	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatalf("FindMonorepoRoot with .gitmodules marker: %v", err)
	}
	if root != dir {
		t.Errorf("want %q, got %q", dir, root)
	}
}

func TestFindMonorepoRootNoMarker(t *testing.T) {
	unsetDeliveryRootEnv(t)
	dir := t.TempDir()

	origWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origWd)

	_, err := FindMonorepoRoot()
	if err == nil {
		t.Fatal("expected error when no markers exist")
	}
}

// ── Benchmark ────────────────────────────────────────────────────────────

func BenchmarkResolveTemplate(b *testing.B) {
	subs := map[string]string{
		"subdomains.gateway": "http://127.0.0.1:18081",
		"subdomains.gitlab":  "127.0.0.1:8012",
		"subdomains.api":     "127.0.0.1:8001",
		"subdomains.www":     "127.0.0.1:4000",
		"baseDomain":         "daydaymoney.com",
	}
	input := "${subdomains.gateway}/api/oidc/authorize?redirect_uri=http://${subdomains.gitlab}/callback"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ResolveTemplate(input, subs)
	}
}
