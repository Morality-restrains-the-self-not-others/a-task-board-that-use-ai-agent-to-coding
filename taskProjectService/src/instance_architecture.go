package main

import (
	"fmt"
	"regexp"
	"strings"
)

// inferInstanceArchitecture maps Aliyun instance_type to CPU architecture (best-effort).
// Keep in sync with taskCloudService/src/compute_image_resolve.go and compute_image_resolve_arch_test.go.
func inferInstanceArchitecture(instanceType string) string {
	t := strings.ToLower(strings.TrimSpace(instanceType))
	if t == "" {
		return ""
	}
	if strings.Contains(t, "arm") ||
		strings.Contains(t, "g8y") || strings.Contains(t, "c8y") || strings.Contains(t, "r8y") ||
		strings.Contains(t, "c6r") || strings.Contains(t, "g6r") || strings.Contains(t, "r6r") {
		return "arm64"
	}
	return "x86_64"
}

func normalizeCPUArchitecture(value string) string {
	n := strings.ToLower(strings.TrimSpace(value))
	switch n {
	case "x86", "x86_64", "amd64":
		return "x86_64"
	case "arm", "arm64", "aarch64":
		return "arm64"
	default:
		return n
	}
}

func normalizeCPUArchitectureList(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		canonical := normalizeCPUArchitecture(v)
		if canonical == "" {
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	return out
}

func knownCPUArchitectures(values []string) []string {
	out := make([]string, 0, 2)
	for _, v := range normalizeCPUArchitectureList(values) {
		if v == "x86_64" || v == "arm64" {
			out = append(out, v)
		}
	}
	return out
}

// cpuArchTokenRe matches ISA tokens with non-alnum boundaries (e.g. private_x86_64-latest).
var cpuArchTokenRe = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(x86_64|amd64|aarch64|arm64)(?:[^a-z0-9]|$)`)

func extractCPUArchitecturesFromText(texts ...string) []string {
	found := make([]string, 0, 2)
	for _, text := range texts {
		for _, m := range cpuArchTokenRe.FindAllStringSubmatch(text, -1) {
			if len(m) > 1 {
				found = append(found, m[1])
			}
		}
	}
	return knownCPUArchitectures(found)
}

func formatImageRequiredArch(canonical, raw []string) string {
	if len(canonical) > 0 {
		return strings.Join(canonical, ", ")
	}
	shown := make([]string, 0, len(raw))
	for _, v := range raw {
		s := strings.TrimSpace(v)
		if s == "" || s == "<nil>" {
			continue
		}
		shown = append(shown, s)
	}
	if len(shown) > 0 {
		return strings.Join(shown, ", ") + "（无法识别为 x86_64/arm64）"
	}
	return "未声明"
}

func formatInstanceSupportedArch(instanceType string) string {
	inst := strings.TrimSpace(instanceType)
	got := inferInstanceArchitecture(inst)
	if got == "" {
		if inst == "" {
			return "未声明"
		}
		return "无法从实例规格 " + inst + " 推断"
	}
	if inst != "" {
		return fmt.Sprintf("%s（实例规格 %s）", got, inst)
	}
	return got
}
