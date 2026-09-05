package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"snowflake"
)

type projectGitOAuthGrant struct {
	ID           string
	CompanyID    string
	ProjectID    string
	UserID       string
	Gitsite      string
	RemoteUserID string
	GrantedAt    time.Time
}

func sanitizeResourceID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

func applyResourceGrantGate(tokenStatus string, hasGrant bool) string {
	status := strings.TrimSpace(tokenStatus)
	if hasGrant {
		return status
	}
	if status == tokenStatusAvailable || status == tokenStatusTokenError {
		return tokenStatusNotBound
	}
	return status
}

func hasProjectGitOAuthGrant(projectID, userID, gitsite string) bool {
	_, ok := lookupProjectGitOAuthGrant(projectID, userID, gitsite)
	return ok
}

func lookupProjectGitOAuthGrant(projectID, userID, gitsite string) (remoteUserID string, ok bool) {
	if db == nil {
		return "", false
	}
	pid := strings.TrimSpace(projectID)
	uid := strings.TrimSpace(userID)
	site := strings.ToLower(strings.TrimSpace(gitsite))
	if pid == "" || uid == "" || site == "" {
		return "", false
	}
	var remote string
	err := db.QueryRow(`
SELECT COALESCE(remote_user_id,'') FROM project_git_oauth_grant
WHERE project_id=? AND task2app_user_id=? AND gitsite=? LIMIT 1`, pid, uid, site).Scan(&remote)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(remote), true
}

func upsertProjectGitOAuthGrant(companyID, projectID, userID, gitsite, remoteUserID string) error {
	if db == nil {
		return sql.ErrConnDone
	}
	pid := strings.TrimSpace(projectID)
	uid := strings.TrimSpace(userID)
	site := strings.ToLower(strings.TrimSpace(gitsite))
	if pid == "" || uid == "" || site == "" {
		return nil
	}
	cid := strings.TrimSpace(companyID)
	if cid == "" {
		_ = db.QueryRow(`SELECT COALESCE(company_id,'') FROM project_entries WHERE id=? LIMIT 1`, pid).Scan(&cid)
	}
	id := snowflake.GenerateIDString()
	_, err := db.Exec(`
INSERT INTO project_git_oauth_grant (id, company_id, project_id, task2app_user_id, gitsite, remote_user_id, granted_at)
VALUES (?, ?, ?, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
	remote_user_id=VALUES(remote_user_id),
	granted_at=NOW(),
	company_id=IF(VALUES(company_id)='', company_id, VALUES(company_id))`,
		id, cid, pid, uid, site, strings.TrimSpace(remoteUserID))
	return err
}

func handleInternalMarkProjectGitOAuthGrant(w http.ResponseWriter, r *http.Request) {
	if !requireProjectInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
	switch r.Method {
	case http.MethodGet:
		handleInternalLookupProjectGitOAuthGrant(w, r)
		return
	case http.MethodPost:
		handleInternalUpsertProjectGitOAuthGrant(w, r)
		return
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func handleInternalLookupProjectGitOAuthGrant(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	projectID := strings.TrimSpace(q.Get("project_id"))
	userID := strings.TrimSpace(q.Get("user_id"))
	gitsite := strings.TrimSpace(q.Get("gitsite"))
	if projectID == "" || userID == "" || gitsite == "" {
		writeError(w, r, http.StatusBadRequest, "project_id, user_id, gitsite required")
		return
	}
	remote, ok := lookupProjectGitOAuthGrant(projectID, userID, gitsite)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"has_grant":      ok,
		"remote_user_id": remote,
	})
}

func handleInternalUpsertProjectGitOAuthGrant(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON")
		return
	}
	var body map[string]string
	if err := json.Unmarshal(raw, &body); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid JSON")
		return
	}
	projectID := strings.TrimSpace(body["project_id"])
	userID := strings.TrimSpace(body["user_id"])
	gitsite := strings.TrimSpace(body["gitsite"])
	if projectID == "" || userID == "" || gitsite == "" {
		writeError(w, r, http.StatusBadRequest, "project_id, user_id, gitsite required")
		return
	}
	if err := upsertProjectGitOAuthGrant(body["company_id"], projectID, userID, gitsite, body["remote_user_id"]); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	publishProjectGitOAuthGranted(projectID, userID, gitsite, body["remote_user_id"])
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}
