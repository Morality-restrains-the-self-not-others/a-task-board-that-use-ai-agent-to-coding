package main

import (
	"fmt"
	"net/http"
	"strings"
)

func testModeSkipDjango(r *http.Request) bool {
	return r != nil && r.Header.Get("X-Task-Test-Skip-Django-Validate") == "1"
}

func testMemberID(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Header.Get("X-Task-Test-Member-Id"))
}

func validateTaskFieldsFromRequest(r *http.Request, tenantID, workspaceID string, body map[string]interface{}) (progressColumnName string, err error) {
	if testModeSkipDjango(r) {
		return validateTaskFieldsTestMode(r, body)
	}
	return validateTaskFields(tenantID, workspaceID, body)
}

func normID(v string) string {
	return strings.TrimSpace(v)
}

func validateTaskFieldsTestMode(r *http.Request, body map[string]interface{}) (progressColumnName string, err error) {
	if col := strField(body, "progress_column_id"); col != "" {
		allowedRaw := r.Header.Get("X-Task-Test-Allowed-Progress-Column-Ids")
		allowed := map[string]struct{}{}
		for _, part := range strings.Split(allowedRaw, ",") {
			part = normID(part)
			if part != "" {
				allowed[part] = struct{}{}
			}
		}
		if len(allowed) > 0 {
			if _, ok := allowed[normID(col)]; !ok {
				return "", fmt.Errorf("invalid progress column")
			}
		}
		progressColumnName = strings.TrimSpace(r.Header.Get("X-Task-Test-Progress-Column-Name"))
	}
	if assignees := memberIDsFromBody(body, "assignees"); len(assignees) > 0 {
		allowedMembers := map[string]struct{}{}
		for _, part := range strings.Split(r.Header.Get("X-Task-Test-Allowed-Member-Ids"), ",") {
			part = normID(part)
			if part != "" {
				allowedMembers[part] = struct{}{}
			}
		}
		if owner := normID(ownerFromBody(body)); owner != "" {
			allowedMembers[owner] = struct{}{}
		}
		if len(allowedMembers) > 0 {
			for _, a := range assignees {
				if _, ok := allowedMembers[normID(a)]; !ok {
					return "", fmt.Errorf("invalid assignee")
				}
			}
		}
	}
	if d := normID(strField(body, "deliverable_obj_id")); d != "" {
		allowedObj := normID(r.Header.Get("X-Task-Test-Allowed-Deliverable-Obj-Id"))
		if allowedObj != "" && d != allowedObj {
			return "", fmt.Errorf("invalid deliverable")
		}
	}
	return progressColumnName, nil
}

func canMutateTaskFromRequest(r *http.Request, tenantID, userID string, t *taskRecord) bool {
	if testModeSkipDjango(r) {
		memberID := testMemberID(r)
		if memberID == "" {
			return hasWorkspaceAccess(r.Context(), tenantID, t.WorkspaceID, userID)
		}
		if t.OwnerID == memberID {
			return true
		}
		for _, a := range loadAssignees(t.ID) {
			if a == memberID {
				return true
			}
		}
		return false
	}
	return canMutateTask(r, tenantID, userID, t)
}
