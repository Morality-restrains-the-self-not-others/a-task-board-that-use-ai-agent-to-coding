package main

import "strings"

func toAliyunCpuArchitecture(normalizedArch string) string {
	switch normalizeRuntimeArchitecture(normalizedArch) {
	case "x86_64":
		return "X86"
	case "arm64":
		return "ARM"
	default:
		return ""
	}
}

func fromAliyunCpuArchitecture(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "X86":
		return "x86_64"
	case "ARM":
		return "arm64"
	default:
		return normalizeRuntimeArchitecture(raw)
	}
}
