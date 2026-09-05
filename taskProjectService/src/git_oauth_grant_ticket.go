package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var consumeGitOAuthGrantTicketFn = consumeGitOAuthGrantTicketLive

func consumeGitOAuthGrantTicket(userID, gitsite, ticketID string) (remoteUserID string, ok bool) {
	return consumeGitOAuthGrantTicketFn(userID, gitsite, ticketID)
}

func grantTicketsFromCreateBody(body map[string]interface{}) []string {
	if body == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		s := strings.TrimSpace(raw)
		if s == "" || s == "<nil>" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(strField(body, "grant_ticket"))
	switch v := body["grant_tickets"].(type) {
	case []interface{}:
		for _, item := range v {
			add(fmt.Sprint(item))
		}
	case []string:
		for _, item := range v {
			add(item)
		}
	}
	return out
}

func uniqueGitsitesFromRepoEntries(entries []gitRepoEntry) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, entry := range entries {
		_, site := extractHostNetloc(entry.URL)
		site = strings.ToLower(strings.TrimSpace(site))
		if site == "" {
			continue
		}
		if _, ok := seen[site]; ok {
			continue
		}
		seen[site] = struct{}{}
		out = append(out, site)
	}
	return out
}

func applyCreateProjectGrantTickets(projectID, companyID, userID string, entries []gitRepoEntry, tickets []string, traceID string) {
	pid := strings.TrimSpace(projectID)
	uid := strings.TrimSpace(userID)
	if pid == "" || uid == "" || len(tickets) == 0 {
		return
	}
	sites := uniqueGitsitesFromRepoEntries(entries)
	if len(sites) == 0 {
		return
	}
	for _, site := range sites {
		if hasProjectGitOAuthGrant(pid, uid, site) {
			continue
		}
		remote := ""
		ok := false
		for _, ticket := range tickets {
			remote, ok = consumeGitOAuthGrantTicket(uid, site, ticket)
			if ok {
				break
			}
		}
		if !ok {
			logWarn("event=project_git_oauth_grant_ticket_unusable project_id="+pid+" user_id="+uid+" gitsite="+site, traceID)
			continue
		}
		if err := upsertProjectGitOAuthGrant(companyID, pid, uid, site, remote); err != nil {
			logError("event=project_git_oauth_grant_upsert_failed project_id="+pid+" gitsite="+site+" err="+err.Error(), traceID)
			continue
		}
		logInfo("event=project_git_oauth_grant_from_ticket project_id="+pid+" user_id="+uid+" gitsite="+site+" via=grant_ticket", traceID)
		publishProjectGitOAuthGranted(pid, uid, site, remote)
		_ = publishDomainEvent(nil, "PROJECT_GIT_OAUTH_GRANTED", map[string]interface{}{
			"project_id": pid, "user_id": uid, "gitsite": site, "remote_user_id": remote, "via": "grant_ticket",
		}, fmt.Sprintf("grant:project:%s:%s:%s", pid, uid, site))
	}
}

func consumeGitOAuthGrantTicketLive(userID, gitsite, ticketID string) (remoteUserID string, ok bool) {
	ticketID = strings.TrimSpace(ticketID)
	userID = strings.TrimSpace(userID)
	gitsite = strings.ToLower(strings.TrimSpace(gitsite))
	if ticketID == "" || userID == "" || gitsite == "" {
		return "", false
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.GitoauthBaseURL), "/")
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
	req.Header.Set("Accept", "application/json")
	if sec := strings.TrimSpace(cfg.GitoauthBridgeSecret); sec != "" {
		req.Header.Set("X-GitOauth-Bridge-Secret", sec)
	}
	if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
		if req.Header.Get("X-GitOauth-Bridge-Secret") == "" {
			req.Header.Set("X-GitOauth-Bridge-Secret", sec)
		}
	}
	resp, err := gitHTTPClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
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
