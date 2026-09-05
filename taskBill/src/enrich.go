package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var enrichHTTP = &http.Client{Timeout: 10 * time.Second}

// enrichBillingRows adds project_name, user_name, workspace_name, task_name
// to billing rows via batch HTTP calls to upstream services.
// Replaces the Django /api/internal/taskbill/enrich-transactions/ bridge.
func enrichBillingRows(ctx context.Context, tenantID int64, rows []map[string]interface{}) []map[string]interface{} {
	if len(rows) == 0 {
		return rows
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Step 1: Collect unique task_ids and direct project_ids
	seenTask := map[string]bool{}
	var taskIDs []string
	directProjectIDs := map[string]bool{}
	for _, row := range rows {
		if row == nil {
			continue
		}
		tid, _ := row["task_id"].(string)
		if tid != "" && !seenTask[tid] {
			seenTask[tid] = true
			taskIDs = append(taskIDs, tid)
		} else if tid == "" {
			// Row has direct project_id (no task_id)
			if pid, _ := row["project_id"].(string); pid != "" {
				directProjectIDs[pid] = true
			}
		}
	}

	// Step 2: Batch-resolve task→project_ids via POST /api/internal/tasks/batch-get/
	taskProjectMap := resolveTaskProjectMappingsBatch(ctx, taskIDs)

	// Step 3: Collect all project_ids from task mappings
	allProjectIDs := make(map[string]bool, len(directProjectIDs))
	for pid := range directProjectIDs {
		allProjectIDs[pid] = true
	}
	taskToProjectName := map[string]string{} // task_id → joined project names
	for _, pids := range taskProjectMap {
		for _, pid := range pids {
			allProjectIDs[pid] = true
		}
	}

	// Step 4: Batch-resolve project names via POST /api/internal/projects/batch-get/
	pidList := make([]string, 0, len(allProjectIDs))
	for pid := range allProjectIDs {
		pidList = append(pidList, pid)
	}
	projectNameMap := resolveProjectNamesBatch(ctx, pidList)

	// Step 5: Build task→project_name (joined with 、)
	for tid, pids := range taskProjectMap {
		var names []string
		for _, pid := range pids {
			if name, ok := projectNameMap[pid]; ok {
				names = append(names, name)
			} else {
				names = append(names, pid) // fallback to raw ID
			}
		}
		if len(names) > 0 {
			taskToProjectName[tid] = strings.Join(names, "、")
		}
	}

	// Step 6: Apply project_name to rows
	for i, row := range rows {
		if row == nil {
			continue
		}
		tid, _ := row["task_id"].(string)
		if tid != "" {
			if name, ok := taskToProjectName[tid]; ok {
				rows[i]["project_name"] = name
			}
		} else {
			// Direct project_id fallback
			if pid, _ := row["project_id"].(string); pid != "" {
				if name, ok := projectNameMap[pid]; ok {
					rows[i]["project_name"] = name
				} else {
					rows[i]["project_name"] = pid
				}
			}
		}
	}

	// Enrich user_name from company member IDs via taskTenantService
	rows = enrichUserNames(ctx, tenantID, rows)

	// Enrich workspace_name via taskProjectService internal API
	rows = enrichWorkspaceNames(ctx, rows)

	// Enrich task_name via taskTaskService
	rows = enrichTaskNames(ctx, rows)

	return rows
}

// enrichTaskNames resolves task_id → task title via taskTaskService.
func enrichTaskNames(ctx context.Context, rows []map[string]interface{}) []map[string]interface{} {
	if ctx == nil {
		ctx = context.Background()
	}
	// Collect unique task_ids (only for rows that don't already have task_name)
	seen := map[string]bool{}
	var taskIDs []string
	for _, row := range rows {
		if row == nil {
			continue
		}
		tid, _ := row["task_id"].(string)
		if tid == "" || seen[tid] {
			continue
		}
		seen[tid] = true
		taskIDs = append(taskIDs, tid)
	}
	if len(taskIDs) == 0 {
		return rows
	}

	// Batch resolve task titles via POST /api/internal/tasks/batch-get/
	titleCache := resolveTaskTitlesBatch(ctx, taskIDs)

	// Apply titles to rows
	for i, row := range rows {
		if row == nil {
			continue
		}
		tid, _ := row["task_id"].(string)
		if title, ok := titleCache[tid]; ok {
			rows[i]["task_name"] = title
		}
	}
	return rows
}

// resolveTaskTitlesBatch calls taskTaskService POST /api/internal/tasks/batch-get/
// to resolve multiple task IDs to titles in a single HTTP round-trip.
func resolveTaskTitlesBatch(ctx context.Context, taskIDs []string) map[string]string {
	out := make(map[string]string, len(taskIDs))
	if len(taskIDs) == 0 {
		return out
	}
	base := strings.TrimRight(cfg.TaskTaskServiceBase, "/")
	if base == "" {
		log.Printf("[taskBill] enrich: taskTaskService base URL not configured, skipping task batch resolve")
		return out
	}
	url := fmt.Sprintf("%s/api/internal/tasks/batch-get/", base)
	payload, _ := json.Marshal(map[string]interface{}{"task_ids": taskIDs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("[taskBill] enrich: task batch-get request error: %v", err)
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := enrichHTTP.Do(req)
	if err != nil {
		log.Printf("[taskBill] enrich: batch-get tasks failed: %v", err)
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskBill] enrich: batch-get tasks returned %d", resp.StatusCode)
		return out
	}
	raw, _ := io.ReadAll(resp.Body)
	var data struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Printf("[taskBill] enrich: task batch-get decode error: %v", err)
		return out
	}
	for _, t := range data.Tasks {
		id, _ := t["id"].(string)
		title, _ := t["title"].(string)
		if id != "" && title != "" {
			out[id] = title
		}
	}
	return out
}

// enrichUserNames resolves user_id (company_member_id) → member_name via taskTenantService batch API.
func enrichUserNames(ctx context.Context, tenantID int64, rows []map[string]interface{}) []map[string]interface{} {
	if ctx == nil {
		ctx = context.Background()
	}
	// Collect unique user_ids
	seen := map[string]bool{}
	var memberIDs []string
	for _, row := range rows {
		if row == nil {
			continue
		}
		uid, _ := row["user_id"].(string)
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		memberIDs = append(memberIDs, uid)
	}
	if len(memberIDs) == 0 {
		return rows
	}

	// Batch resolve member names via POST /api/internal/tenant/members/batch-get/
	nameCache := resolveMemberNamesBatch(ctx, formatID(tenantID), memberIDs)

	// Apply names to rows
	for i, row := range rows {
		if row == nil {
			continue
		}
		uid, _ := row["user_id"].(string)
		if name, ok := nameCache[uid]; ok {
			rows[i]["user_name"] = name
		}
	}
	return rows
}

// resolveMemberNamesBatch calls taskTenantService POST /api/internal/tenant/members/batch-get/
// to resolve multiple user account IDs to member names in a single HTTP round-trip.
// Sends user_ids (not member_ids) because billing_transaction.user_id stores the user account ID,
// which matches tenant_company_member.user_id (not tenant_company_member.id).
func resolveMemberNamesBatch(ctx context.Context, tenantID string, userIDs []string) map[string]string {
	out := make(map[string]string, len(userIDs))
	if len(userIDs) == 0 {
		return out
	}
	base := strings.TrimRight(cfg.TaskTenantServiceURL, "/")
	if base == "" {
		log.Printf("[taskBill] enrich: TaskTenantServiceURL not configured, skipping member batch resolve")
		return out
	}
	url := fmt.Sprintf("%s/api/internal/tenant/members/batch-get/", base)
	payload, _ := json.Marshal(map[string]interface{}{"user_ids": userIDs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("[taskBill] enrich: batch-get request error: %v", err)
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := enrichHTTP.Do(req)
	if err != nil {
		log.Printf("[taskBill] enrich: batch-get members failed: %v", err)
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskBill] enrich: batch-get members returned %d", resp.StatusCode)
		return out
	}
	raw, _ := io.ReadAll(resp.Body)
	var data struct {
		Members []map[string]interface{} `json:"members"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Printf("[taskBill] enrich: batch-get decode error: %v", err)
		return out
	}
	for _, m := range data.Members {
		name, _ := m["member_name"].(string)
		// Use user_id as cache key — billing_transaction.user_id stores the user account ID,
		// which matches tenant_company_member.user_id (not member PK id).
		uid, _ := m["user_id"].(string)
		if uid == "" {
			uid, _ = m["id"].(string) // fallback for member_ids queries
		}
		if uid != "" && name != "" {
			out[uid] = name
		}
	}
	return out
}

// enrichWorkspaceNames resolves workspace_id → workspace name via taskProjectService internal API.
func enrichWorkspaceNames(ctx context.Context, rows []map[string]interface{}) []map[string]interface{} {
	if ctx == nil {
		ctx = context.Background()
	}
	// Collect unique workspace_ids
	seen := map[string]bool{}
	var wsIDs []string
	for _, row := range rows {
		if row == nil {
			continue
		}
		wid, _ := row["workspace_id"].(string)
		if wid == "" || seen[wid] {
			continue
		}
		seen[wid] = true
		wsIDs = append(wsIDs, wid)
	}
	if len(wsIDs) == 0 {
		return rows
	}

	// Batch resolve workspace names via POST /api/internal/workspaces/batch-get/
	nameCache := resolveWorkspaceNamesBatch(ctx, wsIDs)

	// Apply names to rows
	for i, row := range rows {
		if row == nil {
			continue
		}
		wid, _ := row["workspace_id"].(string)
		if name, ok := nameCache[wid]; ok {
			rows[i]["workspace_name"] = name
		}
	}
	return rows
}

// resolveWorkspaceNamesBatch calls taskProjectService POST /api/internal/workspaces/batch-get/
// to resolve multiple workspace IDs to names in a single HTTP round-trip.
func resolveWorkspaceNamesBatch(ctx context.Context, wsIDs []string) map[string]string {
	out := make(map[string]string, len(wsIDs))
	if len(wsIDs) == 0 {
		return out
	}
	base := strings.TrimRight(cfg.TaskProjectServiceBase, "/")
	if base == "" {
		log.Printf("[taskBill] enrich: taskProjectService base URL not configured, skipping workspace batch resolve")
		return out
	}
	url := fmt.Sprintf("%s/api/internal/workspaces/batch-get/", base)
	payload, _ := json.Marshal(map[string]interface{}{"workspace_ids": wsIDs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("[taskBill] enrich: workspace batch-get request error: %v", err)
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := enrichHTTP.Do(req)
	if err != nil {
		log.Printf("[taskBill] enrich: batch-get workspaces failed: %v", err)
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskBill] enrich: batch-get workspaces returned %d", resp.StatusCode)
		return out
	}
	raw, _ := io.ReadAll(resp.Body)
	var data struct {
		Workspaces []map[string]interface{} `json:"workspaces"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Printf("[taskBill] enrich: workspace batch-get decode error: %v", err)
		return out
	}
	for _, w := range data.Workspaces {
		id, _ := w["id"].(string)
		name, _ := w["name"].(string)
		if id != "" && name != "" {
			out[id] = name
		}
	}
	return out
}

// resolveTaskProjectMappingsBatch calls taskTaskService POST /api/internal/tasks/batch-get/
// to resolve task_id → project_ids mapping in a single HTTP round-trip.
func resolveTaskProjectMappingsBatch(ctx context.Context, taskIDs []string) map[string][]string {
	out := make(map[string][]string, len(taskIDs))
	if len(taskIDs) == 0 {
		return out
	}
	base := strings.TrimRight(cfg.TaskTaskServiceBase, "/")
	if base == "" {
		log.Printf("[taskBill] enrich: taskTaskService base URL not configured, skipping task-project mapping")
		return out
	}
	url := fmt.Sprintf("%s/api/internal/tasks/batch-get/", base)
	payload, _ := json.Marshal(map[string]interface{}{"task_ids": taskIDs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("[taskBill] enrich: task-project mapping request error: %v", err)
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := enrichHTTP.Do(req)
	if err != nil {
		log.Printf("[taskBill] enrich: task-project mapping failed: %v", err)
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskBill] enrich: task-project mapping returned %d", resp.StatusCode)
		return out
	}
	raw, _ := io.ReadAll(resp.Body)
	var data struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Printf("[taskBill] enrich: task-project mapping decode error: %v", err)
		return out
	}
	for _, t := range data.Tasks {
		tid, _ := t["id"].(string)
		if tid == "" {
			continue
		}
		pidsRaw, _ := t["project_ids"].([]interface{})
		pids := make([]string, 0, len(pidsRaw))
		for _, p := range pidsRaw {
			if ps, ok := p.(string); ok && ps != "" {
				pids = append(pids, ps)
			}
		}
		out[tid] = pids
	}
	return out
}

// resolveProjectNamesBatch calls taskProjectService POST /api/internal/projects/batch-get/
// to resolve project IDs to names in a single HTTP round-trip.
func resolveProjectNamesBatch(ctx context.Context, projectIDs []string) map[string]string {
	out := make(map[string]string, len(projectIDs))
	if len(projectIDs) == 0 {
		return out
	}
	base := strings.TrimRight(cfg.TaskProjectServiceBase, "/")
	if base == "" {
		log.Printf("[taskBill] enrich: taskProjectService base URL not configured, skipping project name resolve")
		return out
	}
	url := fmt.Sprintf("%s/api/internal/projects/batch-get/", base)
	payload, _ := json.Marshal(map[string]interface{}{"project_ids": projectIDs})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("[taskBill] enrich: project batch-get request error: %v", err)
		return out
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.InternalSecret != "" {
		req.Header.Set("X-Internal-Secret", cfg.InternalSecret)
	}
	resp, err := enrichHTTP.Do(req)
	if err != nil {
		log.Printf("[taskBill] enrich: batch-get projects failed: %v", err)
		return out
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[taskBill] enrich: batch-get projects returned %d", resp.StatusCode)
		return out
	}
	raw, _ := io.ReadAll(resp.Body)
	var data struct {
		Projects []map[string]interface{} `json:"projects"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Printf("[taskBill] enrich: project batch-get decode error: %v", err)
		return out
	}
	for _, p := range data.Projects {
		id, _ := p["id"].(string)
		name, _ := p["name"].(string)
		if id != "" && name != "" {
			out[id] = name
		}
	}
	return out
}
