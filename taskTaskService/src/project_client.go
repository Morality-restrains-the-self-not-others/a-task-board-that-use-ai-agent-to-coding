package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

var projectHTTP = &http.Client{
	Timeout:   15 * time.Second,
	Transport: &http.Transport{Proxy: nil},
}

// projectGitProbeHTTP waits out git-oauth refresh + GitLab REST. Incident
// task_878541740905099264: validate-git-repos returned 200 in 24s after the
// 15s projectHTTP client already aborted → false auto_run skip.
var projectGitProbeHTTP = &http.Client{
	Timeout:   40 * time.Second,
	Transport: &http.Transport{Proxy: nil},
}

func projectRequest(ctx context.Context, method, path, tenantID string, body []byte) (int, json.RawMessage, error) {
	return projectRequestWithUser(ctx, method, path, tenantID, "", body)
}

func projectRequestWithUser(ctx context.Context, method, path, tenantID, userID string, body []byte) (int, json.RawMessage, error) {
	return projectRequestWithClient(projectHTTP, ctx, method, path, tenantID, userID, body)
}

func projectRequestWithClient(client *http.Client, ctx context.Context, method, path, tenantID, userID string, body []byte) (int, json.RawMessage, error) {
	if client == nil {
		client = projectHTTP
	}
	if ctx == nil {
		ctx = context.Background()
	}
	url := strings.TrimRight(cfg.ProjectServiceURL, "/") + path
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return 0, nil, err
	}
	// OPT-20260821-012: 把入站 X-Trace-Id / span 透传到 taskProjectService，
	// 使同一把 trace key 能跨 task-task-service → project-service 检索。
	tracelog.ApplyOutboundHeaders(req, ctx)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	uid := strings.TrimSpace(userID)
	if uid == "" {
		// OPT-20260820-015：taskProjectService /api/projects/ 对空用户返回 401「请先登录」。
		// 服务间调用无终端用户时以 internal 身份走 requireTenantMember 旁路。
		uid = "internal"
	}
	req.Header.Set("X-Auth-User-Id", uid)
	if strings.TrimSpace(cfg.InternalSecret) != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, json.RawMessage(raw), nil
}

func projectGetObject(ctx context.Context, path, tenantID string) (int, map[string]interface{}, error) {
	return projectGetObjectWithClient(projectHTTP, ctx, path, tenantID)
}

func projectGetObjectWithClient(client *http.Client, ctx context.Context, path, tenantID string) (int, map[string]interface{}, error) {
	status, raw, err := projectRequestWithClient(client, ctx, http.MethodGet, path, tenantID, "", nil)
	if err != nil {
		return 0, nil, err
	}
	var out map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return status, out, nil
}

func projectGetArray(ctx context.Context, path, tenantID string) (int, []map[string]interface{}, error) {
	return projectGetArrayWithUser(ctx, path, tenantID, "")
}

func projectGetArrayWithUser(ctx context.Context, path, tenantID, userID string) (int, []map[string]interface{}, error) {
	status, raw, err := projectRequestWithUser(ctx, http.MethodGet, path, tenantID, userID, nil)
	if err != nil {
		return 0, nil, err
	}
	var rows []map[string]interface{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &rows)
	}
	return status, rows, nil
}

