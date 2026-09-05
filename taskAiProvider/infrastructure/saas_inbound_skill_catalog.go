package infrastructure

import (
	"fmt"

	"gopkg.in/yaml.v3"
	"taskAiProvider/domain"
)

const SaasInboundSkillVersionsRel = "docs/skills/saas-container/versions.yaml"

type saasInboundYAML struct {
	Current  string `yaml:"current"`
	Versions []struct {
		Version  string `yaml:"version"`
		Status   string `yaml:"status"`
		Released string `yaml:"released"`
		Summary  string `yaml:"summary"`
		Doc      string `yaml:"doc"`
	} `yaml:"versions"`
}

func ParseSaasInboundSkillCatalog(raw []byte) ([]domain.SkillVersionEntry, string, error) {
	var doc saasInboundYAML
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, "", fmt.Errorf("parse skill versions: %w", err)
	}
	out := make([]domain.SkillVersionEntry, 0, len(doc.Versions))
	for _, v := range doc.Versions {
		out = append(out, domain.SkillVersionEntry{
			Version:  v.Version,
			Status:   v.Status,
			Released: v.Released,
			Summary:  v.Summary,
			Doc:      v.Doc,
		})
	}
	if len(out) == 0 {
		return nil, "", fmt.Errorf("skill versions catalog is empty")
	}
	current := domain.NormalizeSkillVersion(doc.Current)
	if current == "" {
		current = domain.CatalogCurrentVersion(out)
	}
	return out, current, nil
}

func LoadSaasInboundSkillCatalog(repoRoot string) ([]domain.SkillVersionEntry, string, error) {
	raw, err := ReadSaasInboundSkillCatalogYAML(repoRoot)
	if err != nil {
		return nil, "", err
	}
	return ParseSaasInboundSkillCatalog(raw)
}
