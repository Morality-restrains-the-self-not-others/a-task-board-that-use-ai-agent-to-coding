package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	nonBranchTitleChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	branchTitleSpaces   = regexp.MustCompile(`\s+`)
)

// attachLayerGitPushPRMetadata adds pr_base_branch / pr_title / pr_body when resolvable.
// Without pr_base_branch, container oauth-access-push skips GitHub PR creation after a successful push.
func attachLayerGitPushPRMetadata(
	pushBody map[string]any,
	tenantID, workspaceID, taskID string,
) {
	if pushBody == nil {
		return
	}
	pushedTB := ""
	if v, ok := pushBody["target_branch"].(string); ok {
		pushedTB = strings.TrimSpace(v)
	}
	meta := resolveLayerPushGithubPRMetadata(tenantID, workspaceID, taskID, pushedTB)
	if meta == nil {
		return
	}
	for k, v := range meta {
		if strings.TrimSpace(v) != "" {
			pushBody[k] = v
		}
	}
}

// resolveLayerPushGithubPRMetadata mirrors Django resolve_layer_push_github_pr_metadata.
func resolveLayerPushGithubPRMetadata(
	tenantID, workspaceID, taskID, pushedTargetBranch string,
) map[string]string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil
	}
	snap, err := fetchTaskPRSnapHTTP(tenantID, workspaceID, taskID)
	if err != nil || snap == nil {
		return nil
	}
	mergeRaw := strings.TrimSpace(snap.MergeTargetBranch)
	if mergeRaw == "" {
		mergeRaw = "develop"
	}
	head := resolveLayerPushHeadBranch(snap, pushedTargetBranch)
	base := substBranchPlaceholders(mergeRaw, snap.ID, snap.Title)
	if head == "" || base == "" {
		return nil
	}
	title := "Pull request"
	if t := strings.TrimSpace(snap.Title); t != "" {
		if len(t) > 120 {
			t = t[:120]
		}
		title = "PR: " + t
	}
	return map[string]string{
		"pr_base_branch": base,
		"pr_title":       title,
		"pr_body":        "任务 #" + snap.ID + "：由层级推送自动创建。",
	}
}

type taskPRSnap struct {
	ID                   string
	Title                string
	WorkBranchName       string
	MergeTargetBranch    string
	TargetBranchName     string
}

func resolveLayerPushHeadBranch(snap *taskPRSnap, pushedTargetBranch string) string {
	if snap == nil {
		return ""
	}
	pt := strings.TrimSpace(pushedTargetBranch)
	if pt != "" {
		return substBranchPlaceholders(pt, snap.ID, snap.Title)
	}
	if direct := strings.TrimSpace(snap.TargetBranchName); direct != "" {
		return substBranchPlaceholders(direct, snap.ID, snap.Title)
	}
	if work := strings.TrimSpace(snap.WorkBranchName); work != "" {
		return substBranchPlaceholders(work, snap.ID, snap.Title)
	}
	return ""
}

func substBranchPlaceholders(name, taskID, taskTitle string) string {
	result := strings.ReplaceAll(strings.TrimSpace(name), "${taskId}", taskID)
	titleSeg := strings.TrimSpace(taskTitle)
	titleSeg = branchTitleSpaces.ReplaceAllString(titleSeg, "_")
	titleSeg = nonBranchTitleChars.ReplaceAllString(titleSeg, "_")
	if titleSeg == "" {
		titleSeg = "task"
	}
	result = strings.ReplaceAll(result, "${taskTitle}", titleSeg)
	result = strings.ReplaceAll(result, "__taskTitle_", titleSeg)
	return result
}

func fetchTaskPRSnapHTTP(tenantID, workspaceID, taskID string) (*taskPRSnap, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if base == "" {
		return nil, nil
	}
	u := base + "/api/tasks/" + url.PathEscape(taskID) + "/"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tenantID) != "" {
		req.Header.Set("X-Auth-Tenant-Id", tenantID)
	}
	req.Header.Set("X-Auth-User-Id", "internal")
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	ws := strField(body, "workspace_id")
	if ws != "" && strings.TrimSpace(workspaceID) != "" && ws != workspaceID {
		return nil, nil
	}
	id := strField(body, "id")
	if id == "" {
		id = taskID
	}
	snap := &taskPRSnap{
		ID:    id,
		Title: strField(body, "title"),
	}
	if bs, ok := body["branch_strategy"].(map[string]any); ok && bs != nil {
		snap.WorkBranchName = strField(bs, "work_branch_name")
		snap.MergeTargetBranch = strField(bs, "merge_target_branch_name")
		snap.TargetBranchName = strField(bs, "target_branch_name")
	}
	return snap, nil
}
