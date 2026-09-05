package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"autorunstartvm"
	"tracelog"
)

const autoRunGitAccessSkippedCode = "AUTO_RUN_GIT_ACCESS_SKIPPED"

// AUTO_RUN_RUNTIME_ENV_REQUIRED is returned when the installed image has no
// runtime environment matching the project's server_run_template platform/region.
const autoRunRuntimeEnvRequiredCode = "AUTO_RUN_RUNTIME_ENV_REQUIRED"

// autoRunGateError is returned when auto_run=true fails prerequisites (HTTP 400).
type autoRunGateError struct {
	Code    string
	Message string
}

func (e *autoRunGateError) Error() string {
	return e.Message
}

func writeAutoRunGateError(w http.ResponseWriter, err error) bool {
	e, ok := err.(*autoRunGateError)
	if !ok || e == nil {
		return false
	}
	writeErrorMap(w, nil, http.StatusBadRequest, map[string]interface{}{
		"error":  e.Message,
		"detail": e.Message,
		"code":   e.Code,
	})
	return true
}

type autoRunTriggerParams struct {
	TenantID       string
	WorkspaceID    string
	TaskID         string
	UserID         string
	ImageID        string
	RunTemplate    map[string]interface{}
	ClientPublicIP string // browser public IP for auto SG whitelist (start-vm-auto)
	// StartSkipReason: non-empty means soft-skip start-vm (e.g. nested git probe failed),
	// but still create the 【自动运行】comment with this reason embedded.
	StartSkipReason string
	RepoIdentities  []RepoIdentitySelection
	AgentModels     []map[string]interface{}
	GrantTicket     string
}

// scheduleTaskAutoRunFn / triggerTaskAutoRunFn are replaceable in tests.
var scheduleTaskAutoRunFn = scheduleTaskAutoRun
var triggerTaskAutoRunFn = triggerTaskAutoRun

func scheduleTaskAutoRun(p autoRunTriggerParams) {
	go func() {
		if err := triggerTaskAutoRunFn(p); err != nil {
			log.Printf("[taskTaskService] auto_run start-vm failed task_id=%s: %v", p.TaskID, err)
		}
	}()
}

func triggerTaskAutoRun(p autoRunTriggerParams) error {
	commentID, err := resolveAutoRunCommentID(p)
	if err != nil {
		log.Printf("[taskTaskService] auto_run at-comment failed task_id=%s: %v", p.TaskID, err)
		return err
	}
	if skip := strings.TrimSpace(p.StartSkipReason); skip != "" {
		log.Printf(
			"[taskTaskService] auto_run comment-only (start skipped) task_id=%s comment_id=%s reason=%s",
			p.TaskID, commentID, skip,
		)
		tracelog.LogForwardStage(context.Background(), "task_auto_run_start_vm_skipped", map[string]any{
			"task_id":    p.TaskID,
			"comment_id": commentID,
			"reason":     skip,
		})
		return nil
	}
	req, err := autorunstartvm.BuildAutoRunStartVmRequest(p.TaskID, p.ImageID, p.RunTemplate)
	if err != nil {
		return err
	}
	if req.Body == nil {
		req.Body = map[string]interface{}{}
	}
	attachStartVmCommentID(req.Body, commentID, p.TaskID)
	if ip := strings.TrimSpace(p.ClientPublicIP); ip != "" {
		req.Body["client_public_ip"] = ip
	} else if boolField(req.Body, "auto_create_security_group") {
		log.Printf("[taskTaskService] auto_run missing client_public_ip for auto SG task_id=%s", p.TaskID)
	}
	ctx := context.Background()
	traceID := tracelog.NormalizeTraceID(p.TaskID)
	if traceID == "" {
		traceID = tracelog.NewTraceID()
	}
	ctx = tracelog.ContextWithCorrelation(ctx, tracelog.Correlation{
		TraceID: traceID,
		SpanID:  tracelog.NewSpanID(),
	})
	tracelog.LogForwardStage(ctx, "task_auto_run_start_vm_begin", map[string]any{
		"task_id":          p.TaskID,
		"comment_id":       commentID,
		"api_path":         req.APIPath,
		"client_public_ip": strings.TrimSpace(p.ClientPublicIP),
	})
	if err := startVMFn(ctx, p.TenantID, p.WorkspaceID, req.APIPath, p.UserID, req.Body); err != nil {
		tracelog.LogForwardStage(ctx, "task_auto_run_start_vm_error", map[string]any{
			"task_id": p.TaskID,
			"error":   err.Error(),
		})
		return err
	}
	tracelog.LogForwardStage(ctx, "task_auto_run_start_vm_ok", map[string]any{
		"task_id":  p.TaskID,
		"api_path": req.APIPath,
	})
	return nil
}

