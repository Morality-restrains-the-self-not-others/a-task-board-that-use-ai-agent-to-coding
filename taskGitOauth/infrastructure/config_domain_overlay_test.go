package infrastructure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildDomainMapOverlaysConfLocal(t *testing.T) {
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
	dm := buildDomainMap(dir)
	if dm["baseDomain"] != "from-conf-local.test" {
		t.Fatalf("baseDomain=%q, want conf-local overlay", dm["baseDomain"])
	}
	if dm["subdomains.gateway"] != "https://api.from-conf-local.test" {
		t.Fatalf("gateway=%q, want overlayed baseDomain in subdomain template", dm["subdomains.gateway"])
	}
}
