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

// resolveAutoRunClientPublicIP prefers body client_public_ip, else create/update request headers.
// Required because auto_run triggers start-vm as a server-to-server call whose RemoteAddr/XFF
// would otherwise be taskTaskService itself, not the browser user.
func resolveAutoRunClientPublicIP(r *http.Request, body map[string]interface{}) string {
	if body != nil {
		if v := strings.TrimSpace(strField(body, "client_public_ip")); v != "" {
			return v
		}
	}
	return resolveClientIP(r)
}
