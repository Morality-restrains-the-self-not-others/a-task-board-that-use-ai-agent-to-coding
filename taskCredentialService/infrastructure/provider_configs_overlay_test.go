package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProviderYAMLOverlaysConfLocal(t *testing.T) {
	dir := t.TempDir()
	provDir := filepath.Join(dir, "conf", "auth", "task-credential", "git-oauth-providers")
	if err := os.MkdirAll(provDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf", "base.yaml"), []byte("scheme: https\nbaseDomain: example.test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(provDir, "github.yaml")
	if err := os.WriteFile(path, []byte("provider: github\nservice_provider: tracked\ntarget:\n  website: https://github.com\n"), 0644); err != nil {
		t.Fatal(err)
	}
	locDir := filepath.Join(dir, "conf-local", "auth", "task-credential", "git-oauth-providers")
	if err := os.MkdirAll(locDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locDir, "github.yaml"), []byte("service_provider: from-conf-local\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, sp, _ := parseProviderYAML(path)
	if sp != "from-conf-local" {
		t.Fatalf("service_provider=%q, want conf-local overlay", sp)
	}
}
