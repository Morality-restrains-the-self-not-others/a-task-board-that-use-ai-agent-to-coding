package main

import (
	"authz"
	"fmt"
	"net/http"
	"strings"
)

// validateTaskFields validates deliverable/owner/assignees via Go services.
// Progress column validation uses taskProjectService.
// Django /api/internal/taskproject/validate-task-fields/ retired (OPT-052: 2026-07-30).
// deliverable_obj_id, owner_id, assignees are now validated at read time by
// taskProjectService/taskTenantService, not at write time.
func validateTaskFields(tenantID, workspaceID string, body map[string]interface{}) (progressColumnName string, err error) {
	// Validate progress_column_id via taskProjectService (migrated from Django)
	if pcID := strField(body, "progress_column_id"); pcID != "" {
		name, err := validateProgressColumnViaProjectService(tenantID, workspaceID, pcID)
		if err != nil {
			return "", err
		}
		progressColumnName = name
	}
	return progressColumnName, nil
}

// validateProgressColumnViaProjectService calls taskProjectService to validate
// that a progress column ID belongs to the workspace's configured progress system.
func validateProgressColumnViaProjectService(tenantID, workspaceID, progressColumnID string) (string, error) {
	if cfg.ProjectServiceURL == "" {
		return "", nil
	}
	out, status, err := projectServicePost("/api/internal/progress-columns/validate", map[string]interface{}{
		"tenant_id":          tenantID,
		"workspace_id":       workspaceID,
		"progress_column_id": progressColumnID,
	})
	if err != nil {
		return "", fmt.Errorf("进度列验证服务不可用: %w", err)
	}
	if status == 200 {
		return strField(out, "progress_column_name"), nil
	}
	// extract field-level error like Django format
	if msg := fieldError(out, "progress_column_id"); msg != "" {
		return "", fmt.Errorf("%s", msg)
	}
	if msg := strField(out, "error"); msg != "" {
		return "", fmt.Errorf("%s", msg)
	}
	return "", fmt.Errorf("进度列验证失败")
}

func fieldError(out map[string]interface{}, key string) string {
	raw, ok := out[key]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) > 0 {
			return fmt.Sprintf("%v", v[0])
		}
	}
	return ""
}

// resolveUserMember maps user_id → company_member_id + is_admin via taskTenantService.
// Migrated from Django /api/internal/taskproject/resolve-user-member/ (OPT-20260729-024 #8).
func resolveUserMember(tenantID, userID string) (memberID string, isAdmin bool, err error) {
	if strings.TrimSpace(cfg.TaskTenantServiceURL) == "" {
		return "", false, nil
	}
	member, err := tenantResolveMember(tenantID, userID)
	if err != nil {
		return "", false, err
	}
	if member == nil {
		return "", false, fmt.Errorf("member not found")
	}
	memberID = strField(member, "id")
	isAdmin = false
	if v, ok := member["is_admin"].(bool); ok {
		isAdmin = v
	}
	return memberID, isAdmin, nil
}

func canMutateTask(r *http.Request, tenantID, userID string, t *taskRecord) bool {
	if cfg.TaskTenantServiceURL == "" {
		return true
	}
	// v63: 管理员判定迁移至 authz 权限码（task:manage）
	if authz.HasPerm(r, tenantID, authz.PermTaskManage) {
		return true
	}
	memberID, _, err := resolveUserMember(tenantID, userID)
	if err != nil || memberID == "" {
		return false
	}
	if t.OwnerID != "" && t.OwnerID == memberID {
		return true
	}
	for _, aid := range loadAssignees(t.ID) {
		if aid == memberID {
			return true
		}
	}
	return false
}
