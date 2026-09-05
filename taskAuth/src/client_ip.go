package main

import (
	"log"
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

// handleClientIP echoes the request client public IP for navbar display.
func handleClientIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorDetail(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ip := resolveClientIP(r)
	if ip == "" {
		log.Printf("[taskAuth] client-ip: empty remote address")
		writeErrorDetail(w, r, http.StatusBadGateway, "unable to resolve client ip")
		return
	}
	log.Printf("[taskAuth] client-ip resolved ip=%s public=%t", ip, clientip.IsPublicIPAddr(ip))
	writeJSON(w, http.StatusOK, map[string]string{"ip": ip})
}
