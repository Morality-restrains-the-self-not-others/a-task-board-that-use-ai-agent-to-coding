package main

import (
	"database/sql"
	"net/http"
	"strings"
)

func handleProjectBranches(w http.ResponseWriter, r *http.Request, tenantID, projectID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	repoURL := strings.TrimSpace(r.URL.Query().Get("repo_url"))
	if repoURL == "" {
		writeError(w, r, http.StatusBadRequest, "repo_url is required")
		return
	}

	detail, err := loadProjectDetail(projectID)
	if err == sql.ErrNoRows {
		writeError(w, r, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if rejectIfProjectNotInTenant(w, r, detail, tenantID) {
		return
	}

	userID := getAuthUser(r)

	gitlabSession := ""
	if cookie, err := r.Cookie("_gitlab_session"); err == nil {
		gitlabSession = strings.TrimSpace(cookie.Value)
	}

	payload := listProjectRepoBranches(userID, repoURL, gitlabSession, tenantID, traceHeadersFromRequest(r))
	writeJSON(w, http.StatusOK, payload)
}
