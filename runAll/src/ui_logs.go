package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func registerLogsHandlers(mux *http.ServeMux, runner *Runner) {
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}

		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" {
			writeJSONError(w, "name is required")
			return
		}
		if runner.findService(name) == nil {
			writeJSONError(w, fmt.Sprintf("service %q not found", name))
			return
		}

		linesRaw := r.URL.Query().Get("lines")
		if linesRaw == "" {
			writeJSONError(w, "lines is required")
			return
		}
		lines, err := strconv.Atoi(linesRaw)
		if err != nil || lines <= 0 {
			writeJSONError(w, "lines must be a positive integer")
			return
		}
		if lines > maxLogsLines {
			writeJSONError(w, fmt.Sprintf("lines must be <= %d", maxLogsLines))
			return
		}
		if runner.logRepository == nil {
			writeJSONError(w, "log repository is required")
			return
		}

		entries := runner.logRepository.Tail(name, lines)
		if fileEntries := runner.stdioFileLogEntries(name, lines); len(fileEntries) > 0 {
			entries = fileEntries
		}
		if len(entries) == 0 {
			for _, entry := range runner.lifecycleLogSnapshot(name) {
				runner.logRepository.Append(name, entry)
			}
			entries = runner.logRepository.Tail(name, lines)
		} else {
			runner.refreshPendingDiagnostic(name)
			entries = runner.logRepository.Tail(name, lines)
		}

		writeJSON(w, map[string]any{
			"name":  name,
			"lines": entries,
		})
	})
	mux.HandleFunc("/api/logs/clear", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSONError(w, "invalid json")
			return
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			writeJSONError(w, "name is required")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		if runner.logRepository == nil {
			writeJSONError(w, "log repository is required")
			return
		}
		if runner.findService(body.Name) == nil {
			writeJSONError(w, fmt.Sprintf("service %q not found", body.Name))
			return
		}

		runner.logRepository.Clear(body.Name)
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/logs/clear-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONErrorWithStatus(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if runner == nil {
			writeJSONError(w, "runner is required")
			return
		}
		result := runner.ClearServiceLogs()
		writeJSON(w, map[string]any{
			"status":                  result.Status,
			"memory_services_cleared": result.MemoryServicesCleared,
			"files_truncated":         result.FilesTruncated,
			"method":                  "truncate",
		})
	})
}
