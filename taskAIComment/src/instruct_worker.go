package main

import (
	"context"
	"log"
	"strings"
	"time"
)

type instructStreamParams struct {
	CommentID           string
	TenantID            string
	WorkspaceID         string
	TaskID              string
	UserContent         string
	ContainerJobContext map[string]interface{}
	TraceID             string
}

func startInstructStreamAsync(params instructStreamParams) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3700*time.Second)
		defer cancel()
		runInstructStreamForComment(ctx, params)
	}()
}

func runInstructStreamForComment(ctx context.Context, params instructStreamParams) {
	commentID := strings.TrimSpace(params.CommentID)
	tenantID := strings.TrimSpace(params.TenantID)
	workspaceID := strings.TrimSpace(params.WorkspaceID)
	taskID := strings.TrimSpace(params.TaskID)
	userContent := strings.TrimSpace(params.UserContent)
	traceID := strings.TrimSpace(params.TraceID)

	jobCtx := parseContainerJobContext(params.ContainerJobContext)
	var merged strings.Builder
	streamOK := false
	assistantText := ""

	cfgRow, err := lookupCloudServerConfig(ctx, tenantID, workspaceID, taskID)
	baseURL, token := "", ""
	if cfgRow != nil {
		baseURL, token = resolveContainerTarget(cfgRow)
	}
	streamBackend := detectStreamBackend(baseURL, token, traceID)
	instructURL := instructErrorAPIRef(baseURL, tenantID, workspaceID, taskID, streamBackend)

	if err != nil || cfgRow == nil || strings.TrimSpace(baseURL) == "" {
		errMsg := "[后端代理错误] 任务云配置中缺少可访问的 server_url\n"
		publishAIInstructStreamSSE(ctx, taskID, commentID, "error", errMsg, nil, instructURL, traceID)
		assistantText = errMsg
	} else {
		result := openContainerInstructStream(
			ctx, baseURL, token, streamBackend,
			tenantID, workspaceID, taskID, commentID, userContent, traceID, jobCtx,
		)
		if result.Err != nil {
			errMsg := "\n[后端代理错误] api=" + instructURL + " error=" + result.Err.Error() + "\n"
			publishAIInstructStreamSSE(ctx, taskID, commentID, "error", errMsg, nil, instructURL, traceID)
			assistantText = errMsg
		} else if result.StatusCode >= 400 {
			st := result.StatusCode
			errMsg := "[容器响应错误] HTTP " + itoa(st) + " api=" + instructURL + "\n"
			publishAIInstructStreamSSE(ctx, taskID, commentID, "error", errMsg, &st, instructURL, traceID)
			assistantText = errMsg
		} else if result.Chunks != nil {
			streamOK = true
			batcher := newAIInstructChunkBatcher(ctx, taskID, commentID, traceID)
			for chunk := range result.Chunks {
				if len(chunk) == 0 {
					continue
				}
				merged.Write(chunk)
				batcher.AddChunk(string(chunk))
			}
			batcher.Flush()
			assistantText = merged.String()
		}
		if result != nil && result.Close != nil {
			result.Close()
		}
	}

	payload := map[string]interface{}{
		"ai_comment_id":     commentID,
		"task_id":           taskID,
		"tenant_id":         tenantID,
		"workspace_id":      workspaceID,
		"user_content":      userContent,
		"assistant_text":    assistantText,
		"stream_ok":         streamOK,
		"client_abandoned":  false,
	}
	if pubErr := publishDomainEvent(ctx, "AI_ASSISTANT_REPLY_COMPLETED", payload, commentID); pubErr != nil {
		log.Printf("[taskAIComment] AI_ASSISTANT_REPLY_COMPLETED failed comment_id=%s: %v", commentID, pubErr)
	}
	publishAIInstructStreamSSE(ctx, taskID, commentID, "done", "", nil, "", traceID)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
