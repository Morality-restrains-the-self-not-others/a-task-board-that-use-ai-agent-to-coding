package config

import (
	"os"
	"path/filepath"
	"testing"
)

func evalPath(t *testing.T, p string) string {
	t.Helper()
	got, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return got
}

// clone-run cwd is envs/current/<svc>; that tree has conf/base.yaml but
// secrets live in DEPLOY_ROOT/conf-local. CONF_ROOT must win over cwd walk.
func TestFindMonorepoRootHonorsCONFRootOverNestedCloneRunConf(t *testing.T) {
	deploy := t.TempDir()
	if err := os.MkdirAll(filepath.Join(deploy, "conf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deploy, "conf", "base.yaml"), []byte("scheme: https\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(deploy, "envs", "current", "taskEvents")
	nestedConf := filepath.Join(deploy, "envs", "current", "conf")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nestedConf, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedConf, "base.yaml"), []byte("scheme: https\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONF_ROOT", filepath.Join(deploy, "conf"))
	t.Setenv("DEPLOY_ROOT", deploy)

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := evalPath(t, deploy)
	got := evalPath(t, root)
	if got != want {
		t.Fatalf("FindMonorepoRoot must be deploy root for conf-local overlay, want %q got %q", want, got)
	}
}

func TestLoadOverlaysConfLocalWhenCONFRootIsDeploy(t *testing.T) {
	deploy := t.TempDir()
	deDir := filepath.Join(deploy, "conf", "events", "domain-events")
	if err := os.MkdirAll(deDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deploy, "conf", "base.yaml"), []byte("scheme: https\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	global := `transport: kafka
internalSecret: ""
kafka:
  bootstrapServers: kafka:9093
redis:
  host: 127.0.0.1
  port: 6379
  streamKeyPrefix: "domain-events:"
`
	if err := os.WriteFile(filepath.Join(deDir, "config.yaml"), []byte(global), 0o644); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(deploy, "conf-local", "events", "domain-events")
	if err := os.MkdirAll(locDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locDir, "config.yaml"), []byte("internalSecret: secret-from-deploy-conf-local\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(deploy, "envs", "current", "taskEvents")
	nestedConf := filepath.Join(deploy, "envs", "current", "conf", "events", "domain-events")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nestedConf, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deploy, "envs", "current", "conf", "base.yaml"), []byte("scheme: https\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedConf, "config.yaml"), []byte(global), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONF_ROOT", filepath.Join(deploy, "conf"))
	t.Setenv("DEPLOY_ROOT", deploy)

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	cfg, root, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if evalPath(t, root) != evalPath(t, deploy) {
		t.Fatalf("Load root=%q want deploy %q", root, deploy)
	}
	if cfg.InternalSecret != "secret-from-deploy-conf-local" {
		t.Fatalf("internalSecret=%q, cwd walk would miss deploy conf-local", cfg.InternalSecret)
	}
}
