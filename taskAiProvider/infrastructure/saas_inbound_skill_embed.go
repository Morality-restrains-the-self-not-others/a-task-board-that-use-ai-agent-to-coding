package infrastructure

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Runtime copy of docs/skills/saas-container/* — clone-run / daydaymoney-deploy
// has no docs submodule, so the process must not depend on that tree.
//
//go:embed saascontainer/versions.yaml saascontainer/saas-machine-container.md
var saasInboundSkillEmbed embed.FS

const saasInboundSkillEmbedDir = "saascontainer"

func ReadSaasInboundSkillCatalogYAML(repoRoot string) ([]byte, error) {
	if raw, err := readDiskSkillFile(repoRoot, "versions.yaml"); err == nil {
		return raw, nil
	}
	raw, err := saasInboundSkillEmbed.ReadFile(saasInboundSkillEmbedDir + "/versions.yaml")
	if err != nil {
		return nil, fmt.Errorf("skill versions catalog unavailable: %w", err)
	}
	return raw, nil
}

func ReadSaasInboundSkillDoc(repoRoot, name string) ([]byte, error) {
	base, err := sanitizeSkillDocName(name)
	if err != nil {
		return nil, err
	}
	if raw, err := readDiskSkillFile(repoRoot, base); err == nil {
		return raw, nil
	}
	raw, err := saasInboundSkillEmbed.ReadFile(saasInboundSkillEmbedDir + "/" + base)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func sanitizeSkillDocName(name string) (string, error) {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "saas-machine-container.md"
	}
	if strings.Contains(base, "..") || strings.ContainsAny(base, `/\`) {
		return "", fmt.Errorf("invalid skill doc name")
	}
	return base, nil
}

func readDiskSkillFile(repoRoot, name string) ([]byte, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		found, err := FindMonorepoRoot()
		if err != nil {
			return nil, err
		}
		root = found
	}
	p := filepath.Join(root, "docs/skills/saas-container", name)
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		if err == nil {
			err = os.ErrNotExist
		}
		return nil, err
	}
	return os.ReadFile(p)
}
