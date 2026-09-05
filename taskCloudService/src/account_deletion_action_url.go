package main

import (
	"fmt"
	"strings"
)

func cloudResourceActionURL(tenantID, workspaceID, taskID string) string {
	tid := strings.TrimSpace(tenantID)
	wid := strings.TrimSpace(workspaceID)
	tk := strings.TrimSpace(taskID)
	if tid == "" || wid == "" || tk == "" {
		return "/profile/"
	}
	return fmt.Sprintf("/tenant/%s/workspace/%s/task-detail/%s/", tid, wid, tk)
}
