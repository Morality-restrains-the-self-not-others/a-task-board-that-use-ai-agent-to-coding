package tracelog

import "strings"

// NormalizeLevel maps arbitrary level strings to the project canonical set:
// debug | info | warn | error (always lowercase).
func NormalizeLevel(level string) string {
	s := strings.ToLower(strings.TrimSpace(level))
	switch s {
	case "debug", "info", "warn", "error":
		return s
	case "warning":
		return "warn"
	case "fatal", "panic", "critical", "crit":
		return "error"
	case "trace":
		return "debug"
	case "":
		return "info"
	default:
		return s
	}
}
