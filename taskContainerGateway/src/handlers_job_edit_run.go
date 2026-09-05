package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func jobEditRunReadTimeoutSec() float64 {
	raw := strings.TrimSpace(os.Getenv("CONTAINER_JOB_EDIT_RUN_READ_TIMEOUT"))
	if raw == "" {
		return 120
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 120
	}
	return v
}

func handleContainerJobEditRun(
	w http.ResponseWriter,
	r *http.Request,
	sc scope,
	target containerTarget,
	rawBody []byte,
) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "method not allowed, use POST"})
		return
	}
	ctx := r.Context()
	body := parseJSONBody(rawBody)
	jobID := strField(body, "job_id")
	command := strField(body, "command")
	commandKind := strings.ToLower(strField(body, "command_kind"))
	if commandKind == "" {
		commandKind = "trae"
	}
	if jobID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "job_id 必填"})
		return
	}
	if command == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "command 不能为空"})
		return
	}
	if commandKind != "trae" && commandKind != "shell" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "command_kind 须为 trae 或 shell"})
		return
	}
	autoIter, hasAutoIter, errDetail := parseOptionalPositiveInt(body["agent_auto_iteration_count"])
	if errDetail != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": errDetail})
		return
	}

	root := scopedAPIRoot(target.BaseURL, sc)
	readSec := jobEditRunReadTimeoutSec()
	fwd := func(method, url string, reqBody []byte) (int, []byte) {
		st, resp, _ := forwardToOnlineServiceTimeout(ctx, method, url, target.AccessToken, reqBody, readSec)
		return st, resp
	}

	jobsURL := root + "/jobs"
	st, respBody := fwd(http.MethodGet, jobsURL, nil)
	jobsPayload, jobs, listErr := parseJobsListResponse(st, respBody, jobsURL)
	if listErr != "" {
		writeJSON(w, http.StatusBadGateway, jobsPayload)
		return
	}
	byID := map[string]map[string]any{}
	for _, j := range jobs {
		id := strField(j, "id")
		if id != "" {
			byID[id] = j
		}
	}
	targetJob, found := byID[jobID]
	if !found {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "目标任务不存在"})
		return
	}
	if strField(targetJob, "command_kind") == "clone" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "克隆任务不支持修改后执行"})
		return
	}

	childMap := map[string][]string{}
	for _, j := range jobs {
		jid := strField(j, "id")
		pid := strField(j, "parent_job_id")
		if jid != "" && pid != "" {
			childMap[pid] = append(childMap[pid], jid)
		}
	}
	descendants := collectDescendants(jobID, childMap)
	pending := map[string]struct{}{}
	for _, id := range descendants {
		pending[id] = struct{}{}
	}
	for len(pending) > 0 {
		progressed := false
		for jid := range pending {
			hasChild := false
			for _, ch := range childMap[jid] {
				if _, still := pending[ch]; still {
					hasChild = true
					break
				}
			}
			if hasChild {
				continue
			}
			delURL := root + "/jobs/" + pathEscape(jid)
			dst, dbody := fwd(http.MethodDelete, delURL, nil)
			if dst >= 400 {
				detail := upstreamDetail(dbody)
				writeJSON(w, http.StatusBadGateway, buildUpstreamHTTPError(
					http.MethodDelete, delURL, dst, fmt.Sprintf("删除子任务失败(%s): %s", jid, detail),
				))
				return
			}
			delete(pending, jid)
			progressed = true
		}
		if !progressed {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "删除子任务失败：检测到环或状态异常"})
			return
		}
	}

	selfURL := root + "/jobs/" + pathEscape(jobID)
	sst, sbody := fwd(http.MethodDelete, selfURL, nil)
	if sst >= 400 {
		detail := upstreamDetail(sbody)
		writeJSON(w, http.StatusBadGateway, buildUpstreamHTTPError(
			http.MethodDelete, selfURL, sst, fmt.Sprintf("删除当前任务失败: %s", detail),
		))
		return
	}

	createPayload := map[string]any{
		"command":                 command,
		"command_kind":            commandKind,
		"edit_run_delivery":       true,
		"auto_run_commit_message": editRunCommitMessage(command),
	}
	if hasAutoIter {
		createPayload["env"] = map[string]any{"TASK_AGENT_MAX_STEPS": strconv.Itoa(autoIter)}
	}
	parentJobID := strField(targetJob, "parent_job_id")
	repoLayerID := strField(targetJob, "repo_layer_id")
	if parentJobID != "" {
		createPayload["parent_job_id"] = parentJobID
	} else if repoLayerID != "" {
		createPayload["repo_layer_id"] = repoLayerID
	}
	if parentCommentID := strField(body, "parent_comment_id"); parentCommentID != "" {
		createPayload["mounted_parent_comment_id"] = parentCommentID
	}
	if imageID := strField(body, "installed_image_id"); imageID != "" {
		createPayload["edit_run_installed_image_id"] = imageID
	}
	createBody, err := json.Marshal(createPayload)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "failed to encode create body"})
		return
	}
	cst, cbody := fwd(http.MethodPost, jobsURL, createBody)
	if cst >= 400 {
		detail := upstreamDetail(cbody)
		writeJSON(w, http.StatusBadGateway, buildUpstreamHTTPError(http.MethodPost, jobsURL, cst, detail))
		return
	}
	var created any
	if err := json.Unmarshal(cbody, &created); err != nil {
		created = map[string]any{}
	}
	if createdMap, ok := created.(map[string]any); ok {
		if newJID := strField(createdMap, "id"); newJID != "" {
			startJobStream(ctx, sc, target, newJID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                  true,
		"job":                 created,
		"deleted_descendants": len(descendants),
	})
}

