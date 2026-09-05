package main

import (
	"net/http"
	"strings"

	"clientip"
)

// resolveClientIP returns the caller-facing address.
// Implementation lives in shareLib/clientip (single source of truth,
// OPT-20260826-001); the per-service name is kept so call sites and
// unit tests stay unchanged.
func resolveClientIP(r *http.Request) string {
	return clientip.Resolve(r)
}

// resolveStartVmClientPublicIP prefers explicit body client_public_ip, else request headers.
func resolveStartVmClientPublicIP(r *http.Request, body map[string]interface{}) string {
	if body != nil {
		if v := strings.TrimSpace(strField(body, "client_public_ip")); v != "" {
			return v
		}
	}
	return resolveClientIP(r)
}
