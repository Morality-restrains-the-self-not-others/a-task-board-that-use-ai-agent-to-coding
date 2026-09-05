// Package clientip resolves the caller-facing public IP from HTTP request
// headers, skipping Docker bridge / RFC1918 / loopback / link-local hops.
//
// Single source of truth for login history, auto security-group whitelists,
// protocol consent, and start-vm client-IP contexts (OPT-20260826-001).
// taskAuth / taskBill / taskCloudService / taskTaskService used to each carry
// a copy of this algorithm; they now delegate to Resolve.
package clientip

import (
	"net"
	"net/http"
	"strings"
)

// Resolve returns the caller-facing address.
// Docker published ports make RemoteAddr / X-Real-IP the bridge gateway
// (taskfe_default 172.26.0.1, taskgateway_default 172.25.0.1). Walk
// X-Forwarded-For from the right, skip RFC1918 / loopback / link-local,
// then X-Real-IP, then RemoteAddr. If every hop is private, keep the
// leftmost hop (same as the historical first-XFF fallback).
func Resolve(r *http.Request) string {
	if r == nil {
		return ""
	}
	xffHops := SplitForwardedFor(r.Header.Get("X-Forwarded-For"))
	if ip := rightmostPublicIP(xffHops); ip != "" {
		return ip
	}
	xri := CanonicalIPString(r.Header.Get("X-Real-IP"))
	if IsPublicIPAddr(xri) {
		return xri
	}
	remote := CanonicalIPString(r.RemoteAddr)
	if IsPublicIPAddr(remote) {
		return remote
	}
	if len(xffHops) > 0 {
		return xffHops[0]
	}
	if xri != "" {
		return xri
	}
	return remote
}

// SplitForwardedFor splits an X-Forwarded-For list and canonicalizes each hop.
func SplitForwardedFor(xff string) []string {
	out := make([]string, 0, 4)
	for _, part := range strings.Split(xff, ",") {
		if hop := CanonicalIPString(part); hop != "" {
			out = append(out, hop)
		}
	}
	return out
}

func rightmostPublicIP(hops []string) string {
	for i := len(hops) - 1; i >= 0; i-- {
		if IsPublicIPAddr(hops[i]) {
			return hops[i]
		}
	}
	return ""
}

// CanonicalIPString normalizes an IP or host:port into a bare IP string.
func CanonicalIPString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if ip := net.ParseIP(s); ip != nil {
		return ip.String()
	}
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		return s
	}
	host = strings.Trim(host, "[]")
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return strings.TrimSpace(host)
}

// IsPublicIPAddr reports whether s is a routable public unicast address.
func IsPublicIPAddr(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() && !ip.IsUnspecified()
}
