package main

import (
	"context"
	"strings"

	"tracelog"
)

func pushUniqueHTTPURL(raw any, out *[]string, seen map[string]struct{}) {
	u := strings.TrimSpace(asString(raw))
	if u == "" || !strings.HasPrefix(strings.ToLower(u), "http") {
		return
	}
	if _, ok := seen[u]; ok {
		return
	}
	seen[u] = struct{}{}
	*out = append(*out, u)
}

func asString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// collectPrHtmlUrlsFromLayers 从层图节点收集 git_remote.pr_html_url（及嵌套 children）。
func collectPrHtmlUrlsFromLayers(layers []any) []string {
	var out []string
	seen := map[string]struct{}{}
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case []any:
			for _, item := range t {
				walk(item)
			}
		case map[string]any:
			if gr, ok := t["git_remote"].(map[string]any); ok {
				pushUniqueHTTPURL(gr["pr_html_url"], &out, seen)
			}
			if pr, ok := t["pr"].(map[string]any); ok {
				pushUniqueHTTPURL(pr["html_url"], &out, seen)
			}
			if kids, ok := t["children"].([]any); ok {
				walk(kids)
			}
			if nested, ok := t["layers"].([]any); ok {
				walk(nested)
			}
		}
	}
	walk(layers)
	return out
}

func layersFromGraphDoc(doc map[string]any) []any {
	if doc == nil {
		return nil
	}
	layers, _ := doc["layers"].([]any)
	return layers
}

// ensureGitPrRepliesFromLayers 在层图已有 PR URL 时幂等补写人类 git_pr 子评论。
// 容器 auto_run 常不预创建 agent 评论，inbound git-pr-reply 可能从未发出。
func ensureGitPrRepliesFromLayers(ctx context.Context, tenantID, workspaceID, taskID, parentCommentID string, layers []any) {
	parentCommentID = strings.TrimSpace(parentCommentID)
	tenantID = strings.TrimSpace(tenantID)
	workspaceID = strings.TrimSpace(workspaceID)
	taskID = strings.TrimSpace(taskID)
	urls := collectPrHtmlUrlsFromLayers(layers)
	if parentCommentID == "" || tenantID == "" || workspaceID == "" || taskID == "" || len(urls) == 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	logLayerGraphGitPrBackfillStage(ctx, taskID, parentCommentID, len(urls))
	author, err := lookupCommentCreatedByUserID(ctx, parentCommentID)
	if err != nil {
		logInfo("event=layer_graph_git_pr_reply_author_failed parent_comment_id="+parentCommentID+" err="+err.Error(), taskID)
		return
	}
	for _, htmlURL := range urls {
		id, skipped, err := postTaskGitPrReplyComment(ctx, tenantID, workspaceID, taskID, parentCommentID, htmlURL, author)
		if err != nil {
			logInfo("event=layer_graph_git_pr_reply_failed task_id="+taskID+" parent_comment_id="+parentCommentID+" err="+err.Error(), taskID)
			continue
		}
		if skipped {
			logInfo("event=layer_graph_git_pr_reply_idempotent_skip task_id="+taskID+" comment_id="+id+" parent_comment_id="+parentCommentID, taskID)
		} else {
			logInfo("event=layer_graph_git_pr_reply_created task_id="+taskID+" comment_id="+id+" parent_comment_id="+parentCommentID, taskID)
		}
	}
}

func ensureGitPrRepliesFromGraphDoc(ctx context.Context, tenantID, workspaceID, taskID, parentCommentID string, doc map[string]any) {
	ensureGitPrRepliesFromLayers(ctx, tenantID, workspaceID, taskID, parentCommentID, layersFromGraphDoc(doc))
}

func logLayerGraphGitPrBackfillStage(ctx context.Context, taskID, commentID string, n int) {
	if n <= 0 {
		return
	}
	tracelog.LogForwardStage(ctx, "layer_graph_git_pr_reply_attempt", map[string]any{
		"task_id": taskID, "comment_id": commentID, "pr_count": n,
	})
}
