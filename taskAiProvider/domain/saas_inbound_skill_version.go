package domain

import (
	"fmt"
	"strings"
)

const (
	SkillVersionStatusCurrent    = "current"
	SkillVersionStatusDeprecated = "deprecated"
	SkillVersionStatusSunset     = "sunset"

	EventContainerImageSaasInboundSkillVersionAssigned = "ContainerImageSaasInboundSkillVersionAssigned"
)

// SkillVersionEntry is one published container→SaaS inbound contract revision.
type SkillVersionEntry struct {
	Version  string
	Status   string
	Released string
	Summary  string
	Doc      string
}

func NormalizeSkillVersion(v string) string {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	return strings.TrimSpace(s)
}

func isWritableSkillStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case SkillVersionStatusCurrent, SkillVersionStatusDeprecated:
		return true
	default:
		return false
	}
}

// ValidateSaasInboundSkillVersion rejects empty, unknown, and sunset contract versions.
func ValidateSaasInboundSkillVersion(v string, catalog []SkillVersionEntry) error {
	n := NormalizeSkillVersion(v)
	if n == "" {
		return fmt.Errorf("请选择已发布的容器→SaaS 接口版本")
	}
	for _, e := range catalog {
		if NormalizeSkillVersion(e.Version) != n {
			continue
		}
		if !isWritableSkillStatus(e.Status) {
			return fmt.Errorf("接口版本 %s 已停用，请选择仍在支持的版本", n)
		}
		return nil
	}
	return fmt.Errorf("请选择已发布的容器→SaaS 接口版本")
}

func CatalogCurrentVersion(catalog []SkillVersionEntry) string {
	for _, e := range catalog {
		if e.Status == SkillVersionStatusCurrent {
			return NormalizeSkillVersion(e.Version)
		}
	}
	if len(catalog) > 0 {
		return NormalizeSkillVersion(catalog[0].Version)
	}
	return ""
}

// SupportedSkillVersionSet 返回「仍可写/可选用」的契约版本集合（current/deprecated；
// sunset 被排除）。OPT-20260820-027：契约 sunset 后，已上架镜像若仍声明旧版本，
// 公开目录必须过滤掉，避免用户选中已下线实现。
func SupportedSkillVersionSet(catalog []SkillVersionEntry) map[string]bool {
	set := make(map[string]bool, len(catalog))
	for _, e := range catalog {
		if isWritableSkillStatus(e.Status) {
			set[NormalizeSkillVersion(e.Version)] = true
		}
	}
	return set
}

func SkillVersionDocFile(catalog []SkillVersionEntry, version string) (string, bool) {
	n := NormalizeSkillVersion(version)
	for _, e := range catalog {
		if NormalizeSkillVersion(e.Version) == n {
			doc := strings.TrimSpace(e.Doc)
			if doc == "" {
				return "", false
			}
			if strings.Contains(doc, "..") || strings.ContainsAny(doc, `/\`) {
				return "", false
			}
			return doc, true
		}
	}
	return "", false
}
