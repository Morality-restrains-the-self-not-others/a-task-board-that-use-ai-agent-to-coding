package main

import (
	"net/http"
	"strings"
)

func detachTaskProjectsByProjectID(projectID string) (int64, error) {
	pid := strings.TrimSpace(projectID)
	if pid == "" {
		return 0, nil
	}
	res, err := db.Exec(`DELETE FROM task_projects WHERE project_id=?`, pid)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// handleInternalDetachTaskProjects — POST /api/internal/task-projects/detach-by-project/
// Body: {"project_id":"..."}. Idempotent: missing rows still 200 with deleted=0.
func handleInternalDetachTaskProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !isInternalCall(r) && !internalSecretOK(r) {
		writeError(w, r, http.StatusForbidden, "internal only")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON")
		return
	}
	projectID := strings.TrimSpace(strField(body, "project_id"))
	if projectID == "" {
		writeError(w, r, http.StatusBadRequest, "project_id required")
		return
	}
	n, err := detachTaskProjectsByProjectID(projectID)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"project_id": projectID,
		"deleted":    n,
	})
}
