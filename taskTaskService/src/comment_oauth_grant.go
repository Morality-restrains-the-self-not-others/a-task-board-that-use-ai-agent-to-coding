package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func gitsiteFromRepoURL(repoURL string) string {
	raw := strings.TrimSpace(repoURL)
	if raw == "" {
		return ""
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		if strings.EqualFold(parsed.Scheme, "ssh") {
			return strings.ToLower(parsed.Hostname())
		}
		return strings.ToLower(parsed.Host)
	}
	if at := strings.Index(raw, "@"); at >= 0 {
		rest := raw[at+1:]
		if colon := strings.Index(rest, ":"); colon > 0 {
			return strings.ToLower(rest[:colon])
		}
	}
	return ""
}

func stampImplicitCommentOAuthGitsite(selections []RepoIdentitySelection) []RepoIdentitySelection {
	if len(selections) == 0 {
		return selections
	}
	now := time.Now().UTC().Format(time.RFC3339)
	out := make([]RepoIdentitySelection, len(selections))
	copy(out, selections)
	changed := false
	for i := range out {
		if strings.TrimSpace(out[i].OauthGitsite) != "" {
			continue
		}
		site := gitsiteFromRepoURL(out[i].RepoURL)
		if !isOAuthCapableGitsite(site) {
			continue
		}
		out[i].OauthGitsite = site
		if strings.TrimSpace(out[i].OauthGrantedAt) == "" {
			out[i].OauthGrantedAt = now
		}
		changed = true
	}
	if !changed {
		return selections
	}
	return out
}

func stampCommentOAuthGrant(selections []RepoIdentitySelection, gitsite, remoteUserID string) []RepoIdentitySelection {
	site := strings.ToLower(strings.TrimSpace(gitsite))
	if site == "" || len(selections) == 0 {
		return selections
	}
	now := time.Now().UTC().Format(time.RFC3339)
	out := make([]RepoIdentitySelection, len(selections))
	copy(out, selections)
	for i := range out {
		if gitsiteFromRepoURL(out[i].RepoURL) != site {
			continue
		}
		out[i].OauthGitsite = site
		out[i].OauthRemoteUserID = strings.TrimSpace(remoteUserID)
		out[i].OauthGrantedAt = now
	}
	return out
}

const errCodeCommentOAuthGrantMissing = "git_oauth_comment_grant_missing"

func isOAuthCapableGitsite(site string) bool {
	s := strings.ToLower(strings.TrimSpace(site))
	if s == "" {
		return false
	}
	if s == "github.com" || strings.HasSuffix(s, ".github.com") {
		return true
	}
	return strings.Contains(s, "gitlab")
}

func commentIdentitiesHaveOAuthGrant(selections []RepoIdentitySelection, site string) bool {
	site = strings.ToLower(strings.TrimSpace(site))
	if site == "" {
		return false
	}
	for _, row := range selections {
		if strings.EqualFold(strings.TrimSpace(row.OauthGitsite), site) {
			return true
		}
	}
	return false
}

func missingOAuthCapableGrantSite(selections []RepoIdentitySelection, extraURLs ...string) string {
	seen := map[string]struct{}{}
	consider := func(raw string) string {
		site := gitsiteFromRepoURL(raw)
		if !isOAuthCapableGitsite(site) {
			return ""
		}
		if _, ok := seen[site]; ok {
			return ""
		}
		seen[site] = struct{}{}
		if !commentIdentitiesHaveOAuthGrant(selections, site) {
			return site
		}
		return ""
	}
	for _, url := range extraURLs {
		if site := consider(url); site != "" {
			return site
		}
	}
	for _, row := range selections {
		if site := consider(row.RepoURL); site != "" {
			return site
		}
	}
	return ""
}

func loadCommentOAuthGrantsForTask(taskID string) []map[string]string {
	taskID = strings.TrimSpace(taskID)
	if db == nil || taskID == "" {
		return nil
	}
	rows, err := db.Query(
		`SELECT COALESCE(created_by_id,''), COALESCE(repo_identities_json,'') FROM task_comments WHERE task_id=?`,
		taskID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []map[string]string{}
	seen := map[string]struct{}{}
	for rows.Next() {
		var userID, jsonText string
		if err := rows.Scan(&userID, &jsonText); err != nil {
			continue
		}
		userID = strings.TrimSpace(userID)
		sels, _ := ParseRepoIdentities(repoIdentitiesFromJSONColumn(jsonText))
		for _, row := range sels {
			site := strings.ToLower(strings.TrimSpace(row.OauthGitsite))
			if site == "" {
				continue
			}
			key := userID + "\x00" + site
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, map[string]string{"user_id": userID, "gitsite": site})
		}
	}
	return out
}

var consumeGitOAuthGrantTicketFn = consumeGitOAuthGrantTicketLive

func consumeGitOAuthGrantTicket(userID, gitsite, ticketID string) (remoteUserID string, ok bool) {
	return consumeGitOAuthGrantTicketFn(userID, gitsite, ticketID)
}

