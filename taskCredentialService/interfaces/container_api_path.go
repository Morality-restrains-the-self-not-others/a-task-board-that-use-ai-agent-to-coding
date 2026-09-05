package interfaces

import "strings"

// parseContainerAPIPath extracts scope and action from:
// /api/tenant/{tid}/workspace/{wid}/task/{tk}/comment/{cid}/cloud/server-container-token/{action}/
func parseContainerAPIPath(path string) (tenantID, workspaceID, taskID, commentID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 12 {
		return "", "", "", "", "", false
	}
	if parts[0] != "api" || parts[1] != "tenant" || parts[3] != "workspace" || parts[5] != "task" {
		return "", "", "", "", "", false
	}
	if parts[7] != "comment" {
		return "", "", "", "", "", false
	}
	commentID = strings.TrimSpace(parts[8])
	if commentID == "" || commentID == "-" {
		return "", "", "", "", "", false
	}
	if parts[9] != "cloud" || parts[10] != "server-container-token" {
		return "", "", "", "", "", false
	}
	action = strings.Trim(parts[11], "/")
	if action == "" {
		return "", "", "", "", "", false
	}
	return parts[2], parts[4], parts[6], commentID, action, true
}
