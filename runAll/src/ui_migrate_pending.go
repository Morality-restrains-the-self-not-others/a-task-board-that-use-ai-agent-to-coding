package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func registerMigratePendingHandler(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/dev/migrate-status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		bypass := r.URL.Query().Get("refresh") == "1"
		rep := runner.MigratePendingStatus(r.Context(), bypass)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(rep); err != nil {
			log.Printf("[ui] migrate-status encode error: %v", err)
		}
	})
}