func consumeGitOAuthGrantTicketLive(userID, gitsite, ticketID string) (remoteUserID string, ok bool) {
	ticketID = strings.TrimSpace(ticketID)
	userID = strings.TrimSpace(userID)
	gitsite = strings.ToLower(strings.TrimSpace(gitsite))
	if ticketID == "" || userID == "" || gitsite == "" {
		return "", false
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.GitOAuthURL), "/")
	if base == "" {
		return "", false
	}
	body, _ := json.Marshal(map[string]string{
		"id": ticketID, "user_id": userID, "gitsite": gitsite,
	})
	req, err := http.NewRequest(http.MethodPost, base+"/api/internal/git-oauth/grant-ticket/consume/", bytes.NewReader(body))
	if err != nil {
		return "", false
	}
	req.Header.Set("Content-Type", "application/json")
	if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
		req.Header.Set("X-GitOauth-Bridge-Secret", sec)
	}
	resp, err := gitOAuthHTTP.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", false
	}
	var out struct {
		OK           bool   `json:"ok"`
		RemoteUserID string `json:"remote_user_id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || !out.OK {
		return "", false
	}
	return strings.TrimSpace(out.RemoteUserID), true
}

// applyGrantTicketToIdentities consumes the Git OAuth grant ticket per site and
// stamps the comment identities. OPT-20260901-028: the COMMENT_GIT_OAUTH_GRANTED
// event uses the same seed-style idempotency key as applyProjectL2SeedToIdentities
// (grant:comment:{commentID}:{userID}:{gitsite}) and carries comment_id in the
// payload — previously the key was the ticket id, which could not be reconciled
// with per-comment seed events during audit / idempotent consumption.
func applyGrantTicketToIdentities(commentID, userID, ticketID string, selections []RepoIdentitySelection) []RepoIdentitySelection {
	if strings.TrimSpace(ticketID) == "" || len(selections) == 0 {
		return selections
	}
	seen := map[string]struct{}{}
	out := selections
	for _, row := range selections {
		site := gitsiteFromRepoURL(row.RepoURL)
		if site == "" {
			continue
		}
		if _, ok := seen[site]; ok {
			continue
		}
		seen[site] = struct{}{}
		remote, ok := consumeGitOAuthGrantTicket(userID, site, ticketID)
		if !ok {
			continue
		}
		out = stampCommentOAuthGrant(out, site, remote)
		idempotencyKey := fmt.Sprintf("grant:comment:%s:%s:%s", commentID, userID, site)
		_ = publishDomainEventFn(nil, "COMMENT_GIT_OAUTH_GRANTED", map[string]interface{}{
			"comment_id": commentID, "user_id": userID, "gitsite": site, "remote_user_id": remote, "via": "grant_ticket",
		}, idempotencyKey)
	}
	return out
}

func grantTicketFromBody(body map[string]interface{}) string {
	if body == nil {
		return ""
	}
	if s := strings.TrimSpace(fmt.Sprint(body["grant_ticket"])); s != "" && s != "<nil>" {
		return s
	}
	return ""
}

func handleInternalMarkCommentGitOAuthGrant(w http.ResponseWriter, r *http.Request, commentID string) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireInternalSecret(r) {
		writeError(w, r, http.StatusForbidden, "forbidden")
		return
	}
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
	commentID = strings.TrimSpace(commentID)
	userID := strings.TrimSpace(body["user_id"])
	gitsite := strings.ToLower(strings.TrimSpace(body["gitsite"]))
	remote := strings.TrimSpace(body["remote_user_id"])
	if commentID == "" || userID == "" || gitsite == "" {
		writeError(w, r, http.StatusBadRequest, "comment_id, user_id, gitsite required")
		return
	}
	var jsonText string
	err = db.QueryRow(`SELECT COALESCE(repo_identities_json,'') FROM task_comments WHERE id=? LIMIT 1`, commentID).Scan(&jsonText)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "comment not found")
		return
	}
	selections, _ := ParseRepoIdentities(repoIdentitiesFromJSONColumn(jsonText))
	if len(selections) == 0 {
		selections = []RepoIdentitySelection{{RepoURL: "", OauthGitsite: gitsite, OauthRemoteUserID: remote, OauthGrantedAt: time.Now().UTC().Format(time.RFC3339)}}
	} else {
		selections = stampCommentOAuthGrant(selections, gitsite, remote)
	}
	if _, err := db.Exec(`UPDATE task_comments SET repo_identities_json=? WHERE id=?`, RepoIdentitiesJSON(selections), commentID); err != nil {
		writeError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	_ = publishDomainEvent(r.Context(), "COMMENT_GIT_OAUTH_GRANTED", map[string]interface{}{
		"comment_id": commentID, "user_id": userID, "gitsite": gitsite, "remote_user_id": remote,
	}, commentID)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}
