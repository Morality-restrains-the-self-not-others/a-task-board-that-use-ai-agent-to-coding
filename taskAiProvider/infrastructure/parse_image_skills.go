package infrastructure

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"taskAiProvider/domain"
	"gopkg.in/yaml.v3"
)

var imageSkillNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

type imageSkillsYAML struct {
	Version int `yaml:"version"`
	Skills  []struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"skills"`
}

func ParseImageSkillsYAML(raw string) (domain.ImageSkillList, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return domain.ImageSkillList{Version: 1, Skills: []domain.ImageSkill{}}, nil
	}
	var doc imageSkillsYAML
	if err := yaml.Unmarshal([]byte(raw), &doc); err != nil {
		return domain.ImageSkillList{}, fmt.Errorf("invalid imageSkills.yaml: %w", err)
	}
	if doc.Version == 0 {
		doc.Version = 1
	}
	if doc.Version != 1 {
		return domain.ImageSkillList{}, fmt.Errorf("unsupported imageSkills.yaml version %d", doc.Version)
	}
	if len(doc.Skills) > domain.MaxImageSkills {
		return domain.ImageSkillList{}, fmt.Errorf("too many skills (max %d)", domain.MaxImageSkills)
	}
	out := domain.ImageSkillList{Version: doc.Version, Skills: make([]domain.ImageSkill, 0, len(doc.Skills))}
	seen := map[string]struct{}{}
	for i, s := range doc.Skills {
		name := strings.TrimSpace(s.Name)
		if !imageSkillNameRE.MatchString(name) {
			return domain.ImageSkillList{}, fmt.Errorf("invalid skill name %q", s.Name)
		}
		if _, ok := seen[name]; ok {
			return domain.ImageSkillList{}, fmt.Errorf("duplicate skill name %q", name)
		}
		seen[name] = struct{}{}
		desc := strings.TrimSpace(s.Description)
		if utf8.RuneCountInString(desc) > 512 {
			return domain.ImageSkillList{}, fmt.Errorf("skill %q description too long", name)
		}
		item := domain.ImageSkill{Name: name, Description: desc, IsDefault: i == 0}
		out.Skills = append(out.Skills, item)
	}
	if len(out.Skills) > 0 {
		out.DefaultSkill = out.Skills[0].Name
	}
	return out, nil
}
