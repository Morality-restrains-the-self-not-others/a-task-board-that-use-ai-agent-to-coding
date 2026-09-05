package main

import (
	"fmt"
	"strings"
)

type commentOAuthGrant struct {
	UserID  string
	Gitsite string
}

func commentOAuthGrantsFromCompact(raw any) ([]commentOAuthGrant, bool) {
	arr, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	var out []commentOAuthGrant
	seen := map[string]bool{}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		userID := strings.TrimSpace(fmt.Sprintf("%v", m["user_id"]))
		if userID == "" || userID == "<nil>" {
			userID = commentAuthorUserID(m)
		}
		site := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", m["gitsite"])))
		if site == "" || site == "<nil>" {
			continue
		}
		key := userID + "\x00" + site
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, commentOAuthGrant{UserID: userID, Gitsite: site})
	}
	return out, true
}

func commentOAuthGrantsFromTaskBody(body map[string]any) []commentOAuthGrant {
	if body == nil {
		return nil
	}
	if grants, ok := commentOAuthGrantsFromCompact(body["comment_oauth_grants"]); ok {
		return grants
	}
	raw, ok := body["comments"].([]any)
	if !ok || len(raw) == 0 {
		return nil
	}
	var out []commentOAuthGrant
	seen := map[string]bool{}
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		userID := commentAuthorUserID(m)
		idents, _ := m["repo_identities"].([]any)
		for _, ident := range idents {
			row, ok := ident.(map[string]any)
			if !ok {
				continue
			}
			site := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", row["oauth_gitsite"])))
			if site == "" || site == "<nil>" {
				continue
			}
			key := userID + "\x00" + site
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, commentOAuthGrant{UserID: userID, Gitsite: site})
		}
	}
	return out
}

func commentAuthorUserID(comment map[string]any) string {
	if comment == nil {
		return ""
	}
	if created, ok := comment["created_by"].(map[string]any); ok {
		id := strings.TrimSpace(fmt.Sprintf("%v", created["id"]))
		if id != "" && id != "<nil>" {
			return id
		}
	}
	id := strings.TrimSpace(fmt.Sprintf("%v", comment["created_by_id"]))
	if id == "<nil>" {
		return ""
	}
	return id
}

func userHasCommentOAuthGrant(grants []commentOAuthGrant, userID, gitsite string) bool {
	uid := strings.TrimSpace(userID)
	site := strings.ToLower(strings.TrimSpace(gitsite))
	if uid == "" || site == "" {
		return false
	}
	for _, g := range grants {
		if strings.TrimSpace(g.UserID) == uid && strings.EqualFold(g.Gitsite, site) {
			return true
		}
	}
	return false
}

func commentGrantMissingDetail(gitsite string) string {
	site := strings.TrimSpace(gitsite)
	if site == "" {
		site = "git"
	}
	return "该任务评论尚未完成 Git OAuth 使用授权（gitsite=" + site + "），请重新完成授权后再推送"
}
