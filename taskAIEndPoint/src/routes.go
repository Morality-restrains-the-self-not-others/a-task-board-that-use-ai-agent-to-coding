package main

import "strings"

type llmRouteMatch struct {
	TenantID    string
	WorkspaceID string
	TaskID      string
	Provider    string
	SubPath     string
}

func parseLlmProxyPath(path string) (llmRouteMatch, bool) {
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	// api/tenant/{t}/workspace/{w}/task/{task}/llm/{provider}/v1/...
	if len(parts) < 10 {
		return llmRouteMatch{}, false
	}
	if parts[0] != "api" || parts[1] != "tenant" || parts[3] != "workspace" || parts[5] != "task" || parts[7] != "llm" {
		return llmRouteMatch{}, false
	}
	if parts[8] == "" || parts[9] != "v1" {
		return llmRouteMatch{}, false
	}
	subPath := ""
	if len(parts) > 10 {
		subPath = strings.Join(parts[10:], "/")
	}
	return llmRouteMatch{
		TenantID:    parts[2],
		WorkspaceID: parts[4],
		TaskID:      parts[6],
		Provider:    parts[8],
		SubPath:     subPath,
	}, true
}
