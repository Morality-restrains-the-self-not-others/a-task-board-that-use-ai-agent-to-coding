package saas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// taskProjectURL returns the taskProjectService internal URL.
func taskProjectURL() string {
	if v := strings.TrimSpace(os.Getenv("TASK_PROJECT_SERVICE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://127.0.0.1:8016"
}

// deliverableSystem is a single deliverable system entry from taskProjectService.
type deliverableSystem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	IsSystem  bool   `json:"is_system"`
}

// setDefaultDeliverableDirect calls taskProjectService directly to set the
// tenant's default deliverable system, bypassing Django.
func (r *Repository) setDefaultDeliverableDirect(companyID string) error {
	base := taskProjectURL()

	// 1. List deliverable systems for the tenant
	systems, err := r.listDeliverableSystems(base, companyID)
	if err != nil {
		return fmt.Errorf("list deliverable systems: %w", err)
	}

	// 2. Check if a default already exists
	for _, s := range systems {
		if s.IsDefault {
			return nil // Already set
		}
	}

	// 3. Find the system-level default deliverable
	var systemDefaultID string
	for _, s := range systems {
		if s.IsSystem {
			systemDefaultID = s.ID
			break
		}
	}
	if systemDefaultID == "" {
		return fmt.Errorf("no system default deliverable found for tenant %s", companyID)
	}

	// 4. Set as default
	return r.setDefaultDeliverableSystem(base, companyID, systemDefaultID)
}

// listDeliverableSystems calls GET /api/projects/tenant_id/{tid}/deliverable-systems/
func (r *Repository) listDeliverableSystems(base, tenantID string) ([]deliverableSystem, error) {
	url := fmt.Sprintf("%s/api/projects/deliverable-systems/tenant_id/%s/", base, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list deliverable systems status %d", resp.StatusCode)
	}
	var systems []deliverableSystem
	if err := json.NewDecoder(resp.Body).Decode(&systems); err != nil {
		return nil, err
	}
	return systems, nil
}

// setDefaultDeliverableSystem calls POST /api/projects/tenant_id/{tid}/deliverable-systems/{id}/set-default/
func (r *Repository) setDefaultDeliverableSystem(base, tenantID, systemID string) error {
	url := fmt.Sprintf("%s/api/projects/deliverable-systems/%s/set-default/tenant_id/%s/", base, systemID, tenantID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("set default deliverable status %d", resp.StatusCode)
	}
	return nil
}

// setDefaultProgressDirect calls taskProjectService directly to ensure the
// tenant has a default progress system, bypassing Django.
func (r *Repository) setDefaultProgressDirect(companyID string) error {
	base := taskProjectURL()
	checkURL := fmt.Sprintf("%s/api/projects/settings/default-progress-system/tenant_id/%s/", base, companyID)

	// 1. Check if a default progress system already exists
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Tenant-Id", companyID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("get default progress system status %d", resp.StatusCode)
	}
	var body struct {
		ProgressSystem *json.RawMessage `json:"progress_system"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	if body.ProgressSystem != nil {
		return nil // Already set
	}

	// 2. No default set — find the system default progress system (is_default=1)
	listURL := fmt.Sprintf("%s/api/system-admin/progress-systems/", base)
	listResp, err := r.client.Get(listURL)
	if err != nil {
		return fmt.Errorf("list progress systems: %w", err)
	}
	defer listResp.Body.Close()
	var listBody struct {
		ProgressSystems []struct {
			ID        string `json:"id"`
			IsDefault bool   `json:"is_default"`
		} `json:"project_progress_systems"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listBody); err != nil {
		return fmt.Errorf("decode progress systems list: %w", err)
	}
	var defaultSystemID string
	for _, ps := range listBody.ProgressSystems {
		if ps.IsDefault {
			defaultSystemID = ps.ID
			break
		}
	}
	if defaultSystemID == "" && len(listBody.ProgressSystems) > 0 {
		defaultSystemID = listBody.ProgressSystems[0].ID
	}
	if defaultSystemID == "" {
		return fmt.Errorf("no default progress system found for tenant %s", companyID)
	}

	// 3. Set the default progress system for this tenant
	postBody := map[string]string{"system_id": defaultSystemID}
	postJSON, _ := json.Marshal(postBody)
	postReq, err := http.NewRequest(http.MethodPost, checkURL, strings.NewReader(string(postJSON)))
	if err != nil {
		return err
	}
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("X-Auth-Tenant-Id", companyID)
	postReq.Header.Set("X-Auth-User-Id", "internal")
	postResp, err := r.client.Do(postReq)
	if err != nil {
		return fmt.Errorf("set default progress system: %w", err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode >= 400 {
		return fmt.Errorf("set default progress system status %d", postResp.StatusCode)
	}
	return nil
}

// workspaceSummary is a minimal workspace entry from taskProjectService list.
type workspaceSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	CreatedAt string `json:"created_at"`
}

// CreatedWorkspace is the result of creating a default workspace.
type CreatedWorkspace struct {
	WorkspaceID    string
	WorkspaceName  string
	CreatedAt      time.Time
	AlreadyExisted bool
}

// createDefaultWorkspaceDirect calls taskProjectService directly to ensure
// a default workspace exists for the company, bypassing Django.
func (r *Repository) createDefaultWorkspaceDirect(companyID, creatorID, companyName string) (*CreatedWorkspace, error) {
	base := taskProjectURL()

	// 1. Check if default workspace already exists
	existing, err := r.findDefaultWorkspace(base, companyID)
	if err == nil && existing != nil {
		createdAt := time.Now().UTC()
		if existing.CreatedAt != "" {
			if t, e := time.Parse(time.RFC3339, existing.CreatedAt); e == nil {
				createdAt = t
			}
		}
		return &CreatedWorkspace{WorkspaceID: existing.ID, WorkspaceName: existing.Name, CreatedAt: createdAt, AlreadyExisted: true}, nil
	}

	// 2. Find the tenant's default deliverable system
	deliverableID, deliverableFrom, err := r.findTenantDefaultDeliverable(base, companyID)
	if err != nil {
		return nil, fmt.Errorf("find default deliverable: %w", err)
	}

	// 3. Name the default workspace after the company (OPT-20260827-020),
	// so cross-tenant lists don't all show「用户的工作空间」. Idempotent:
	// an existing is_default workspace is never renamed (step 1 returns early).
	workspaceName := "用户的工作空间"
	if companyName != "" {
		workspaceName = companyName + " 的工作空间"
	}
	description := "系统自动创建的工作空间"

	// 4. Create the workspace
	created, err := r.createWorkspace(base, companyID, workspaceName, description, deliverableID, deliverableFrom)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", err)
	}

	// 5. Try to set workspace progress system (best-effort)
	progID, err := r.findTenantDefaultProgress(base, companyID)
	if err == nil && progID != "" {
		_ = r.setWorkspaceProgressSystem(base, companyID, created.ID, progID)
	}

	createdAt := time.Now().UTC()
	if created.CreatedAt != "" {
		if t, e := time.Parse(time.RFC3339, created.CreatedAt); e == nil {
			createdAt = t
		}
	}
	return &CreatedWorkspace{WorkspaceID: created.ID, WorkspaceName: created.Name, CreatedAt: createdAt, AlreadyExisted: false}, nil
}

