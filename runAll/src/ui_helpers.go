package main

import (
	"encoding/json"
	"log"
	"net/http"
	"runAll/src/domain"
)

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[ui] json encode error: %v", err)
	}
}

func shouldProbeListenPortForStatus(status Status) bool {
	return domain.IsStartableServiceStatus(string(status))
}

func buildStatusPayload(store *StatusStore, runner *Runner) []serviceStatusPayload {
	services := store.All()
	if runner != nil {
		runner.prefillStartablePortProbeCache(services)
	}
	result := make([]serviceStatusPayload, 0, len(services))
	for _, svc := range services {
		payload := serviceStatusPayload{
			ServiceStatus: cloneServiceStatusForDisplay(svc),
			Hint:          deriveFailureHint(svc),
			Buildable:     false,
			Language:      "—",
		}
		if runner != nil {
			payload.SessionID = resolveServiceSessionID(runner, svc.Name)
			if cfgSvc := runner.findService(svc.Name); cfgSvc != nil {
				payload.Buildable = serviceBuildable(cfgSvc)
				payload.Language = detectServiceLanguage(*cfgSvc)
				// Port probes spawn lsof per service; only startable rows use listen_port_active in the UI.
				if shouldProbeListenPortForStatus(svc.Status) {
					payload.ListenPortActive = runner.hasActivePortListeners(cfgSvc)
				}
			}
		}
		result = append(result, payload)
	}
	return result
}

// cloneServiceStatusForDisplay hides internal StatusPending from the Web UI/API consumers.
func cloneServiceStatusForDisplay(svc *ServiceStatus) *ServiceStatus {
	if svc == nil {
		return nil
	}
	copy := *svc
	if copy.Status == StatusPending {
		copy.Status = ""
	}
	if len(copy.DependsOn) > 0 {
		deps := make([]DepStatus, len(copy.DependsOn))
		copy.DependsOn = deps
		for i, dep := range svc.DependsOn {
			deps[i] = dep
			if deps[i].Status == StatusPending {
				deps[i].Status = ""
			}
		}
	}
	return &copy
}

func writeJSONError(w http.ResponseWriter, message string) {
	writeJSONErrorWithStatus(w, http.StatusBadRequest, message)
}

func writeJSONErrorWithStatus(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("[ui] json encode error: %v", err)
	}
}

// writeJSONConflictWithBulkProgress returns 409 plus the live bulk snapshot so
// the Status UI can attach the progress panel instead of only showing a banner.
func writeJSONConflictWithBulkProgress(w http.ResponseWriter, runner *Runner, message string) {
	payload := map[string]any{"error": message}
	if runner != nil {
		if snap := runner.ActiveBulkProgress(); snap != nil {
			payload["active_bulk_progress"] = snap
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusConflict)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("[ui] json encode error: %v", err)
	}
}