// editRunCommitMessage 取指令首行作交付 commit message（截断）。
func editRunCommitMessage(command string) string {
	s := strings.TrimSpace(command)
	if s == "" {
		return "edit_run"
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	const max = 72
	runes := []rune(s)
	if len(runes) > max {
		s = string(runes[:max])
	}
	if s == "" {
		return "edit_run"
	}
	return s
}

func parseOptionalPositiveInt(raw any) (int, bool, string) {
	if raw == nil {
		return 0, false, ""
	}
	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, false, ""
		}
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return 0, false, "agent_auto_iteration_count 须为正整数"
		}
		return n, true, ""
	case float64:
		n := int(v)
		if float64(n) != v || n <= 0 {
			return 0, false, "agent_auto_iteration_count 须为正整数"
		}
		return n, true, ""
	case json.Number:
		n, err := v.Int64()
		if err != nil || n <= 0 {
			return 0, false, "agent_auto_iteration_count 须为正整数"
		}
		return int(n), true, ""
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" || s == "<nil>" {
			return 0, false, ""
		}
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			return 0, false, "agent_auto_iteration_count 须为正整数"
		}
		return n, true, ""
	}
}

func parseJobsListResponse(status int, body []byte, url string) (map[string]any, []map[string]any, string) {
	if status >= 400 {
		detail := "拉取容器任务列表失败"
		text := strings.TrimSpace(string(body))
		if text != "" {
			detail = detail + ": " + truncateRunes(text, 500)
		} else {
			detail = fmt.Sprintf("%s: HTTP %d", detail, status)
		}
		code := status
		if code < 400 {
			code = http.StatusBadGateway
		}
		return buildUpstreamHTTPError(http.MethodGet, url, code, detail), nil, detail
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		detail := "拉取容器任务列表失败"
		return buildUpstreamHTTPError(http.MethodGet, url, http.StatusBadGateway, detail), nil, detail
	}
	rawJobs, ok := parsed["jobs"].([]any)
	if !ok {
		detail := "拉取容器任务列表失败"
		return buildUpstreamHTTPError(http.MethodGet, url, http.StatusBadGateway, detail), nil, detail
	}
	out := make([]map[string]any, 0, len(rawJobs))
	for _, item := range rawJobs {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return nil, out, ""
}

func collectDescendants(rootID string, childMap map[string][]string) []string {
	var out []string
	queue := []string{rootID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, ch := range childMap[cur] {
			out = append(out, ch)
			queue = append(queue, ch)
		}
	}
	return out
}

func upstreamDetail(body []byte) string {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err == nil {
		if d := strField(parsed, "detail"); d != "" {
			return d
		}
	}
	return truncateRunes(string(body), 500)
}

func buildUpstreamHTTPError(method, url string, statusCode int, detail string) map[string]any {
	detailText := truncateRunes(detail, 1200)
	return map[string]any{
		"detail":      fmt.Sprintf("容器接口返回错误: %s %s -> HTTP %d; detail=%s", method, url, statusCode, detailText),
		"http_status": statusCode,
		"upstream": map[string]any{
			"method": method,
			"url":    url,
			"status": statusCode,
			"detail": detailText,
		},
	}
}

func truncateRunes(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}