// validateAutoRunPrerequisites enforces CloudServiceURL + image + linked project run template.
// Returns the resolved run template for subsequent start-vm.
func validateAutoRunPrerequisites(tenantID, imageID string, linkedProjectIDs []string) (map[string]interface{}, error) {
	if strings.TrimSpace(cfg.CloudServiceURL) == "" {
		return nil, &autoRunGateError{
			Code:    "AUTO_RUN_CLOUD_UNAVAILABLE",
			Message: "自动运行不可用：云服务未配置",
		}
	}
	if strings.TrimSpace(imageID) == "" {
		return nil, &autoRunGateError{
			Code:    "AUTO_RUN_IMAGE_REQUIRED",
			Message: "自动运行需要选择已安装镜像",
		}
	}
	if len(linkedProjectIDs) == 0 {
		return nil, &autoRunGateError{
			Code:    "AUTO_RUN_PROJECT_REQUIRED",
			Message: "自动运行需要关联项目",
		}
	}

	projects := make([]map[string]interface{}, 0, len(linkedProjectIDs))
	linked := make([]map[string]interface{}, 0, len(linkedProjectIDs))
	for _, pid := range linkedProjectIDs {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		status, data, err := projectGetObject(
			context.Background(),
			fmt.Sprintf("/api/projects/tenant_id/%s/%s", tenantID, pid),
			tenantID,
		)
		if err != nil || status != http.StatusOK || data == nil {
			log.Printf("[taskTaskService] auto_run project fetch failed project_id=%s status=%d err=%v", pid, status, err)
			continue
		}
		projects = append(projects, data)
		linked = append(linked, map[string]interface{}{"project_id": pid})
	}
	if len(projects) == 0 {
		return nil, &autoRunGateError{
			Code:    "AUTO_RUN_PROJECT_REQUIRED",
			Message: "自动运行需要关联项目",
		}
	}

	tpl := autorunstartvm.ResolveProjectServerRunTemplateFromProjects(projects, linked)
	if !autorunstartvm.RunTemplateIsConfigured(tpl) {
		return nil, &autoRunGateError{
			Code:    "AUTO_RUN_RUN_TEMPLATE_REQUIRED",
			Message: "自动运行需要项目配置完整运行模版",
		}
	}
	// Project-level allow gate: server_run_template.default_auto_run must be true.
	// When false/absent, create/update with auto_run=true must not start machine nodes.
	if !boolField(tpl, "default_auto_run") {
		return nil, &autoRunGateError{
			Code:    "AUTO_RUN_PROJECT_NOT_ALLOWED",
			Message: "自动运行不可用：项目未允许自动运行",
		}
	}
	// Ensure template can actually build a start-vm request (region + platform_id).
	if _, err := autorunstartvm.BuildAutoRunStartVmRequest("validate", imageID, tpl); err != nil {
		return nil, &autoRunGateError{
			Code:    "AUTO_RUN_RUN_TEMPLATE_REQUIRED",
			Message: "自动运行需要项目配置完整运行模版",
		}
	}

	// Verify the installed image has a resolvable external_image_id before accepting auto_run.
	// Without it, start-vm-auto will fail later when trying to resolve runtime environments.
	img, err := lookupInstalledImageFn(tenantID, imageID)
	if err != nil || img == nil {
		return nil, &autoRunGateError{
			Code:    autoRunRuntimeEnvRequiredCode,
			Message: "自动运行不可用：无法获取已安装镜像信息，镜像可能已被删除",
		}
	}
	if strings.TrimSpace(img.ExternalImageID) == "" {
		return nil, &autoRunGateError{
			Code:    autoRunRuntimeEnvRequiredCode,
			Message: "自动运行不可用：已安装镜像缺少 external_image_id，无法匹配运行环境",
		}
	}

	return tpl, nil
}

