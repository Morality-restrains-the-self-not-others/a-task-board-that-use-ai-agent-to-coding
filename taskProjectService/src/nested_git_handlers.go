package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
)

func handleProjectNestedGitRepos(w http.ResponseWriter, r *http.Request, tenantID, projectID string) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
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

	repoURL := strings.TrimSpace(r.URL.Query().Get("repo_url"))
	if repoURL == "" {
		if repos, ok := detail["git_repos"].([]string); ok && len(repos) > 0 {
			repoURL = strings.TrimSpace(repos[0])
		}
	}
	if repoURL == "" {
		writeError(w, r, http.StatusBadRequest, "No repository URL provided")
		return
	}

	userID := getAuthUser(r)
	gitlabSession := ""
	if cookie, err := r.Cookie("_gitlab_session"); err == nil {
		gitlabSession = strings.TrimSpace(cookie.Value)
	}

	log.Printf("[taskProjectService] nested-git-repos project=%s user=%s parent=%s", projectID, userID, repoURL)
	payload := listNestedGitRepos(userID, repoURL, gitlabSession, tenantID, traceHeadersFromRequest(r))
	writeJSON(w, http.StatusOK, payload)
}
