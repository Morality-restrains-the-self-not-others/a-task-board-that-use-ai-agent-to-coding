package infrastructure

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"taskAiProvider/domain"
)

func TestParseSaasInboundSkillCatalogCurrentV1(t *testing.T) {
	raw := []byte("current: \"1\"\nversions:\n  - version: \"1\"\n    status: current\n    released: \"2026-08-16\"\n    summary: hello\n    doc: saas-machine-container.md\n")
	entries, current, err := ParseSaasInboundSkillCatalog(raw)
	if err != nil {
		t.Fatal(err)
	}
	if current != "1" || len(entries) != 1 || entries[0].Doc != "saas-machine-container.md" {
		t.Fatalf("current=%s entries=%+v", current, entries)
	}
}

func TestLoadSaasInboundSkillCatalogFromRepo(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Skip(err)
	}
	p := filepath.Join(root, SaasInboundSkillVersionsRel)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("catalog missing: %v", err)
	}
	entries, current, err := LoadSaasInboundSkillCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	if current != "1" {
		t.Fatalf("current=%s", current)
	}
	if err := domain.ValidateSaasInboundSkillVersion("1", entries); err != nil {
		t.Fatal(err)
	}
}

// Clone-run / daydaymoney-deploy has db/registry.yaml but no docs/ submodule.
// Catalog must still load (runtime embed), otherwise vendor modal 500s.
func TestLoadSaasInboundSkillCatalogFallsBackToEmbed(t *testing.T) {
	entries, current, err := LoadSaasInboundSkillCatalog(t.TempDir())
	if err != nil {
		t.Fatalf("embed fallback: %v", err)
	}
	if current != "1" {
		t.Fatalf("current=%s", current)
	}
	if err := domain.ValidateSaasInboundSkillVersion("1", entries); err != nil {
		t.Fatal(err)
	}
	raw, err := ReadSaasInboundSkillDoc(t.TempDir(), "saas-machine-container.md")
	if err != nil {
		t.Fatalf("embed skill md: %v", err)
	}
	if !bytes.Contains(raw, []byte("SaaS Machine Container Skill")) {
		t.Fatalf("embedded skill md missing title: %s", truncate(raw, 120))
	}
}

func TestEmbeddedCatalogMatchesDocsSSOT(t *testing.T) {
	root, err := FindMonorepoRoot()
	if err != nil {
		t.Skip(err)
	}
	diskYAML, err := os.ReadFile(filepath.Join(root, SaasInboundSkillVersionsRel))
	if err != nil {
		t.Skip(err)
	}
	embYAML, err := saasInboundSkillEmbed.ReadFile("saascontainer/versions.yaml")
	if err != nil {
		t.Fatalf("embed yaml: %v", err)
	}
	if string(bytes.TrimSpace(diskYAML)) != string(bytes.TrimSpace(embYAML)) {
		t.Fatal("infrastructure/saascontainer/versions.yaml drifted from docs/skills/saas-container/versions.yaml")
	}
	diskMD, err := os.ReadFile(filepath.Join(root, "docs/skills/saas-container/saas-machine-container.md"))
	if err != nil {
		t.Skip(err)
	}
	embMD, err := saasInboundSkillEmbed.ReadFile("saascontainer/saas-machine-container.md")
	if err != nil {
		t.Fatalf("embed md: %v", err)
	}
	if string(bytes.TrimSpace(diskMD)) != string(bytes.TrimSpace(embMD)) {
		t.Fatal("infrastructure/saascontainer/saas-machine-container.md drifted from docs SSOT")
	}
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
