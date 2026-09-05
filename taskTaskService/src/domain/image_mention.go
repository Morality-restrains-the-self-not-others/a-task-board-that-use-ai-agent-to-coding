package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const MentionTypeInstalledImage = "installed_image"

var imageSkillNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// ImageMention is a structured @ reference to an installed container image.
type ImageMention struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Skill string `json:"skill,omitempty"`
}

// ImageSkillList is the catalog bound to an installed image (first item is default).
type ImageSkillList struct {
	Version      int          `json:"version"`
	DefaultSkill string       `json:"default_skill"`
	Skills       []ImageSkill `json:"skills"`
}

type ImageSkill struct {
	// ID 是 taskCloudService D1=B 服务端派生的稳定技能 ID（sk_<sha1(name+seed)>[:12]），
	// 由 cloud 的 image_skills 响应透传；存量/未回填时为 ""。
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

func (l ImageSkillList) DefaultName() string {
	if strings.TrimSpace(l.DefaultSkill) != "" {
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

var (
	ErrMentionAtModeDisabled = errors.New("container image at mode is disabled")
	ErrTooManyMentions       = errors.New("at most one mention allowed")
	ErrInvalidMention        = errors.New("invalid mention")
	ErrUnknownImageSkill     = errors.New("unknown image skill")
)

// ParseMentions extracts mentions from a comment request body field.
func ParseMentions(raw interface{}) ([]ImageMention, error) {
	if raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: mentions must be an array", ErrInvalidMention)
	}
	if len(arr) == 0 {
		return nil, nil
	}
	out := make([]ImageMention, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%w: mention must be an object", ErrInvalidMention)
		}
		skill := strings.TrimSpace(fmt.Sprintf("%v", m["skill"]))
		if skill == "<nil>" {
			skill = ""
		}
		out = append(out, ImageMention{
			Type:  strings.TrimSpace(fmt.Sprintf("%v", m["type"])),
			ID:    strings.TrimSpace(fmt.Sprintf("%v", m["id"])),
			Name:  strings.TrimSpace(fmt.Sprintf("%v", m["name"])),
			Skill: skill,
		})
	}
	return out, nil
}

// ValidateMentions enforces at-mode gate and mention shape (0..1 per comment).
func ValidateMentions(atModeEnabled bool, mentions []ImageMention) error {
	if len(mentions) == 0 {
		return nil
	}
	if len(mentions) > 1 {
		return ErrTooManyMentions
	}
	if !atModeEnabled {
		return ErrMentionAtModeDisabled
	}
	m := mentions[0]
	if m.Type != MentionTypeInstalledImage {
		return fmt.Errorf("%w: unsupported mention type %q", ErrInvalidMention, m.Type)
	}
	if m.ID == "" || m.ID == "<nil>" {
		return fmt.Errorf("%w: mention id required", ErrInvalidMention)
	}
	if m.Skill != "" && !imageSkillNameRE.MatchString(m.Skill) {
		return fmt.Errorf("%w: invalid skill name", ErrInvalidMention)
	}
	return nil
}

// ApplyMentionSkill fills default skill or rejects unknown names when the image published a list.
func ApplyMentionSkill(m *ImageMention, list ImageSkillList) error {
	if m == nil {
		return nil
	}
	skill := strings.TrimSpace(m.Skill)
	if skill == "" {
		m.Skill = list.DefaultName()
		return nil
	}
	if !imageSkillNameRE.MatchString(skill) {
		return fmt.Errorf("%w: invalid skill name", ErrInvalidMention)
	}
	if len(list.Skills) == 0 {
		return nil
	}
	if !list.Has(skill) {
		return fmt.Errorf("%w: %s", ErrUnknownImageSkill, skill)
	}
	m.Skill = skill
	return nil
}

func DecodeImageSkillsJSON(raw interface{}) ImageSkillList {
	empty := ImageSkillList{Version: 1, Skills: []ImageSkill{}}
	if raw == nil {
		return empty
	}
	var b []byte
	switch t := raw.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return empty
		}
		b = []byte(t)
	case map[string]interface{}:
		var err error
		b, err = json.Marshal(t)
		if err != nil {
			return empty
		}
	default:
		var err error
		b, err = json.Marshal(t)
		if err != nil {
			return empty
		}
	}
	var list ImageSkillList
	if err := json.Unmarshal(b, &list); err != nil {
		return empty
	}
	if list.Skills == nil {
		list.Skills = []ImageSkill{}
	}
	return list
}

// MentionsJSON serializes mentions for DB storage; empty slice → "".
func MentionsJSON(mentions []ImageMention) string {
	if len(mentions) == 0 {
		return ""
	}
	b, err := json.Marshal(mentions)
	if err != nil {
		return ""
	}
	return string(b)
}

// ParseMentionsJSON deserializes stored mentions_json.
func ParseMentionsJSON(raw string) ([]ImageMention, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []ImageMention
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}