// probeGitAccessForAutoRun returns a non-empty reason when linked project Git / nested Git
// repos cannot be fetched for the acting user. Empty reason means auto-start may proceed.
// Fail-closed: transport errors and non-OK HTTP also skip start.
// When the project's auto_clone_nested_repos=false, only the parent repo needs to be
// reachable — nested-git-repos (.gitmodules) probing is skipped (OPT-20260818-047).
func probeGitAccessForAutoRun(ctx context.Context, userID, tenantID string, linkedProjectIDs []string) string {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return "无法获取 Git 仓库：缺少用户身份，已跳过自动启动服务器"
	}
	seen := map[string]struct{}{}
	for _, pid := range linkedProjectIDs {
		pid = strings.TrimSpace(pid)
		if pid == "" {
			continue
		}
		meta, err := loadProjectRepoMeta(ctx, tenantID, pid)
		if err != nil {
			log.Printf("[taskTaskService] auto_run git probe project fetch failed project_id=%s: %v", pid, err)
			return "无法获取 Git 仓库列表：关联项目读取失败，已跳过自动启动服务器"
		}
		for _, e := range meta.Entries {
			repoURL := strings.TrimSpace(e.URL)
			if repoURL == "" {
				continue
			}
			if _, dup := seen[repoURL]; dup {
				continue
			}
			seen[repoURL] = struct{}{}
			var reason string
			if meta.AutoCloneNestedRepos {
				reason = probeNestedGitReposAccess(uid, tenantID, repoURL)
			} else {
				reason = probeParentGitReposAccess(uid, tenantID, repoURL)
			}
			if reason != "" {
				return reason
			}
		}
	}
	return ""
}

