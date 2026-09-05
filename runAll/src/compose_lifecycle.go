package main

import "strings"

func isDetachLaunchCommand(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	if strings.Contains(lower, "run-infra.sh") {
		return true
	}
	if strings.Contains(lower, "run.sh managed") || strings.Contains(lower, "run.sh start") {
		return true
	}
	if !strings.Contains(lower, "compose") {
		return false
	}
	if !strings.Contains(lower, " up") {
		return false
	}
	return strings.Contains(lower, "-d")
}