// listAccessibleWorkspaceIDs fetches the caller's accessible workspaces in one round-trip
// (GET /workspaces/?mine=1). On project-service failure, returns err so callers can fall back.
func listAccessibleWorkspaceIDs(ctx context.Context, tenantID, userID string) ([]string, error) {
	uid := strings.TrimSpace(userID)
	// /api/projects/ 约定（6edee88 移除 legacy /api/tenant/* 路由，OPT-20260809-006）
	path := fmt.Sprintf("/api/projects/workspaces/tenant_id/%s?mine=1", tenantID)
	status, rows, err := projectGetArrayWithUser(ctx, path, tenantID, uid)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("workspaces list status %d", status)
	}
	out := make([]string, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		id := strings.TrimSpace(fmt.Sprintf("%v", row["id"]))
		if id == "" || id == "<nil>" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

// WorkspaceAtModeReader reads container_image_at_mode_enabled for a workspace.
// Replaceable in tests (no real HTTP).
type WorkspaceAtModeReader func(tenantID, workspaceID string) (bool, error)

var readWorkspaceAtModeEnabled WorkspaceAtModeReader = defaultReadWorkspaceAtModeEnabled

func defaultReadWorkspaceAtModeEnabled(tenantID, workspaceID string) (bool, error) {
	ws, err := verifyWorkspace(context.Background(), tenantID, workspaceID)
	if err != nil {
		return false, err
	}
	return workspaceAtModeFromMap(ws), nil
}

func workspaceAtModeFromMap(ws map[string]interface{}) bool {
	if ws == nil {
		return false
	}
	v, ok := ws["container_image_at_mode_enabled"]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case float64:
		return x != 0
	case int:
		return x != 0
	case int64:
		return x != 0
	default:
		return false
	}
}

func verifyWorkspace(ctx context.Context, tenantID, workspaceID string) (map[string]interface{}, error) {
	status, data, err := projectGetObject(
		ctx,
		fmt.Sprintf("/api/projects/workspaces/tenant_id/%s/%s",
			url.PathEscape(tenantID), url.PathEscape(workspaceID)),
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		log.Printf("[taskTaskService] verifyWorkspace tenant=%s workspace=%s project_status=%d", tenantID, workspaceID, status)
		return nil, fmt.Errorf("workspace not found")
	}
	return data, nil
}

func listWorkspaceAccess(ctx context.Context, tenantID, workspaceID string) ([]map[string]interface{}, error) {
	status, rows, err := projectGetArray(
		ctx,
		fmt.Sprintf("/api/projects/workspace-access/tenant_id/%s/workspace-permissions?workspace_id=%s", tenantID, workspaceID),
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("workspace access unavailable")
	}
	return rows, nil
}

func hasWorkspaceAccess(ctx context.Context, tenantID, workspaceID, userID string) bool {
	// Django / 服务端只读（task_client.get_task_by_id 等）使用 X-Auth-User-Id=internal，
	// 不在 workspace-access 成员表中；须放行，否则容器 task-detail 得到 task=null、project_repos=[]。
	if strings.TrimSpace(userID) == "internal" {
		return true
	}
	rows, err := listWorkspaceAccess(ctx, tenantID, workspaceID)
	if err == nil && len(rows) > 0 {
		for _, row := range rows {
			// workspace-permissions 富化响应使用 user/group 键（旧 raw 行是 user_id/group_id）
			uid := strings.TrimSpace(fmt.Sprintf("%v", row["user"]))
			if uid == "" || uid == "<nil>" {
				uid = strings.TrimSpace(fmt.Sprintf("%v", row["user_id"]))
			}
			gid := strings.TrimSpace(fmt.Sprintf("%v", row["group"]))
			if gid == "" || gid == "<nil>" {
				gid = strings.TrimSpace(fmt.Sprintf("%v", row["group_id"]))
			}
			// Direct user match (fast path)
			if uid != "" && uid != "<nil>" && uid == userID {
				return true
			}
			// OPT-20260726-021: Check group_id membership via taskTenantService internal API
			// Matches Django has_workspace_access group-id logic. Fallback to user_id-only
			// if tenant service is unreachable (permissive rather than strict deny).
			if gid != "" && gid != "<nil>" && gid != "0" {
				if checkUserInGroup(userID, gid) {
					return true
				}
			}
		}
		return false
	}
	if err != nil {
		// Fail-closed: if workspace-access API is unavailable, deny access rather than
		// silently granting it (aligns with search ACL fail-closed policy). OPT-20260720-024.
		log.Printf("[taskTaskService] hasWorkspaceAccess: list error, fail-closed: %v", err)
		return false
	}
	ws, werr := verifyWorkspace(ctx, tenantID, workspaceID)
	return werr == nil && ws != nil
}

