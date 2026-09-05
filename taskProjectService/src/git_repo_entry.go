package main

import (
	"fmt"
	"regexp"
	"strings"
)

var cloneAliasSanitizeRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// sanitizeCloneAlias mirrors trae-agent directory-name sanitization for user aliases.
func sanitizeCloneAlias(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = cloneAliasSanitizeRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-._")
	return s
}

type gitRepoEntry struct {
	URL        string
	CloneAlias string
}

func strFromAny(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// parseGitRepoItems accepts legacy string[] or [{url, clone_alias}] mixed lists.
func parseGitRepoItems(raw []interface{}) []gitRepoEntry {
	out := make([]gitRepoEntry, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		url := ""
		alias := ""
		switch v := item.(type) {
		case string:
			url = strings.TrimSpace(v)
		case map[string]interface{}:
			url = strings.TrimSpace(strFromAny(v["url"]))
			if url == "" {
				url = strings.TrimSpace(strFromAny(v["repo_url"]))
			}
			alias = sanitizeCloneAlias(strFromAny(v["clone_alias"]))
			if alias == "" {
				alias = sanitizeCloneAlias(strFromAny(v["alias"]))
			}
		default:
			url = strings.TrimSpace(strFromAny(v))
		}
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		out = append(out, gitRepoEntry{URL: url, CloneAlias: alias})
	}
	return out
}

// duplicateCloneAliasError returns a user-facing error when two non-empty aliases collide
// (case-insensitive). Empty aliases are allowed and do not collide.
func duplicateCloneAliasError(entries []gitRepoEntry) string {
	seen := map[string]struct{}{}
	for _, e := range entries {
		alias := strings.ToLower(strings.TrimSpace(e.CloneAlias))
		if alias == "" {
			continue
		}
		if _, ok := seen[alias]; ok {
			return "同一项目内仓库别名不能重复"
		}
		seen[alias] = struct{}{}
	}
	return ""
}
