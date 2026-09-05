package domain

import (
	"net/url"
	"strings"
)

const (
	ReachabilityUnconfigured    = "unconfigured"
	ReachabilitySkippedIntranet = "skipped_intranet"
	ReachabilityReachable       = "reachable"
	ReachabilityUnreachable     = "unreachable"
	OAuthNetworkOK              = "ok"
)

// ClassifyReachability decides whether the control plane should probe base_url.
// Intranet GitLab is expected to be unreachable from the platform.
func ClassifyReachability(configured, intranet bool) (skip bool, status string) {
	if !configured {
		return true, ReachabilityUnconfigured
	}
	if intranet {
		return true, ReachabilitySkippedIntranet
	}
	return false, ""
}

// OAuthProbeNetworkStatus maps GitLab liveness onto the user-app-connection probe.
// Non-GitLab hosts omit network_status. Intranet GitLab skips live probes.
// Public GitLab that cannot be reached is unreachable even if a cached token exists.
func OAuthProbeNetworkStatus(isGitLab, intranet, liveUnreachable bool) string {
	if !isGitLab {
		return ""
	}
	if intranet {
		return ReachabilitySkippedIntranet
	}
	if liveUnreachable {
		return ReachabilityUnreachable
	}
	return OAuthNetworkOK
}

// RepoHTTPOrigin returns scheme://host for an http(s) repo or base URL.
func RepoHTTPOrigin(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	return scheme + "://" + u.Host
}

// ProbeTargetURL joins a normalized base URL with "/" for a cheap liveness GET.
func ProbeTargetURL(baseURL string) string {
	s := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if s == "" {
		return ""
	}
	return s + "/"
}
