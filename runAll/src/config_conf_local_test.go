package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadConfig_MergesConfLocalEnv(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	localDir := filepath.Join(root, "conf-local")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tracked := []byte(`
version: "1"
groups:
  - name: app
    services:
      - name: svc
        command: echo
        health_check:
          url: "http://127.0.0.1:9"
        env:
          TASK2APP_SSO_JWT_SECRET: ""
`)
	overlay := []byte(`
groups:
  - services:
      - env:
          TASK2APP_SSO_JWT_SECRET: from-conf-local
`)
	if err := os.WriteFile(filepath.Join(confDir, "runAll.yaml"), tracked, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "runAll.yaml"), overlay, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(filepath.Join(confDir, "runAll.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	got := cfg.Groups[0].Services[0].Env["TASK2APP_SSO_JWT_SECRET"]
	if got != "from-conf-local" {
		t.Fatalf("env overlay = %q", got)
	}
}

// TestLoadConfig_RejectsConfLocalExtendingServices guards against a stale
// positional conf-local overlay injecting service entries beyond the base
// `groups[].services` list (nightly 2026-09-05: a group inserted mid-file
// shifted the overlay indices, appending nameless services to the wrong group).
func TestLoadConfig_RejectsConfLocalExtendingServices(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	localDir := filepath.Join(root, "conf-local")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tracked := []byte(`
version: "1"
groups:
  - name: app
    services:
      - name: svc
        command: echo
        health_check:
          url: "http://127.0.0.1:9"
`)
	// Overlay written against an older base that had one more service in the group.
	overlay := []byte(`
groups:
  - services:
      - env:
          TASK2APP_SSO_JWT_SECRET: from-conf-local
      - {}
`)
	if err := os.WriteFile(filepath.Join(confDir, "runAll.yaml"), tracked, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "runAll.yaml"), overlay, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(filepath.Join(confDir, "runAll.yaml"))
	if err == nil {
		t.Fatal("LoadConfig succeeded, want conf-local drift error")
	}
	for _, want := range []string{"out of sync", "app", "adds 1 service"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("LoadConfig error %q missing %q", err.Error(), want)
		}
	}
}

// TestLoadConfig_RejectsConfLocalExtendingGroups guards the same drift when the
// overlay grows the top-level `groups` list beyond the base config.
func TestLoadConfig_RejectsConfLocalExtendingGroups(t *testing.T) {
	root := t.TempDir()
	confDir := filepath.Join(root, "conf")
	localDir := filepath.Join(root, "conf-local")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tracked := []byte(`
version: "1"
groups:
  - name: app
    services:
      - name: svc
        command: echo
        health_check:
          url: "http://127.0.0.1:9"
`)
	overlay := []byte(`
groups:
  - {}
  - {}
`)
	if err := os.WriteFile(filepath.Join(confDir, "runAll.yaml"), tracked, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "runAll.yaml"), overlay, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(filepath.Join(confDir, "runAll.yaml"))
	if err == nil {
		t.Fatal("LoadConfig succeeded, want conf-local drift error")
	}
	if !strings.Contains(err.Error(), "out of sync") {
		t.Fatalf("LoadConfig error %q missing drift hint", err.Error())
	}
}

func TestOverlayConfLocalRelNestedAppConfig(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "conf", "auth", "task-auth")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "conf-local", "auth", "task-auth"), 0o755); err != nil {
		t.Fatal(err)
	}
	tracked := []byte("host: 0.0.0.0\nport: 8003\ninternalSecret: \"\"\n")
	if err := os.WriteFile(filepath.Join(appDir, "config.yaml"), tracked, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "conf-local", "auth", "task-auth", "config.yaml"), []byte("internalSecret: from-conf-local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	merged, err := overlayConfLocalRel(root, "auth/task-auth/config.yaml", tracked)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Host           string `yaml:"host"`
		Port           int    `yaml:"port"`
		InternalSecret string `yaml:"internalSecret"`
	}
	if err := yaml.Unmarshal(merged, &out); err != nil {
		t.Fatal(err)
	}
	if out.Host != "0.0.0.0" || out.Port != 8003 {
		t.Fatalf("base fields lost: %+v", out)
	}
	if out.InternalSecret != "from-conf-local" {
		t.Fatalf("internalSecret=%q, want conf-local overlay", out.InternalSecret)
	}
}
