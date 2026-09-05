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

	"taskGitOauth/domain"
	"taskGitOauth/infrastructure"
)

func oauthGrantFromQuery(r *http.Request) (kind, id, repoURL string) {
	if r == nil {
		return "", "", ""
	}
	q := r.URL.Query()
	return strings.TrimSpace(q.Get("grant_kind")), strings.TrimSpace(q.Get("grant_id")), strings.TrimSpace(q.Get("repo_url"))
}

func applyOAuthGrantQuery(st *infrastructure.OAuthBrowserState, r *http.Request) {
	if st == nil || r == nil {
		return
	}
	kind, id, repoURL := oauthGrantFromQuery(r)
	if kind != "" {
		st.GrantKind = kind
	}
	if id != "" {
		st.GrantID = id
	}
	if repoURL != "" {
		st.RepoURL = repoURL
	}
}

func appendQueryParam(path, key, value string) string {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
		return path
	}
	u, err := url.Parse(path)
	if err != nil || u == nil {
		joiner := "?"
		if strings.Contains(path, "?") {
			joiner = "&"
		}
		return path + joiner + url.QueryEscape(key) + "=" + url.QueryEscape(value)
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

func (a *App) finishResourceGrant(uid, remoteUserID string, st *infrastructure.OAuthBrowserState, nextPath string) (outPath, ticket string) {
	outPath = nextPath
	if a == nil || strings.TrimSpace(uid) == "" {
		return outPath, ""
	}
	kind := domain.GrantKindPending
	grantID := ""
	repoURL := ""
	if st != nil {
		if k := strings.TrimSpace(st.GrantKind); k != "" {
			kind = k
		}
		grantID = strings.TrimSpace(st.GrantID)
		repoURL = strings.TrimSpace(st.RepoURL)
	}
	gitsite := domain.GitsiteFromRepoURL(repoURL)
	if gitsite == "" {
		return outPath, ""
	}
	switch kind {
	case domain.GrantKindProject:
		if grantID != "" {
			a.markProjectGitOAuthGrant(grantID, uid, gitsite, remoteUserID)
			return outPath, ""
		}
	case domain.GrantKindComment:
		if grantID != "" {
			a.markCommentGitOAuthGrant(grantID, uid, gitsite, remoteUserID)
			return outPath, ""
		}
	}
	id, err := a.DB.IssueGrantTicket(uid, gitsite, remoteUserID, 0)
	if err != nil {
		logWarn("issue grant ticket: %v", err)
		return outPath, ""
	}
	return appendQueryParam(nextPath, "grant_ticket", id), id
}

func (a *App) markProjectGitOAuthGrant(projectID, userID, gitsite, remoteUserID string) {
	base := strings.TrimRight(strings.TrimSpace(a.Cfg.TaskProjectServiceURL), "/")
	if base == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{
		"project_id":     projectID,
		"user_id":        userID,
		"gitsite":        gitsite,
		"remote_user_id": remoteUserID,
	})
	endpoint := base + "/api/internal/projects/git-oauth-grant/"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		logWarn("mark project grant request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if sec := strings.TrimSpace(a.Cfg.DjangoInternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logWarn("mark project grant: %v", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		logWarn("mark project grant http %d", resp.StatusCode)
		return
	}
	publishDomainEvent(a.Cfg, "PROJECT_GIT_OAUTH_GRANTED", map[string]any{
		"project_id": projectID, "user_id": userID, "gitsite": gitsite, "remote_user_id": remoteUserID,
	}, domain.GrantIdempotencyKey(domain.GrantKindProject, projectID, userID, gitsite))
}

func (a *App) markCommentGitOAuthGrant(commentID, userID, gitsite, remoteUserID string) {
	base := strings.TrimRight(strings.TrimSpace(a.Cfg.TaskTaskServiceURL), "/")
	if base == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{
		"comment_id":     commentID,
		"user_id":        userID,
		"gitsite":        gitsite,
		"remote_user_id": remoteUserID,
	})
	endpoint := fmt.Sprintf("%s/api/internal/comments/%s/git-oauth-grant/", base, url.PathEscape(commentID))
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		logWarn("mark comment grant request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if sec := strings.TrimSpace(a.Cfg.DjangoInternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logWarn("mark comment grant: %v", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		logWarn("mark comment grant http %d", resp.StatusCode)
		return
	}
	publishDomainEvent(a.Cfg, "COMMENT_GIT_OAUTH_GRANTED", map[string]any{
		"comment_id": commentID, "user_id": userID, "gitsite": gitsite, "remote_user_id": remoteUserID,
	}, domain.GrantIdempotencyKey(domain.GrantKindComment, commentID, userID, gitsite))
}
