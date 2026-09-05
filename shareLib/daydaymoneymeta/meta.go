package daydaymoneymeta

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	CurrentVersion = 1
	MaxTags        = 20
	MaxTagLen      = 64
	MaxServiceID   = 64
)

var (
	serviceIDRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{1,63}$`)
	forbiddenKeys = []string{
		"workspace_id", "workspace_ids", "project_id", "project_ids",
		"company_id", "tenant_id",
	}
)

// Meta is the v1 daydaymoney.yaml document (service identity only).
type Meta struct {
	Version     int      `yaml:"version" json:"version"`
	ServiceID   string   `yaml:"service_id" json:"service_id"`
	DisplayName string   `yaml:"display_name" json:"display_name"`
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags" json:"tags"`
}

// ParseYAML parses and validates daydaymoney.yaml content.
func ParseYAML(raw string) (*Meta, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty yaml")
	}
	var root map[string]interface{}
	if err := yaml.Unmarshal([]byte(trimmed), &root); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}
	for _, key := range forbiddenKeys {
		if _, ok := root[key]; ok {
			return nil, fmt.Errorf("forbidden key %q: daydaymoney.yaml must not encode workspace/project/tenant ids", key)
		}
	}
	var m Meta
	if err := yaml.Unmarshal([]byte(trimmed), &m); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks Meta invariants and normalizes tags (trim, dedupe, ensure svc: tag).
func (m *Meta) Validate() error {
	if m == nil {
		return fmt.Errorf("nil meta")
	}
	if m.Version != CurrentVersion {
		return fmt.Errorf("unsupported version %d (want %d)", m.Version, CurrentVersion)
	}
	m.ServiceID = strings.TrimSpace(m.ServiceID)
	if m.ServiceID == "" {
		return fmt.Errorf("service_id is required")
	}
	if len(m.ServiceID) > MaxServiceID || !serviceIDRe.MatchString(m.ServiceID) {
		return fmt.Errorf("invalid service_id %q", m.ServiceID)
	}
	m.DisplayName = strings.TrimSpace(m.DisplayName)
	m.Description = strings.TrimSpace(m.Description)
	m.Tags = NormalizeTags(m.Tags, m.ServiceID)
	if len(m.Tags) == 0 {
		return fmt.Errorf("tags required")
	}
	svcTag := "svc:" + m.ServiceID
	found := false
	for _, t := range m.Tags {
		if strings.EqualFold(t, svcTag) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tags must include %q", svcTag)
	}
	return nil
}

// NormalizeTags trims, drops empties/oversized, dedupes case-insensitively, caps count.
// Ensures svc:<serviceID> is present when serviceID non-empty.
func NormalizeTags(tags []string, serviceID string) []string {
	out := make([]string, 0, len(tags)+1)
	seen := map[string]struct{}{}
	add := func(tag string) {
		tag = strings.TrimSpace(tag)
		if tag == "" || len(tag) > MaxTagLen {
			return
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, tag)
	}
	for _, t := range tags {
		add(t)
		if len(out) >= MaxTags {
			break
		}
	}
	if sid := strings.TrimSpace(serviceID); sid != "" && len(out) < MaxTags {
		svc := "svc:" + sid
		if _, ok := seen[strings.ToLower(svc)]; !ok {
			// prepend canonical svc tag
			out = append([]string{svc}, out...)
			if len(out) > MaxTags {
				out = out[:MaxTags]
			}
		}
	}
	return out
}

// MergeTags unions existing and incoming tags with normalization.
func MergeTags(existing, incoming []string, serviceID string) []string {
	combined := append([]string{}, existing...)
	combined = append(combined, incoming...)
	return NormalizeTags(combined, serviceID)
}

// ServiceTag returns svc:<serviceID>.
func ServiceTag(serviceID string) string {
	return "svc:" + strings.TrimSpace(serviceID)
}

// TagEquals reports case-insensitive equality.
func TagEquals(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// TagsContain reports whether tags contain needle (case-insensitive).
func TagsContain(tags []string, needle string) bool {
	for _, t := range tags {
		if TagEquals(t, needle) {
			return true
		}
	}
	return false
}

// LoadFile reads and parses an daydaymoney.yaml path.
func LoadFile(path string) (*Meta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseYAML(string(b))
}

// LoadNearDir looks for daydaymoney.yaml in dir then dir's parents (max 4).
func LoadNearDir(dir string) (*Meta, error) {
	cur := dir
	for i := 0; i < 5; i++ {
		p := filepath.Join(cur, "daydaymoney.yaml")
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return LoadFile(p)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return nil, fmt.Errorf("daydaymoney.yaml not found near %s", dir)
}
