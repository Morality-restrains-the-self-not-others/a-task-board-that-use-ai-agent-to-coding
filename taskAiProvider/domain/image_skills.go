package domain

const (
	EventContainerImageSkillsExtracted = "ContainerImageSkillsExtracted"
	DefaultImageSkillsPath             = "/app/imageSkills.yaml"
	MaxImageSkills                     = 32
)

// ImageSkill is one named agent skill shipped in a container image.
type ImageSkill struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

// ImageSkillList is the bound catalog. Skills[0] is always the default when non-empty.
type ImageSkillList struct {
	Version      int          `json:"version"`
	DefaultSkill string       `json:"default_skill"`
	Skills       []ImageSkill `json:"skills"`
}

func (l ImageSkillList) DefaultName() string {
	if l.DefaultSkill != "" {
		return l.DefaultSkill
	}
	if len(l.Skills) == 0 {
		return ""
	}
	return l.Skills[0].Name
}

func (l ImageSkillList) Has(name string) bool {
	for _, s := range l.Skills {
		if s.Name == name {
			return true
		}
	}
	return false
}
