package confload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadAppConfigMergesConfLocal(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "conf", "billing", "paypal")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("mode: sandbox\nclient_secret: \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	localDir := filepath.Join(dir, "conf-local", "billing", "paypal")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "config.yaml"), []byte("client_secret: from-conf-local\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Mode         string `yaml:"mode"`
		ClientSecret string `yaml:"client_secret"`
	}
	if err := ReadAppConfig(dir, "billing/paypal", &cfg); err != nil {
		t.Fatalf("ReadAppConfig: %v", err)
	}
	if cfg.Mode != "sandbox" {
		t.Fatalf("mode=%q, want sandbox from conf", cfg.Mode)
	}
	if cfg.ClientSecret != "from-conf-local" {
		t.Fatalf("client_secret=%q, want conf-local overlay", cfg.ClientSecret)
	}
}

func TestReadAppFragmentMergesConfLocal(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "conf", "auth", "task-auth")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("host: 0.0.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "sms.yaml"), []byte("sms:\n  aliyun:\n    access_key_id: \"\"\n    sign_name: public-sign\n"), 0644); err != nil {
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
			Aliyun *struct {
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
		t.Fatalf("access_key_id=%q, want conf-local overlay", wrap.SMS.Aliyun.AccessKeyID)
	}
	if wrap.SMS.Aliyun.SignName != "public-sign" {
		t.Fatalf("sign_name=%q, want committed fragment", wrap.SMS.Aliyun.SignName)
	}
}

func TestReadAppConfigMergesListSecretsByIndex(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "conf", "auth", "task-auth")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("oidc:\n  bootstrapClients:\n    - clientId: a\n      clientSecret: \"\"\n    - clientId: b\n      clientSecret: \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	localDir := filepath.Join(dir, "conf-local", "auth", "task-auth")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "config.yaml"), []byte("oidc:\n  bootstrapClients:\n    - clientSecret: sa\n    - clientSecret: sb\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		OIDC struct {
			BootstrapClients []struct {
				ClientID     string `yaml:"clientId"`
				ClientSecret string `yaml:"clientSecret"`
			} `yaml:"bootstrapClients"`
		} `yaml:"oidc"`
	}
	if err := ReadAppConfig(dir, "auth/task-auth", &cfg); err != nil {
		t.Fatalf("ReadAppConfig: %v", err)
	}
	if len(cfg.OIDC.BootstrapClients) != 2 {
		t.Fatalf("len=%d", len(cfg.OIDC.BootstrapClients))
	}
	if cfg.OIDC.BootstrapClients[0].ClientID != "a" || cfg.OIDC.BootstrapClients[0].ClientSecret != "sa" {
		t.Fatalf("item0=%+v", cfg.OIDC.BootstrapClients[0])
	}
	if cfg.OIDC.BootstrapClients[1].ClientID != "b" || cfg.OIDC.BootstrapClients[1].ClientSecret != "sb" {
		t.Fatalf("item1=%+v", cfg.OIDC.BootstrapClients[1])
	}
}

func TestReadAppConfigIgnoresConfigLocalYaml(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "conf", "billing", "paypal")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("mode: sandbox\nclient_secret: \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.local.yaml"), []byte("client_secret: \"\"\nrunAllStartEnabled: true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	localDir := filepath.Join(dir, "conf-local", "billing", "paypal")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "config.yaml"), []byte("client_secret: from-conf-local\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Mode               string `yaml:"mode"`
		ClientSecret       string `yaml:"client_secret"`
		RunAllStartEnabled bool   `yaml:"runAllStartEnabled"`
	}
	if err := ReadAppConfig(dir, "billing/paypal", &cfg); err != nil {
		t.Fatalf("ReadAppConfig: %v", err)
	}
	if cfg.ClientSecret != "from-conf-local" {
		t.Fatalf("client_secret=%q, want conf-local (config.local.yaml must not wipe it)", cfg.ClientSecret)
	}
	if cfg.RunAllStartEnabled {
		t.Fatal("runAllStartEnabled from config.local.yaml must be ignored")
	}
}

func TestReadAppConfigMergesDockerInfraConfLocal(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "conf", "events", "domain-events")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("transport: kafka\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "docker-infra.yaml"), []byte("redis:\n  host: tracked-host\n  port: 6379\n"), 0644); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(dir, "conf-local", "events", "domain-events")
	if err := os.MkdirAll(locDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locDir, "docker-infra.yaml"), []byte("redis:\n  host: from-conf-local\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Transport string `yaml:"transport"`
		Redis     struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
		} `yaml:"redis"`
	}
	if err := ReadAppConfig(dir, "events/domain-events", &cfg); err != nil {
		t.Fatalf("ReadAppConfig: %v", err)
	}
	if cfg.Transport != "kafka" {
		t.Fatalf("transport=%q", cfg.Transport)
	}
	if cfg.Redis.Host != "from-conf-local" {
		t.Fatalf("redis.host=%q, want conf-local overlay of docker-infra.yaml", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 6379 {
		t.Fatalf("redis.port=%d, want tracked fragment port preserved", cfg.Redis.Port)
	}
}

func TestUnmarshalYAMLMergedOverlaysConfLocal(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "conf", "billing", "paypal")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf", "base.yaml"), []byte("scheme: https\nbaseDomain: example.test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), []byte("client_id: pub\nclient_secret: \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(dir, "conf-local", "billing", "paypal")
	if err := os.MkdirAll(locDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locDir, "config.yaml"), []byte("client_secret: overlay-secret\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
	}
	if err := UnmarshalYAMLMerged(dir, "billing/paypal/config.yaml", &cfg); err != nil {
		t.Fatalf("UnmarshalYAMLMerged: %v", err)
	}
	if cfg.ClientID != "pub" || cfg.ClientSecret != "overlay-secret" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestMergeYAMLAtPathOverlaysConfLocal(t *testing.T) {
	dir := t.TempDir()
	provDir := filepath.Join(dir, "conf", "auth", "task-credential", "git-oauth-providers")
	if err := os.MkdirAll(provDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf", "base.yaml"), []byte("scheme: https\nbaseDomain: example.test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(provDir, "github.yaml")
	if err := os.WriteFile(path, []byte("provider: github\nclient_secret: \"\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(dir, "conf-local", "auth", "task-credential", "git-oauth-providers")
	if err := os.MkdirAll(locDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locDir, "github.yaml"), []byte("client_secret: from-conf-local\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := MergeYAMLAtPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["provider"] != "github" {
		t.Fatalf("provider=%v", got["provider"])
	}
	if got["client_secret"] != "from-conf-local" {
		t.Fatalf("client_secret=%v, want conf-local overlay", got["client_secret"])
	}
}

func TestResolveBaseYamlMergesConfLocal(t *testing.T) {
	dir := t.TempDir()
	writeTestBaseYaml(t, dir, `
scheme: https
baseDomain: tracked.example
subdomains:
  gateway: ${scheme}://api.${baseDomain}
`)
	if err := os.MkdirAll(filepath.Join(dir, "conf-local"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf-local", "base.yaml"), []byte("baseDomain: from-conf-local.test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	subs := ResolveBaseYaml(dir)
	if got := subs["baseDomain"]; got != "from-conf-local.test" {
		t.Fatalf("baseDomain=%q, want conf-local overlay", got)
	}
	if got := subs["subdomains.gateway"]; got != "https://api.from-conf-local.test" {
		t.Fatalf("gateway=%q, want overlayed baseDomain in subdomain template", got)
	}
}
