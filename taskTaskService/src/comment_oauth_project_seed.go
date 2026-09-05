package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type taskLinkedProjectRepo struct {
	ProjectID   string
	RepoAddress string
}

var lookupProjectGitOAuthGrantFn = lookupProjectGitOAuthGrantLive

func lookupProjectGitOAuthGrant(projectID, userID, gitsite string) (remoteUserID string, ok bool) {
	return lookupProjectGitOAuthGrantFn(projectID, userID, gitsite)
}

func lookupProjectGitOAuthGrantLive(projectID, userID, gitsite string) (remoteUserID string, ok bool) {
	projectID = strings.TrimSpace(projectID)
	userID = strings.TrimSpace(userID)
	gitsite = strings.ToLower(strings.TrimSpace(gitsite))
	if projectID == "" || userID == "" || gitsite == "" {
		return "", false
	}
	if strings.TrimSpace(cfg.ProjectServiceURL) == "" {
		return "", false
	}
	q := url.Values{}
	q.Set("project_id", projectID)
	q.Set("user_id", userID)
	q.Set("gitsite", gitsite)
	path := "/api/internal/projects/git-oauth-grant/?" + q.Encode()
	status, raw, err := projectRequest(context.Background(), http.MethodGet, path, "", nil)
	if err != nil {
		log.Printf("[taskTaskService] event=auto_run_project_l2_lookup_failed project_id=%s user_id=%s gitsite=%s err=%v",
			projectID, userID, gitsite, err)
		return "", false
	}
	if status != http.StatusOK {
		preview := string(raw)
		if len(preview) > 180 {
			preview = preview[:180]
		}
		log.Printf("[taskTaskService] event=auto_run_project_l2_lookup_failed project_id=%s user_id=%s gitsite=%s status=%d body=%s",
			projectID, userID, gitsite, status, preview)
		return "", false
	}
	var out struct {
		HasGrant     bool   `json:"has_grant"`
		RemoteUserID string `json:"remote_user_id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || !out.HasGrant {
		return "", false
	}
	return strings.TrimSpace(out.RemoteUserID), true
}

func loadTaskLinkedProjectRepos(taskID string) []taskLinkedProjectRepo {
	taskID = strings.TrimSpace(taskID)
	if db == nil || taskID == "" {
		return nil
	}
	rows, err := db.Query(
		`SELECT COALESCE(project_id,''), COALESCE(repo_address,'') FROM task_projects WHERE task_id=? ORDER BY created_at, id`,
		taskID,
	)
	if err != nil {
		log.Printf("[taskTaskService] event=auto_run_project_l2_links_failed task_id=%s err=%v", taskID, err)
		return nil
	}
	defer rows.Close()
	out := []taskLinkedProjectRepo{}
	for rows.Next() {
		var pid, addr string
		if err := rows.Scan(&pid, &addr); err != nil {
			continue
		}
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		out = append(out, taskLinkedProjectRepo{ProjectID: pid, RepoAddress: strings.TrimSpace(addr)})
	}
	return out
}

func uniqueNonEmpty(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func projectIDsForRepoURL(links []taskLinkedProjectRepo, repoURL string) []string {
	repoURL = strings.TrimSpace(repoURL)
	site := gitsiteFromRepoURL(repoURL)
	var exact, bySite, all []string
	for _, row := range links {
		all = append(all, row.ProjectID)
		if repoURL != "" && strings.TrimSpace(row.RepoAddress) == repoURL {
			exact = append(exact, row.ProjectID)
			continue
		}
		if site != "" && gitsiteFromRepoURL(row.RepoAddress) == site {
			bySite = append(bySite, row.ProjectID)
		}
	}
	if got := uniqueNonEmpty(exact); len(got) > 0 {
		return got
	}
	if got := uniqueNonEmpty(bySite); len(got) > 0 {
		return got
	}
	return uniqueNonEmpty(all)
}

func prepareCommentOAuthIdentities(commentID, userID, taskID, ticket string, selections []RepoIdentitySelection) []RepoIdentitySelection {
	out := applyGrantTicketToIdentities(commentID, userID, ticket, selections)
	out = applyProjectL2SeedToIdentities(commentID, userID, taskID, out)
	return applyForkSourceCommentL2SeedToIdentities(commentID, userID, taskID, out)
}

func prepareAutoRunRepoIdentities(commentID, userID, taskID, ticket string, selections []RepoIdentitySelection) []RepoIdentitySelection {
	return stampImplicitCommentOAuthGitsite(prepareCommentOAuthIdentities(commentID, userID, taskID, ticket, selections))
}

func applyProjectL2SeedToIdentities(commentID, userID, taskID string, selections []RepoIdentitySelection) []RepoIdentitySelection {
	if len(selections) == 0 {
		return selections
	}
	userID = strings.TrimSpace(userID)
	commentID = strings.TrimSpace(commentID)
	if userID == "" {
		return selections
	}
	links := loadTaskLinkedProjectRepos(taskID)
	if len(links) == 0 {
		return selections
	}
	out := make([]RepoIdentitySelection, len(selections))
	copy(out, selections)
	for i := range out {
		if strings.TrimSpace(out[i].OauthGitsite) != "" {
			continue
		}
		site := gitsiteFromRepoURL(out[i].RepoURL)
		if !isOAuthCapableGitsite(site) {
			continue
		}
		var seeded bool
		var remote string
		var hitProject string
		for _, pid := range projectIDsForRepoURL(links, out[i].RepoURL) {
			got, ok := lookupProjectGitOAuthGrant(pid, userID, site)
			if !ok {
				continue
			}
			remote = got
			hitProject = pid
			seeded = true
			break
		}
		if !seeded {
			continue
		}
		nowStamp := stampCommentOAuthGrant([]RepoIdentitySelection{out[i]}, site, remote)
		if len(nowStamp) == 1 {
			out[i] = nowStamp[0]
		}
		idempotencyKey := fmt.Sprintf("grant:comment:%s:%s:%s", commentID, userID, site)
		_ = publishDomainEvent(nil, "COMMENT_GIT_OAUTH_GRANTED", map[string]interface{}{
			"comment_id":     commentID,
			"user_id":        userID,
			"gitsite":        site,
			"remote_user_id": remote,
			"project_id":     hitProject,
			"via":            "project_l2_seed",
		}, idempotencyKey)
		log.Printf("[taskTaskService] event=auto_run_project_l2_seed comment_id=%s task_id=%s user_id=%s gitsite=%s project_id=%s",
			commentID, taskID, userID, site, hitProject)
	}
	return out
}
