package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"tracelog"
)

const maxGitLabAPIMessageLen = 300

// parseGitLabAPIMessage extracts GitLab REST error "message" or OAuth error_description.
func parseGitLabAPIMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	var payload struct {
		Message          json.RawMessage `json:"message"`
		Error            string          `json:"error"`
		ErrorDescription string          `json:"error_description"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return truncateGitLabAPIMessage(trimmed)
	}
	if msg := gitlabMessageFromRaw(payload.Message); msg != "" {
		return truncateGitLabAPIMessage(msg)
	}
	if desc := strings.TrimSpace(payload.ErrorDescription); desc != "" {
		return truncateGitLabAPIMessage(desc)
	}
	if errName := strings.TrimSpace(payload.Error); errName != "" {
		return truncateGitLabAPIMessage(errName)
	}
	return truncateGitLabAPIMessage(trimmed)
}

func gitlabMessageFromRaw(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	// GitLab sometimes returns {"message":{"base":["..."]}} or other objects.
	compact := strings.TrimSpace(string(raw))
	return compact
}

func truncateGitLabAPIMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = strings.TrimSpace(msg[:i])
	}
	if len(msg) > maxGitLabAPIMessageLen {
		return msg[:maxGitLabAPIMessageLen] + "…"
	}
	return msg
}

// formatGitLabAPIError builds a user-facing error that includes GitLab's own reason.
func formatGitLabAPIError(statusCode int, body []byte) string {
	msg := parseGitLabAPIMessage(body)
	switch statusCode {
	case http.StatusUnauthorized:
		if msg != "" {
			return fmt.Sprintf("GitLab API 拒绝访问（401）：%s。OAuth 令牌可能无效或 scope 不足，请在项目页重新 OAuth 授权或检查 Git 网站授权设置。", msg)
		}
		return "GitLab API 拒绝访问（401）：OAuth 令牌无效或 scope 不足，请在项目页重新 OAuth 授权或检查 Git 网站授权设置。"
	case http.StatusForbidden:
		if msg == "" {
			return "GitLab API 访问被拒绝（403）。请确认账号对该仓库有权限，或重新完成 OAuth 授权。"
		}
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "rate limit") {
			return fmt.Sprintf("GitLab API 限流（403）：%s", msg)
		}
		return fmt.Sprintf("GitLab API 访问被拒绝（403）：%s", msg)
	case http.StatusTooManyRequests:
		if msg != "" {
			return fmt.Sprintf("GitLab API 限流（429）：%s", msg)
		}
		return "GitLab API 限流（429）：请稍后重试"
	case http.StatusNotFound:
		if msg != "" {
			return fmt.Sprintf("GitLab repository not found（404）：%s", msg)
		}
		return "GitLab repository not found"
	default:
		if msg != "" {
			return fmt.Sprintf("GitLab API error: %d — %s", statusCode, msg)
		}
		return fmt.Sprintf("GitLab API error: %d", statusCode)
	}
}

func logGitLabAPIFailure(op, repoURL string, statusCode int, body []byte, traceHeaders ...map[string]string) {
	msg := parseGitLabAPIMessage(body)
	if msg == "" {
		msg = "(empty)"
	}
	tracelog.EmitWithTrace(traceIDFromTraceHeaders(traceHeaders...), "error", "gitlab-api", "taskProjectService", map[string]string{
		"op":     op,
		"status": strconv.Itoa(statusCode),
		"repo":   redactRepoURLForLog(repoURL),
		"msg":    msg,
	})
}

// traceIDFromTraceHeaders extracts the outbound trace id from the first non-empty
// trace-header map (X-Trace-Id, falling back to traceparent) so structured log
// lines can be correlated via Loki `{job=~".+"} | json | trace_id=`.
func traceIDFromTraceHeaders(traceHeaders ...map[string]string) string {
	trace := outboundTrace(traceHeaders...)
	if len(trace) == 0 {
		return ""
	}
	if tid := strings.TrimSpace(trace["X-Trace-Id"]); tid != "" {
		return tid
	}
	if tp := strings.TrimSpace(trace["traceparent"]); tp != "" {
		if tid, _, ok := tracelog.ParseTraceParent(tp); ok {
			return tid
		}
	}
	return ""
}

// gitlabProjectResponseAllowsPush reports Developer+ (access_level >= 30) on
// project_access or group_access. GitLab GET /projects/:id includes permissions
// when the request is authenticated: https://docs.gitlab.com/ee/api/projects.html
// Missing permissions is treated as unknown (do not false-deny).
func gitlabProjectResponseAllowsPush(body []byte) bool {
	var payload struct {
		Permissions *struct {
			ProjectAccess *struct {
				AccessLevel int `json:"access_level"`
			} `json:"project_access"`
			GroupAccess *struct {
				AccessLevel int `json:"access_level"`
			} `json:"group_access"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Permissions == nil {
		return true
	}
	const developerAccess = 30
	if payload.Permissions.ProjectAccess != nil && payload.Permissions.ProjectAccess.AccessLevel >= developerAccess {
		return true
	}
	if payload.Permissions.GroupAccess != nil && payload.Permissions.GroupAccess.AccessLevel >= developerAccess {
		return true
	}
	return false
}
