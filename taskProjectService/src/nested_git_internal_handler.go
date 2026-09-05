package main

import (
	"log"
	"net/http"
	"strings"
)

// handleInternalNestedGitRepos serves GET /api/internal/nested-git-repos/?repo_url=&user_id=&tenant_id=
// (company_id alias) for service-to-service discovery. tenant_id is required to match Path A GitLab.
func handleInternalNestedGitRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	repoURL := strings.TrimSpace(r.URL.Query().Get("repo_url"))
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	tenantID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if tenantID == "" {
		tenantID = strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	}
	if tenantID == "" {
		tenantID = requestTenantID(r)
	}
	if repoURL == "" {
		writeError(w, r, http.StatusBadRequest, "repo_url is required")
		return
	}
	log.Printf("[taskProjectService] event=internal_nested_git_repos user=%s tenant_id=%s parent=%s", userID, tenantID, repoURL)
	payload := listNestedGitRepos(userID, repoURL, "", tenantID, traceHeadersFromRequest(r))
	writeJSON(w, http.StatusOK, payload)
}
