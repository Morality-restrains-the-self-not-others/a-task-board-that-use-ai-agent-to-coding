package smtp

import (
	"strings"
)

// IsPermanent reports SMTP failures that must not be retried.
// QQ 535 Login fail (wrong auth code / account abnormal / login frequency limited)
// is permanent: retrying exhausts DLT and tightens QQ rate limits.
// Network/DNS/timeout stay retryable.
func IsPermanent(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "server misbehaving") ||
		strings.Contains(msg, "temporary") ||
		strings.Contains(msg, "eof") {
		return false
	}
	if strings.Contains(msg, "535") || strings.Contains(msg, "login fail") {
		return true
	}
	if strings.Contains(msg, "550") || strings.Contains(msg, "553") || strings.Contains(msg, "554") {
		return true
	}
	return false
}
