package infrastructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommittedVendorDocsConfigHasNoSecrets(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "conf", "ai", "ai-provider", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if !strings.Contains(text, "backend: local") {
		t.Fatal("committed config.yaml must default vendorDocs.backend=local")
	}
	if !strings.Contains(text, "ai-provider-1259712831") {
		t.Fatal("committed bucket missing")
	}
	if strings.Contains(text, "AKID") {
		t.Fatal("committed config.yaml must not contain COS secret id")
	}
	if !strings.Contains(text, "${subdomains.provider}") {
		t.Fatal("committed corsAllowedOrigins must template provider SPA origin")
	}
}

func TestWriteVendorDocsPathFragmentRoundtrip(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "conf", "ai", "ai-provider")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := WriteVendorDocsPathFragment(root, "vendor-docs", "{keyPrefix}/{userId}/{kind}_{id}{ext}"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, vendorDocsPathFragment))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "vendor-docs") || !strings.Contains(string(raw), "pathRule") {
		t.Fatalf("fragment=%s", raw)
	}
	if err := WriteVendorDocsPathFragment(root, "vendor-docs", "{evil}"); err == nil {
		t.Fatal("expected invalid rule")
	}
}
