package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type commentOAuthGrantRow struct {
	Gitsite      string
	RemoteUserID string
}

func commentIdentitiesNeedOAuthSeed(selections []RepoIdentitySelection) bool {
	for _, row := range selections {
		site := gitsiteFromRepoURL(row.RepoURL)
		if isOAuthCapableGitsite(site) && strings.TrimSpace(row.OauthGitsite) == "" {
			return true
		}
	}
	return false
}

func commentOAuthL2Changed(before, after []RepoIdentitySelection) bool {
	if len(before) != len(after) {
		return true
	}
	for i := range after {
		if strings.TrimSpace(before[i].OauthGitsite) != strings.TrimSpace(after[i].OauthGitsite) ||
			strings.TrimSpace(before[i].OauthRemoteUserID) != strings.TrimSpace(after[i].OauthRemoteUserID) {
			return true
		}
	}
	return false
}

func loadSameUserCommentOAuthGrants(taskID, userID string) []commentOAuthGrantRow {
	taskID = strings.TrimSpace(taskID)
	userID = strings.TrimSpace(userID)
	if db == nil || taskID == "" || userID == "" {
		return nil
	}
	rows, err := db.Query(
		`SELECT COALESCE(repo_identities_json,'') FROM task_comments WHERE task_id=? AND created_by_id=?`,
		taskID, userID,
	)
	if err != nil {
		log.Printf("[taskTaskService] event=fork_source_l2_load_failed task_id=%s user_id=%s err=%v", taskID, userID, err)
		return nil
	}
	defer rows.Close()
	seen := map[string]struct{}{}
	out := []commentOAuthGrantRow{}
	for rows.Next() {
		var jsonText string
		if err := rows.Scan(&jsonText); err != nil {
			continue
		}
		sels, _ := ParseRepoIdentities(repoIdentitiesFromJSONColumn(jsonText))
		for _, row := range sels {
			site := strings.ToLower(strings.TrimSpace(row.OauthGitsite))
			remote := strings.TrimSpace(row.OauthRemoteUserID)
			if site == "" || remote == "" {
				continue
			}
			if _, ok := seen[site]; ok {
				continue
			}
			seen[site] = struct{}{}
			out = append(out, commentOAuthGrantRow{Gitsite: site, RemoteUserID: remote})
		}
	}
	return out
}

func stampMissingCommentOAuthGrant(selections []RepoIdentitySelection, gitsite, remoteUserID string) ([]RepoIdentitySelection, bool) {
	site := strings.ToLower(strings.TrimSpace(gitsite))
	if site == "" || len(selections) == 0 {
		return selections, false
	}
	out := make([]RepoIdentitySelection, len(selections))
	copy(out, selections)
	changed := false
	remote := strings.TrimSpace(remoteUserID)
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range out {
		if gitsiteFromRepoURL(out[i].RepoURL) != site {
			continue
		}
		if strings.TrimSpace(out[i].OauthGitsite) != "" {
			continue
		}
		out[i].OauthGitsite = site
		out[i].OauthRemoteUserID = remote
		out[i].OauthGrantedAt = now
		changed = true
	}
	return out, changed
}

func applyForkSourceCommentL2SeedToIdentities(commentID, userID, taskID string, selections []RepoIdentitySelection) []RepoIdentitySelection {
	if len(selections) == 0 || !commentIdentitiesNeedOAuthSeed(selections) {
		return selections
	}
	userID = strings.TrimSpace(userID)
	commentID = strings.TrimSpace(commentID)
	taskID = strings.TrimSpace(taskID)
	if userID == "" || taskID == "" {
		return selections
	}
	t, err := loadTask(taskID)
	if err != nil || t == nil {
		return selections
	}
	sourceID := strings.TrimSpace(t.ForkFromID)
	if sourceID == "" {
		return selections
	}
	grants := loadSameUserCommentOAuthGrantsWalkingForkChain(sourceID, userID)
	if len(grants) == 0 {
		log.Printf("[taskTaskService] event=fork_source_l2_none comment_id=%s task_id=%s fork_from=%s user_id=%s",
			commentID, taskID, sourceID, userID)
		return selections
	}
	out := selections
	for _, g := range grants {
		next, changed := stampMissingCommentOAuthGrant(out, g.Gitsite, g.RemoteUserID)
		if !changed {
			continue
		}
		out = next
		idempotencyKey := fmt.Sprintf("grant:comment:%s:%s:%s", commentID, userID, g.Gitsite)
		_ = publishDomainEventFn(nil, "COMMENT_GIT_OAUTH_GRANTED", map[string]interface{}{
			"comment_id":        commentID,
			"user_id":           userID,
			"gitsite":           g.Gitsite,
			"remote_user_id":    g.RemoteUserID,
			"fork_from_task_id": sourceID,
			"via":               "fork_source_comment_l2_seed",
		}, idempotencyKey)
		log.Printf("[taskTaskService] event=fork_source_l2_seed comment_id=%s task_id=%s fork_from=%s user_id=%s gitsite=%s",
			commentID, taskID, sourceID, userID, g.Gitsite)
	}
	return out
}

func repairCommentOAuthL2FromForkAndProject(taskID, commentID, userID string, selections []RepoIdentitySelection) []RepoIdentitySelection {
	out := applyProjectL2SeedToIdentities(commentID, userID, taskID, selections)
	out = applyForkSourceCommentL2SeedToIdentities(commentID, userID, taskID, out)
	return stampImplicitCommentOAuthGitsite(out)
}

const maxForkSourceL2WalkDepth = 16

func loadSameUserCommentOAuthGrantsWalkingForkChain(startTaskID, userID string) []commentOAuthGrantRow {
	seen := map[string]struct{}{}
	taskID := strings.TrimSpace(startTaskID)
	for depth := 0; depth < maxForkSourceL2WalkDepth && taskID != ""; depth++ {
		if _, dup := seen[taskID]; dup {
			break
		}
		seen[taskID] = struct{}{}
		if grants := loadSameUserCommentOAuthGrants(taskID, userID); len(grants) > 0 {
			return grants
		}
		t, err := loadTask(taskID)
		if err != nil || t == nil {
			break
		}
		taskID = strings.TrimSpace(t.ForkFromID)
	}
	return nil
}