// checkUserInGroup queries taskTenantService internal API to verify group membership.
// Returns false if the API is unreachable (permissive fallback — avoid denying access
// due to transient infra issues; the caller still requires workspace_access entry to exist).
func checkUserInGroup(userID, groupID string) bool {
	if cfg.TaskTenantServiceURL == "" {
		return false
	}
	url := fmt.Sprintf("%s/api/internal/tenant/groups/user-in-group?user_id=%s&group_id=%s",
		strings.TrimRight(cfg.TaskTenantServiceURL, "/"), userID, groupID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-TaskBill-Internal-Secret", cfg.InternalSecret)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[taskTaskService] checkUserInGroup: tenant service unreachable for user=%s group=%s: %v",
			userID, groupID, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return false
	}
	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		InGroup bool `json:"in_group"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return false
	}
	return result.InGroup
}

func getProjectRepoURLs(ctx context.Context, tenantID, projectID string) ([]string, error) {
	entries, err := getProjectRepoEntries(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.URL)
	}
	return out, nil
}

type projectRepoEntry struct {
	URL        string
	CloneAlias string
}

type projectRepoMeta struct {
	Entries              []projectRepoEntry
	AutoCloneNestedRepos bool
}

func autoCloneNestedReposFromMap(data map[string]interface{}) bool {
	if data == nil {
		return true
	}
	v, ok := data["auto_clone_nested_repos"]
	if !ok || v == nil {
		return true
	}
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return !(s == "false" || s == "0")
	default:
		return true
	}
}

func loadProjectRepoMeta(ctx context.Context, tenantID, projectID string) (projectRepoMeta, error) {
	meta := projectRepoMeta{AutoCloneNestedRepos: true}
	status, data, err := projectGetObject(
		ctx,
		fmt.Sprintf("/api/projects/tenant_id/%s/%s", tenantID, projectID),
		tenantID,
	)
	if err != nil || status != 200 {
		return meta, fmt.Errorf("project not found")
	}
	meta.AutoCloneNestedRepos = autoCloneNestedReposFromMap(data)
	if raw, ok := data["git_repo_entries"].([]interface{}); ok && len(raw) > 0 {
		out := make([]projectRepoEntry, 0, len(raw))
		for _, item := range raw {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			url := strings.TrimSpace(strFromAny(m["url"]))
			if url == "" {
				url = strings.TrimSpace(strFromAny(m["repo_url"]))
			}
			if url == "" {
				continue
			}
			out = append(out, projectRepoEntry{
				URL:        url,
				CloneAlias: strings.TrimSpace(strFromAny(m["clone_alias"])),
			})
		}
		if len(out) > 0 {
			meta.Entries = out
			return meta, nil
		}
	}
	repos, _ := data["git_repos"].([]interface{})
	out := []projectRepoEntry{}
	for _, r := range repos {
		if s, ok := r.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, projectRepoEntry{URL: strings.TrimSpace(s)})
		}
	}
	meta.Entries = out
	return meta, nil
}

func getProjectAutoCloneNestedRepos(ctx context.Context, tenantID, projectID string) bool {
	meta, err := loadProjectRepoMeta(ctx, tenantID, projectID)
	if err != nil {
		return true
	}
	return meta.AutoCloneNestedRepos
}

func getProjectRepoEntries(ctx context.Context, tenantID, projectID string) ([]projectRepoEntry, error) {
	meta, err := loadProjectRepoMeta(ctx, tenantID, projectID)
	if err != nil {
		return nil, err
	}
	return meta.Entries, nil
}

// resolveDeliverableObj resolves a deliverable_obj_id to {id, name} via project-service.
// Returns nil on any error — callers should fall back to deliverable_obj_id string.
func resolveDeliverableObj(tenantID, objID string) map[string]interface{} {
	if objID == "" || tenantID == "" {
		return nil
	}
	status, data, err := projectGetObject(
		context.Background(),
		fmt.Sprintf("/api/internal/deliverable-systems/lookup?id=%s", objID),
		tenantID,
	)
	if err != nil || status != 200 {
		return nil
	}
	out := map[string]interface{}{
		"id": objID,
	}
	if name, ok := data["name"].(string); ok && name != "" {
		out["name"] = name
	}
	if isSys, ok := data["is_system"].(bool); ok {
		out["is_system"] = isSys
	}
	return out
}