// probeParentGitReposAccess validates parent-repo OAuth via taskProjectService
// validate-git-repos with probe_access=false. Used when auto_clone_nested_repos=false
// — reading .gitmodules is not required, so nested-git-repos errors must not
// soft-skip start-vm. Remote GitLab REST (probe_access=true) can exceed the
// previous 15s client (task_878541740905099264: 24s 200 after client abort).
func probeParentGitReposAccess(userID, tenantID, repoURL string) string {
	uid := strings.TrimSpace(userID)
	body, _ := json.Marshal(map[string]interface{}{
		"urls":         []string{repoURL},
		"probe_access": false,
	})
	path := fmt.Sprintf("/api/projects/tenant_id/%s/validate-git-repos", tenantID)
	status, raw, err := projectRequestWithClient(projectGitProbeHTTP, context.Background(), http.MethodPost, path, tenantID, uid, body)
	if err != nil {
		log.Printf("[taskTaskService] auto_run parent-git probe transport error repo=%s: %v", repoURL, err)
		return parentGitProbeFailureReason(err, 0)
	}
	if status != http.StatusOK {
		log.Printf("[taskTaskService] auto_run parent-git probe bad status repo=%s status=%d", repoURL, status)
		return parentGitProbeFailureReason(nil, status)
	}
	var out struct {
		Results []struct {
			URL          string `json:"url"`
			IsAccessible bool   `json:"is_accessible"`
			Message      string `json:"message"`
		} `json:"results"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	for _, res := range out.Results {
		if !res.IsAccessible {
			if msg := strings.TrimSpace(res.Message); msg != "" && msg != "<nil>" {
				return msg
			}
			return "无法访问父 Git 仓库：无有效授权，已跳过自动启动服务器"
		}
	}
	return ""
}

func parentGitProbeFailureReason(err error, status int) string {
	if err != nil {
		if isHTTPClientTimeout(err) {
			return "无法验证父 Git 仓库访问：探测超时，已跳过自动启动服务器"
		}
		return "无法验证父 Git 仓库访问：探测失败，已跳过自动启动服务器"
	}
	if status != http.StatusOK && status != 0 {
		return fmt.Sprintf("无法验证父 Git 仓库访问：探测失败（HTTP %d），已跳过自动启动服务器", status)
	}
	return "无法验证父 Git 仓库访问：探测失败，已跳过自动启动服务器"
}

func isHTTPClientTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Client.Timeout") || strings.Contains(msg, "context deadline exceeded")
}

func probeNestedGitReposAccess(userID, tenantID, repoURL string) string {
	tid := strings.TrimSpace(tenantID)
	q := url.Values{}
	q.Set("repo_url", repoURL)
	q.Set("user_id", userID)
	if tid != "" {
		q.Set("company_id", tid)
		q.Set("tenant_id", tid)
	} else {
		log.Printf("[taskTaskService] event=auto_run_nested_git_probe_missing_tenant repo=%s user_id=%s", repoURL, userID)
	}
	path := "/api/internal/nested-git-repos/?" + q.Encode()
	status, data, err := projectGetObjectWithClient(projectGitProbeHTTP, context.Background(), path, tid)
	if err != nil {
		log.Printf("[taskTaskService] auto_run nested-git probe transport error repo=%s tenant_id=%s: %v", repoURL, tid, err)
		return "无法获取子 Git 仓库列表：探测失败，已跳过自动启动服务器"
	}
	if status != http.StatusOK || data == nil {
		log.Printf("[taskTaskService] auto_run nested-git probe bad status repo=%s tenant_id=%s status=%d", repoURL, tid, status)
		return "无法获取子 Git 仓库列表：探测失败，已跳过自动启动服务器"
	}
	errMsg := strings.TrimSpace(fmt.Sprintf("%v", data["error"]))
	if errMsg == "" || errMsg == "<nil>" {
		return ""
	}
	return errMsg
}

func applyAutoRunStartSkip(payload map[string]interface{}, skipReason string) {
	if payload == nil || strings.TrimSpace(skipReason) == "" {
		return
	}
	payload["auto_run_start_skipped"] = true
	payload["auto_run_start_skip_reason"] = skipReason
	payload["code"] = autoRunGitAccessSkippedCode
}

// shouldValidateAutoRunOnTaskUpdate is false when the update enters 已完成/已取消.
// Terminal progress must not be blocked by auto-run image/template gates.
func shouldValidateAutoRunOnTaskUpdate(autoRun, enteringTerminal bool) bool {
	return autoRun && !enteringTerminal
}

func logSkippedAutoRunPrereqOnTerminal(ctx context.Context, taskID, columnName string) {
	tracelog.LogForwardStage(ctx, "task_auto_run_prereq_skipped_terminal", map[string]any{
		"task_id":              taskID,
		"progress_column_name": columnName,
	})
}

// persistAutoRunStartSkipReason stores soft-skip reason for cold-open task detail.
func persistAutoRunStartSkipReason(taskID, skipReason string) {
	tid := strings.TrimSpace(taskID)
	reason := strings.TrimSpace(skipReason)
	if db == nil || tid == "" {
		return
	}
	if _, err := db.Exec(
		`UPDATE task_tasks SET auto_run_start_skip_reason=? WHERE id=?`,
		reason, tid,
	); err != nil {
		log.Printf("[taskTaskService] persist auto_run_start_skip_reason failed task_id=%s: %v", tid, err)
	}
}

// clearAutoRunStartSkipReason clears soft-skip after a successful schedule decision.
func clearAutoRunStartSkipReason(taskID string) {
	persistAutoRunStartSkipReason(taskID, "")
}

func linkedProjectIDsFromBody(body map[string]interface{}) []string {
	projs, ok := body["projects"].([]interface{})
	if !ok || len(projs) == 0 {
		return nil
	}
	out := make([]string, 0, len(projs))
	seen := map[string]struct{}{}
	for _, raw := range projs {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		pid := strField(item, "project_id")
		if pid == "" {
			continue
		}
		if _, dup := seen[pid]; dup {
			continue
		}
		seen[pid] = struct{}{}
		out = append(out, pid)
	}
	return out
}

func linkedProjectIDsFromBodyOrTask(body map[string]interface{}, taskID, tenantID string) []string {
	if ids := linkedProjectIDsFromBody(body); len(ids) > 0 {
		return ids
	}
	rows := loadProjects(taskID, tenantID)
	out := make([]string, 0, len(rows))
	seen := map[string]struct{}{}
	for _, row := range rows {
		pid := strings.TrimSpace(fmt.Sprintf("%v", row["project_id"]))
		if pid == "" || pid == "<nil>" {
			continue
		}
		if _, dup := seen[pid]; dup {
			continue
		}
		seen[pid] = struct{}{}
		out = append(out, pid)
	}
	return out
}
