package gatewaycors

import "strings"

// AllowHeadersWith appends service-specific headers without duplicates.
func AllowHeadersWith(extra ...string) string {
	if len(extra) == 0 {
		return AllowHeaders
	}
	seen := make(map[string]bool, len(CORSAllowHeaderNames)+len(extra))
	out := make([]string, 0, len(CORSAllowHeaderNames)+len(extra))
	for _, h := range CORSAllowHeaderNames {
		if seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, h)
	}
	for _, h := range extra {
		h = strings.TrimSpace(h)
		if h == "" || seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, h)
	}
	return strings.Join(out, ", ")
}
