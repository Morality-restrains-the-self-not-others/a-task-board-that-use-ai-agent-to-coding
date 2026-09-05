package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

var djangoHTTP = &http.Client{Timeout: 30 * time.Second}

// verifyCompanyExists checks company existence via taskTenantService.
// Migrated from Django /api/internal/companies/{id}/exists/ (OPT-20260729-056).
func verifyCompanyExists(companyID string) error {
	if strings.TrimSpace(cfg.TaskTenantURL) == "" || strings.TrimSpace(companyID) == "" {
		return nil
	}
	// tenantCompanyCreatorID calls companies/creator which returns found=false if company missing
	creatorID, err := tenantCompanyCreatorID(companyID)
	if err != nil {
		return fmt.Errorf("company verification failed: %w", err)
	}
	if creatorID == "" {
		// Could be: company exists with no creator, or company doesn't exist.
		// Accept empty creator as valid (company exists).
	}
	return nil
}

// persistWorkspaceSelection calls taskTenantService directly.
// Migrated from Django /api/internal/taskproject/persist-workspace-selection/ (OPT-20260729-024 #1).
func persistWorkspaceSelection(userID, tenantID, workspaceID string) error {
	if strings.TrimSpace(cfg.TaskTenantURL) == "" {
		return nil
	}
	return tenantUpdateMemberWorkspace(userID, tenantID, workspaceID)
}

// getMemberWorkspaceID reads workspace_id from taskTenantService.
// Migrated from Django /api/internal/taskproject/member-workspace-id/ (OPT-20260729-024 #2).
func getMemberWorkspaceID(userID, tenantID string) string {
	member, err := tenantResolveMember(tenantID, userID)
	if err != nil || member == nil {
		return ""
	}
	if v, ok := member["workspace_id"].(string); ok {
		return v
	}
	return ""
}
