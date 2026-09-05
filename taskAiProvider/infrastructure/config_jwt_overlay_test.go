package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePlaceholdersOverlaysConfLocal(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	tracked := "scheme: https\nbaseDomain: tracked.example\nsubdomains:\n  gateway: ${scheme}://api.${baseDomain}\n"
	if err := os.WriteFile(filepath.Join(dir, "conf", "base.yaml"), []byte(tracked), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "conf-local"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "conf-local", "base.yaml"), []byte("baseDomain: from-conf-local.test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := resolvePlaceholders(dir, "${subdomains.gateway}")
	if got != "https://api.from-conf-local.test" {
		t.Fatalf("placeholder=%q, want conf-local baseDomain in subdomain template", got)
	}
}
