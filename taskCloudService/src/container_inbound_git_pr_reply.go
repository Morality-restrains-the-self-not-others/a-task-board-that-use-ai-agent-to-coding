package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracelog"
)

func gitPrProviderOf(htmlURL string) string {
	u := strings.ToLower(strings.TrimSpace(htmlURL))
	if strings.Contains(u, "/-/merge_requests/") || strings.Contains(u, "gitlab") {
		return "gitlab"
	}
	return "github"
}

func buildGitPrReplyCommentURL(base, tenantID, workspaceID, taskID, parentID string) string {
	return strings.TrimRight(strings.TrimSpace(base), "/") +
		"/api/tenant_id/" + url.PathEscape(tenantID) +
		"/workspaceId/" + url.PathEscape(workspaceID) +
		"/tasks/" + url.PathEscape(taskID) +
		"/comments/" + url.PathEscape(parentID) + "/"
}

func lookupCommentCreatedByUserID(ctx context.Context, commentID string) (string, error) {
	commentID = strings.TrimSpace(commentID)
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if commentID == "" || base == "" {
		return "", fmt.Errorf("comment id and task service url required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		base+"/api/internal/comments/"+url.PathEscape(commentID)+"/created-by", nil)
	if err != nil {
		return "", err
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("created-by status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", err
	}
	uid := strings.TrimSpace(body.UserID)
	if uid == "" {
		return "", fmt.Errorf("created-by missing user_id")
	}
	return uid, nil
}

func postTaskGitPrReplyComment(ctx context.Context, tenantID, workspaceID, taskID, parentID, htmlURL, authorUserID string) (id string, skipped bool, err error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.TaskServiceURL), "/")
	if base == "" {
		return "", false, fmt.Errorf("task service url not configured")
	}
	htmlURL = strings.TrimSpace(htmlURL)
	parentID = strings.TrimSpace(parentID)
	authorUserID = strings.TrimSpace(authorUserID)
	if htmlURL == "" || parentID == "" || authorUserID == "" {
		return "", false, fmt.Errorf("html_url, parent_comment_id, author required")
	}
	payload, err := json.Marshal(map[string]interface{}{
		"content":        htmlURL,
		"execution_mode": "independent",
		"git_pr":         map[string]string{"html_url": htmlURL, "provider": gitPrProviderOf(htmlURL)},
	})
	if err != nil {
		return "", false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		buildGitPrReplyCommentURL(base, tenantID, workspaceID, taskID, parentID),
		bytes.NewReader(payload))
	if err != nil {
		return "", false, err
	}
	tracelog.ApplyOutboundHeaders(req, ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-User-Id", authorUserID)
	req.Header.Set("X-Auth-Tenant-Id", tenantID)
	if sec := strings.TrimSpace(cfg.InternalSecret); sec != "" {
		req.Header.Set("X-Internal-Secret", sec)
	}
	client := &http.Client{Transport: &http.Transport{Proxy: nil}, Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("comment create status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var created map[string]interface{}
	_ = json.Unmarshal(raw, &created)
	id = strings.TrimSpace(fmt.Sprintf("%v", created["id"]))
	if id == "<nil>" {
		id = ""
	}
	return id, resp.StatusCode == http.StatusOK, nil
}

func handleGitPrReply(w http.ResponseWriter, ctx context.Context, cfgRow *CloudServerConfig, body map[string]any, tenantID, workspaceID, taskID string) {
	htmlURL := inboundBodyString(body, "html_url")
	if htmlURL == "" {
		if gp, ok := body["git_pr"].(map[string]any); ok {
			htmlURL = inboundBodyString(gp, "html_url")
		}
	}
	parentID := inboundBodyString(body, "parent_comment_id")
	if parentID == "" && cfgRow != nil {
		parentID = strings.TrimSpace(cfgRow.CommentID)
	}
	if htmlURL == "" || parentID == "" {
		writeErrorJSON(w, nil, http.StatusBadRequest, "html_url and parent_comment_id required")
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	author, err := lookupCommentCreatedByUserID(ctx, parentID)
	if err != nil {
		logInfo("event=git_pr_reply_author_lookup_failed parent_comment_id="+parentID+" err="+err.Error(), taskID)
		writeErrorJSON(w, nil, http.StatusBadGateway, "无法解析评论作者")
		return
	}
	id, skipped, err := postTaskGitPrReplyComment(ctx, tenantID, workspaceID, taskID, parentID, htmlURL, author)
	if err != nil {
		logInfo("event=git_pr_reply_comment_failed task_id="+taskID+" parent_comment_id="+parentID+" err="+err.Error(), taskID)
		writeErrorJSON(w, nil, http.StatusBadGateway, err.Error())
		return
	}
	if skipped {
		logInfo("event=git_pr_reply_comment_idempotent_skip task_id="+taskID+" comment_id="+id+" parent_comment_id="+parentID, taskID)
	} else {
		logInfo("event=git_pr_reply_comment_created task_id="+taskID+" comment_id="+id+" parent_comment_id="+parentID, taskID)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":                true,
		"id":                id,
		"skipped":           skipped,
		"parent_comment_id": parentID,
	})
}
