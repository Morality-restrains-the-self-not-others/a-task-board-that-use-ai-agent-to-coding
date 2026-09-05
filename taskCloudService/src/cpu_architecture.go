package main

import (
	"regexp"
)

// Keep CPU ISA helpers in sync with taskProjectService/src/instance_architecture.go.

func knownCPUArchitectures(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 2)
	for _, v := range values {
		canonical := normalizeArchitecture(v)
		if canonical != "x86_64" && canonical != "arm64" {
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
