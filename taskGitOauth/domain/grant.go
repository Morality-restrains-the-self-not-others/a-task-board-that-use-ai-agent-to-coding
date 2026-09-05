package domain

import (
	"net/url"
	"strings"
)

const (
	GrantKindProject = "project"
	GrantKindComment = "comment"
	GrantKindPending = "pending"
)

// GitsiteFromRepoURL is the plan/spec name for GitsiteHost.
func GitsiteFromRepoURL(repoURL string) string {
	return GitsiteHost(repoURL)
}

// GitsiteHost returns the host[:port] used as L2 grant key (not repo slug).
func GitsiteHost(repoURL string) string {
	raw := strings.TrimSpace(repoURL)
	if raw == "" {
		return ""
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		if strings.EqualFold(parsed.Scheme, "ssh") {
			return strings.ToLower(parsed.Hostname())
		}
		return strings.ToLower(parsed.Host)
	}
	// scp-like git@host:path
	if at := strings.Index(raw, "@"); at >= 0 {
		rest := raw[at+1:]
		if colon := strings.Index(rest, ":"); colon > 0 {
			return strings.ToLower(rest[:colon])
		}
	}
	return ""
}

// GrantIdempotencyKey is the Kafka / upsert boundary for one resource grant.
func GrantIdempotencyKey(kind, resourceID, userID, gitsite string) string {
	return "grant:" + strings.TrimSpace(kind) + ":" + strings.TrimSpace(resourceID) + ":" +
		strings.TrimSpace(userID) + ":" + strings.ToLower(strings.TrimSpace(gitsite))
}

// ApplyResourceGrantGate demotes L1-only token_status when the resource has no L2 marker.
func ApplyResourceGrantGate(tokenStatus string, hasGrant bool) string {
	status := strings.TrimSpace(tokenStatus)
	if hasGrant {
		return status
	}
	if status == "token_available" || status == "token_error" {
		return "not_bound"
	}
	return status
}
