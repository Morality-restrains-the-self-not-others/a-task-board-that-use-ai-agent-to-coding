package main

import (
	"net/http"

	"clientip"
)

// resolveClientIP returns the caller-facing address.
// Implementation lives in shareLib/clientip (single source of truth,
// OPT-20260826-001); the per-service name is kept so call sites and
// unit tests stay unchanged.
func resolveClientIP(r *http.Request) string {
	return clientip.Resolve(r)
}
