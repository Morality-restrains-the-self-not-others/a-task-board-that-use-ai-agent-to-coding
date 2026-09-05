package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindMonorepoRootHonorsCONF_ROOT(t *testing.T) {
	deploy := t.TempDir()
	confDir := filepath.Join(deploy, "conf")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(confDir, "base.yaml"), []byte("scheme: https\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONF_ROOT", confDir)
	t.Setenv("DEPLOY_ROOT", "")
	unrelated := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(unrelated); err != nil {
		t.Fatal(err)
	}
	got, err := findMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != deploy {
		t.Fatalf("findMonorepoRoot=%q want CONF_ROOT deploy %q", got, deploy)
	}
}
