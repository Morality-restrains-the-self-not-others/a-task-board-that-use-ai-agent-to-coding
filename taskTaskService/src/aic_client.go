package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var aiCommentHTTP = &http.Client{Timeout: 15 * time.Second}

// notifyContainerAgentPendingFn is replaceable in tests.
var notifyContainerAgentPendingFn = notifyContainerAgentPending

func notifyContainerAgentPending(tenantID, workspaceID, taskID, parentCommentID, imageID, imageName, content, createdBy string, source ...string) error {
	return notifyContainerAgentPendingWithModels(tenantID, workspaceID, taskID, parentCommentID, imageID, imageName, content, createdBy, nil, source...)
}

func notifyContainerAgentPendingWithModels(tenantID, workspaceID, taskID, parentCommentID, imageID, imageName, content, createdBy string, agentModels []map[string]interface{}, source ...string) error {
	// OPT-20260823-008: do not pre-create a pending container_agent row.
	// The container POSTs /container-agent-comments after bootstrap, then streams.
	src := ""
	if len(source) > 0 {
		src = strings.TrimSpace(source[0])
	}
	model := ""
	if len(agentModels) > 0 {
		model = optionalJSONString(agentModels[0], "model")
	}
	log.Printf("[taskTaskService] event=at_mention_notify_agent_deferred_to_container tenant_id=%s workspace_id=%s task_id=%s parent_comment_id=%s image_id=%s image_name=%s source=%q models=%d model=%s created_by=%s content_len=%d",
		tenantID, workspaceID, taskID, parentCommentID, imageID, imageName, src, len(agentModels), model, createdBy, len(content))
	logInfo("event=at_mention_notify_agent_deferred_to_container task_id="+taskID+" comment_id="+parentCommentID, taskID)
	return nil
}

func setAICommentInternalSecret(req *http.Request) {
	if cfg.TaskAICommentInternalSecret != "" {
		req.Header.Set("X-TaskAIComment-Internal-Secret", cfg.TaskAICommentInternalSecret)
	} else if cfg.InternalSecret != "" {
		req.Header.Set("X-TaskAIComment-Internal-Secret", cfg.InternalSecret)
	}
}

type commentPageResult struct {
	results    []map[string]interface{}
	hasMore    bool
	nextCursor interface{}
}

func buildCommentListURL(base, path string, limit int) string {
	u := strings.TrimRight(base, "/") + path
	if limit > 0 {
		u += "?limit=" + strconv.Itoa(limit)
	} else {
		u += "?full=1"
	}
	return u
}

func fetchAICommentsForTask(taskID string, limit int) (commentPageResult, error) {
	base := strings.TrimSpace(cfg.AICommentServiceURL)
	if base == "" {
		return commentPageResult{results: []map[string]interface{}{}}, nil
	}
	url := buildCommentListURL(
		base,
		"/api/internal/task-ai-comment/tasks/"+url.PathEscape(taskID)+"/ai-comments",
		limit,
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return commentPageResult{}, err
	}
	setAICommentInternalSecret(req)
	resp, err := aiCommentHTTP.Do(req)
	if err != nil {
		log.Printf("[taskTaskService] event=fetch_ai_comments_failed task_id=%s err=%v", taskID, err)
		return commentPageResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Printf("[taskTaskService] event=fetch_ai_comments_http task_id=%s status=%d body=%s", taskID, resp.StatusCode, string(raw))
		return commentPageResult{}, fmt.Errorf("aiComment list status %d", resp.StatusCode)
	}
	return decodeCommentPageJSON(raw)
}

func fetchContainerAgentCommentsForTask(taskID string, limit int) (commentPageResult, error) {
	base := strings.TrimSpace(cfg.AICommentServiceURL)
	if base == "" {
		return commentPageResult{results: []map[string]interface{}{}}, nil
	}
	url := buildCommentListURL(
		base,
		"/api/internal/task-ai-comment/tasks/"+url.PathEscape(taskID)+"/container-agent-comments",
		limit,
	)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return commentPageResult{}, err
	}
	setAICommentInternalSecret(req)
	resp, err := aiCommentHTTP.Do(req)
	if err != nil {
		log.Printf("[taskTaskService] event=fetch_container_agent_comments_failed task_id=%s err=%v", taskID, err)
		return commentPageResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		log.Printf("[taskTaskService] event=fetch_container_agent_comments_http task_id=%s status=%d body=%s", taskID, resp.StatusCode, string(raw))
		return commentPageResult{}, fmt.Errorf("container agent list status %d", resp.StatusCode)
	}
	return decodeCommentPageJSON(raw)
}

func decodeCommentPageJSON(raw []byte) (commentPageResult, error) {
	var arr []map[string]interface{}
	if err := json.Unmarshal(raw, &arr); err == nil {
		if arr == nil {
			arr = []map[string]interface{}{}
		}
		return commentPageResult{results: arr}, nil
	}
	var page struct {
		Results    []map[string]interface{} `json:"results"`
		HasMore    bool                     `json:"has_more"`
		NextCursor interface{}              `json:"next_cursor"`
	}
	if err := json.Unmarshal(raw, &page); err != nil {
		return commentPageResult{}, err
	}
	if page.Results == nil {
		page.Results = []map[string]interface{}{}
	}
	return commentPageResult{
		results:    page.Results,
		hasMore:    page.HasMore,
		nextCursor: page.NextCursor,
	}, nil
}

func decodeCommentListJSON(raw []byte) ([]map[string]interface{}, error) {
	page, err := decodeCommentPageJSON(raw)
	if err != nil {
		return nil, err
	}
	return page.results, nil
}