// findDefaultWorkspace looks for an existing default workspace for the tenant.
func (r *Repository) findDefaultWorkspace(base, tenantID string) (*workspaceSummary, error) {
	workspaces, err := r.listWorkspaces(base, tenantID)
	if err != nil {
		return nil, err
	}
	for _, ws := range workspaces {
		if ws.IsDefault {
			return &ws, nil
		}
	}
	return nil, nil // not found, not an error
}

// listWorkspaces calls GET /api/projects/tenant_id/{tid}/workspaces/
func (r *Repository) listWorkspaces(base, tenantID string) ([]workspaceSummary, error) {
	url := fmt.Sprintf("%s/api/projects/workspaces/tenant_id/%s/", base, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list workspaces status %d", resp.StatusCode)
	}
	var workspaces []workspaceSummary
	if err := json.NewDecoder(resp.Body).Decode(&workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

// findTenantDefaultDeliverable returns (deliverableSystemID, from, error).
func (r *Repository) findTenantDefaultDeliverable(base, tenantID string) (string, string, error) {
	systems, err := r.listDeliverableSystems(base, tenantID)
	if err != nil {
		return "", "", err
	}
	for _, s := range systems {
		if s.IsDefault {
			return s.ID, "company", nil
		}
	}
	for _, s := range systems {
		if s.IsSystem {
			return s.ID, "system", nil
		}
	}
	return "", "", fmt.Errorf("no default or system deliverable for tenant %s", tenantID)
}

// findTenantDefaultProgress returns the default progress system ID if configured.
func (r *Repository) findTenantDefaultProgress(base, tenantID string) (string, error) {
	url := fmt.Sprintf("%s/api/projects/settings/default-progress-system/tenant_id/%s/", base, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	var body struct {
		ProgressSystem *struct {
			ID string `json:"id"`
		} `json:"progress_system"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.ProgressSystem != nil {
		return body.ProgressSystem.ID, nil
	}
	return "", nil
}

// createWorkspace calls POST /api/projects/tenant_id/{tid}/workspaces/
func (r *Repository) createWorkspace(base, tenantID, name, description, deliverableID, deliverableFrom string) (*workspaceSummary, error) {
	body := map[string]interface{}{
		"name":                    name,
		"description":             description,
		"deliverable_system_id":   deliverableID,
		"deliverable_system_from": deliverableFrom,
		"is_default":              true,
		"task_archive_tier":       "7d",
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/projects/workspaces/tenant_id/%s/", base, tenantID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respRaw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create workspace status %d: %s", resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	var ws workspaceSummary
	if err := json.NewDecoder(resp.Body).Decode(&ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

// handleWorkspaceCreatedDirect sets up workspace admins and progress bindings
// via taskProjectService directly, bypassing Django.
func (r *Repository) handleWorkspaceCreatedDirect(workspaceID, workspaceName, companyID, createdBy string) error {
	base := taskProjectURL()
	wid := workspaceID

	// 1. Verify workspace exists via internal API
	if err := r.getWorkspaceByID(base, wid); err != nil {
		return fmt.Errorf("workspace %s not found: %w", wid, err)
	}

	// 2. Ensure workspace admin for creator
	adminIDs := []string{}
	if createdBy != "" {
		adminIDs = append(adminIDs, createdBy)
	}
	for _, uid := range adminIDs {
		if err := r.ensureWorkspaceAdmin(base, companyID, wid, uid); err != nil {
			// best-effort: log but don't fail
			_ = err
		}
	}

	// 3. Ensure workspace progress system binding
	_ = r.ensureWorkspaceProgress(base, companyID, wid)
	return nil
}

// getWorkspaceByID verifies a workspace exists via internal API.
func (r *Repository) getWorkspaceByID(base, workspaceID string) error {
	url := fmt.Sprintf("%s/api/internal/workspaces/%s", base, workspaceID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("get workspace status %d", resp.StatusCode)
	}
	return nil
}

// workspaceAccessEntry represents a single access row from taskProjectService.
type workspaceAccessEntry struct {
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
}

// ensureWorkspaceAdmin creates admin access if none exists for the user.
func (r *Repository) ensureWorkspaceAdmin(base, tenantID, workspaceID, userID string) error {
	// List existing access
	access, err := r.listWorkspaceAccess(base, tenantID, workspaceID)
	if err != nil {
		return err
	}
	for _, a := range access {
		if a.UserID == userID && a.Permission == "admin" {
			return nil // Already admin
		}
		if a.UserID == userID {
			return nil // Already has some access
		}
	}
	return r.createWorkspaceAccess(base, tenantID, workspaceID, userID, "admin")
}

func (r *Repository) listWorkspaceAccess(base, tenantID, workspaceID string) ([]workspaceAccessEntry, error) {
	url := fmt.Sprintf("%s/api/projects/workspace-access/workspace-permissions/tenant_id/%s/", base, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	// Filter for specific workspace
	q := req.URL.Query()
	q.Set("workspace_id", workspaceID)
	req.URL.RawQuery = q.Encode()
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list access status %d", resp.StatusCode)
	}
	var entries []workspaceAccessEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *Repository) createWorkspaceAccess(base, tenantID, workspaceID, userID, permission string) error {
	body := map[string]interface{}{
		"workspace_id": workspaceID,
		"user_id":      userID,
		"permission":   permission,
	}
	raw, _ := json.Marshal(body)
	// taskProjectService 创建/更新入口为 workspace-access/set-permission；
	// workspace-permissions 仅支持 GET 列表（POST 返回 405）。
	url := fmt.Sprintf("%s/api/projects/workspace-access/set-permission/tenant_id/%s/", base, tenantID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respRaw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create access status %d: %s", resp.StatusCode, strings.TrimSpace(string(respRaw)))
	}
	return nil
}

// ensureWorkspaceProgress binds tenant default progress to workspace if not set.
func (r *Repository) ensureWorkspaceProgress(base, tenantID, workspaceID string) error {
	// Check if already has a progress system
	url := fmt.Sprintf("%s/api/projects/workspaces/%s/progress-system/tenant_id/%s/", base, workspaceID, tenantID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		var body struct {
			ProgressSystemID string `json:"progress_system_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err == nil && body.ProgressSystemID != "" {
			return nil // Already set
		}
	}
	// Get tenant default progress
	progID, err := r.findTenantDefaultProgress(base, tenantID)
	if err != nil || progID == "" {
		return nil // No default — skip
	}
	return r.setWorkspaceProgressSystem(base, tenantID, workspaceID, progID)
}

// setWorkspaceProgressSystem calls POST .../workspaces/{wid}/progress-system/
func (r *Repository) setWorkspaceProgressSystem(base, tenantID, workspaceID, progressSystemID string) error {
	body := map[string]string{"progress_system_id": progressSystemID}
	raw, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/api/projects/workspaces/%s/progress-system/tenant_id/%s/", base, workspaceID, tenantID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	req.Header.Set("X-Auth-User-Id", "internal")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("set workspace progress status %d", resp.StatusCode)
	}
	return nil
}
