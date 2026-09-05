package main

import (
	"context"
	"encoding/json"
	"strings"
)

const humanCommentSelectCols = `id,created_by_id,content,COALESCE(mentions_json,'[]'),execution_mode,COALESCE(depends_on_comment_ids,'[]'),COALESCE(repo_identities_json,''),COALESCE(parent_comment_id,''),COALESCE(git_pr_html_url,''),COALESCE(git_pr_json,''),created_at`

type humanCommentScan struct {
	id, author, content, mentionsJSON, executionMode, dependsOnIDsJSON, identitiesJSON string
	parentCommentID, gitPRHTMLURL, gitPRJSON                                           string
}

type parsedGitPR struct {
	htmlURL  string
	provider string
	json     string
}

func parseGitPRFromBody(body map[string]interface{}) parsedGitPR {
	if body == nil {
		return parsedGitPR{}
	}
	raw, ok := body["git_pr"]
	if !ok || raw == nil {
		return parsedGitPR{}
	}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return parsedGitPR{}
	}
	htmlURL := strings.TrimSpace(strField(m, "html_url"))
	if htmlURL == "" {
		return parsedGitPR{}
	}
	provider := strings.ToLower(strings.TrimSpace(strField(m, "provider")))
	blob, _ := json.Marshal(map[string]interface{}{
		"html_url": htmlURL,
		"provider": provider,
	})
	return parsedGitPR{htmlURL: htmlURL, provider: provider, json: string(blob)}
}

func loadHumanCommentByID(id string) map[string]interface{} {
	id = strings.TrimSpace(id)
	if id == "" || db == nil {
		return nil
	}
	row := db.QueryRow(`SELECT `+humanCommentSelectCols+` FROM task_comments WHERE id=?`, id)
	item, _, err := scanHumanCommentRowFromRows(row)
	if err != nil {
		return nil
	}
	return item
}

func findCommentIDByGitPR(taskID, htmlURL string) string {
	htmlURL = strings.TrimSpace(htmlURL)
	if htmlURL == "" || db == nil {
		return ""
	}
	var id string
	err := db.QueryRow(
		`SELECT id FROM task_comments WHERE task_id=? AND git_pr_html_url=? ORDER BY created_at ASC, id ASC LIMIT 1`,
		taskID, htmlURL,
	).Scan(&id)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(id)
}

func applyGitPRToCommentItem(item map[string]interface{}, parentID, htmlURL, gitPRJSON string) {
	if parentID != "" {
		item["parent_comment_id"] = parentID
	}
	htmlURL = strings.TrimSpace(htmlURL)
	if htmlURL == "" {
		return
	}
	item["git_pr_html_url"] = htmlURL
	gp := map[string]interface{}{"html_url": htmlURL}
	if strings.TrimSpace(gitPRJSON) != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(gitPRJSON), &parsed); err == nil && parsed != nil {
			gp = parsed
			if strings.TrimSpace(strField(gp, "html_url")) == "" {
				gp["html_url"] = htmlURL
			}
		}
	}
	item["git_pr"] = gp
}

// buildGitPrReplySSEStatusData 构造任务详情 SSE 载荷，供 taskSSE 推送到已打开的任务页。
func buildGitPrReplySSEStatusData(commentID, parentID, htmlURL, provider, tenantID, workspaceID, userID string) map[string]interface{} {
	return map[string]interface{}{
		"event_name":         "task_git_pr_reply_created",
		"comment_id":         strings.TrimSpace(commentID),
		"parent_comment_id":  strings.TrimSpace(parentID),
		"git_pr_html_url":    strings.TrimSpace(htmlURL),
		"git_pr_provider":    strings.TrimSpace(provider),
		"tenant_id":          strings.TrimSpace(tenantID),
		"workspace_id":       strings.TrimSpace(workspaceID),
		"created_by_user_id": strings.TrimSpace(userID),
		"message":            "PR 回复评论已创建",
	}
}

// publishGitPrReplyCreatedSideEffects 领域审计事件 + 任务级 SSE，使未刷新的任务详情 Feed 能插入 git_pr 子评论。
func publishGitPrReplyCreatedSideEffects(ctx context.Context, taskID, tenantID, workspaceID, commentID, parentID, htmlURL, provider, userID string) {
	_ = publishDomainEventFn(ctx, "TASK_GIT_PULL_REQUEST_RECORDED", map[string]interface{}{
		"task_id":            taskID,
		"tenant_id":          tenantID,
		"workspace_id":       workspaceID,
		"comment_id":         commentID,
		"parent_comment_id":  parentID,
		"git_pr_html_url":    htmlURL,
		"git_pr_provider":    provider,
		"created_by_user_id": userID,
	}, taskID)
	statusData := buildGitPrReplySSEStatusData(commentID, parentID, htmlURL, provider, tenantID, workspaceID, userID)
	_ = publishDomainEventFn(ctx, "SSE_MESSAGE", map[string]interface{}{
		"task_id":     taskID,
		"status_data": statusData,
	}, taskID)
}
