package main

import (
	"encoding/json"
	"strings"
)

func normalizeDependsOnCommentIDs(raw interface{}) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	add := func(s string) {
		id := strings.TrimSpace(s)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	switch v := raw.(type) {
	case nil:
		return out
	case []string:
		for _, s := range v {
			add(s)
		}
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				add(s)
			}
		}
	case string:
		s := strings.TrimSpace(v)
		if s == "" || s == "[]" {
			return out
		}
		if strings.HasPrefix(s, "[") {
			var arr []string
			if err := json.Unmarshal([]byte(s), &arr); err == nil {
				for _, id := range arr {
					add(id)
				}
				return out
			}
			var anyArr []interface{}
			if err := json.Unmarshal([]byte(s), &anyArr); err == nil {
				for _, item := range anyArr {
					if id, ok := item.(string); ok {
						add(id)
					}
				}
				return out
			}
		}
		for _, part := range strings.Split(s, ",") {
			add(part)
		}
	}
	return out
}

func dependsOnCommentIDsJSON(ids []string) string {
	if len(ids) == 0 {
		return "[]"
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func parseDependsOnCommentIDsJSON(raw string) []string {
	return normalizeDependsOnCommentIDs(strings.TrimSpace(raw))
}
