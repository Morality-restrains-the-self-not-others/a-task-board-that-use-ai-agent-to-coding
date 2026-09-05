package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func registerProxyHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/proxy-config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleProxyConfigGet(w, r, runner)
		case http.MethodPost:
			handleProxyConfigPost(w, r, runner)
		default:
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
}

type proxyConfigPayload struct {
	UseProxy   bool             `json:"use_proxy"`
	PerService map[string]*bool `json:"per_service,omitempty"`
}

func handleProxyConfigGet(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if runner == nil || runner.cfg == nil {
		writeJSONError(w, "runner is required")
		return
	}
	payload := proxyConfigPayload{
		UseProxy:   runner.cfg.UseProxy,
		PerService: buildPerServiceProxyMap(runner.cfg),
	}
	writeJSON(w, payload)
}

func handleProxyConfigPost(w http.ResponseWriter, r *http.Request, runner *Runner) {
	if runner == nil || runner.cfg == nil {
		writeJSONError(w, "runner is required")
		return
	}
	var body struct {
		UseProxy *bool `json:"use_proxy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, "invalid json")
		return
	}
	if body.UseProxy == nil {
		writeJSONError(w, "use_proxy is required")
		return
	}
	runner.cfg.UseProxy = *body.UseProxy
	log.Printf("[api] proxy-config updated: use_proxy=%v", runner.cfg.UseProxy)
	writeJSON(w, map[string]any{
		"status":    "ok",
		"use_proxy": runner.cfg.UseProxy,
	})
}

// buildPerServiceProxyMap returns per-service proxy overrides for services that
// have a non-nil UseProxy field.
func buildPerServiceProxyMap(cfg *Config) map[string]*bool {
	if cfg == nil {
		return nil
	}
	result := make(map[string]*bool)
	for _, svc := range cfg.Flatten() {
		if svc.UseProxy != nil {
			result[svc.Name] = svc.UseProxy
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// uiListenerLost handles a UI listener that returned with a non-closed error. The
// runAll process must exit promptly — a UI-less runAll would keep polling health
// checks and, if ever SIGTERM'd, tear down the managed services it believes it owns
// (OPT-20260817-001). Managed services are intentionally left running because a
// sibling instance may still own them after a bind failure or a dead listener.
