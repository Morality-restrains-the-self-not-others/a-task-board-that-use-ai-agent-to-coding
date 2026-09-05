package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

const maxGitHubAPIMessageLen = 300

// parseGitHubAPIMessage extracts GitHub REST error "message" (or plain-text body).
func parseGitHubAPIMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err == nil {
		return truncateGitHubAPIMessage(strings.TrimSpace(payload.Message))
	}
	return truncateGitHubAPIMessage(trimmed)
}

func truncateGitHubAPIMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	// Strip HTML unicorn / gateway pages — keep first line only.
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = strings.TrimSpace(msg[:i])
	}
	if len(msg) > maxGitHubAPIMessageLen {
		return msg[:maxGitHubAPIMessageLen] + "…"
	}
	return msg
}

// formatGitHubAPIError builds a user-facing error that includes GitHub's own reason.
func formatGitHubAPIError(statusCode int, body []byte) string {
	msg := parseGitHubAPIMessage(body)
	switch statusCode {
	case http.StatusUnauthorized:
		if msg != "" {
			return fmt.Sprintf("GitHub API 拒绝访问（401）：%s。access_token 可能无效或已过期，请解除并重新绑定 GitHub。", msg)
		}
		return "GitHub API 拒绝访问：access_token 无效或已过期，请解除并重新绑定 GitHub。"
	case http.StatusForbidden:
		if msg == "" {
			return "GitHub API 访问被拒绝（403）。若为组织仓库可能需要 SSO 或在组织中批准该 GitHub App。"
		}
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "rate limit") {
			return fmt.Sprintf("GitHub API 限流（403）：%s", msg)
		}
		if strings.Contains(lower, "saml") || strings.Contains(lower, "sso") ||
			strings.Contains(lower, "organization") || strings.Contains(lower, "approved by") {
			return fmt.Sprintf("GitHub API 访问被拒绝（403）：%s。若为组织仓库可能需要 SSO 或在组织中批准该 GitHub App。", msg)
		}
		return fmt.Sprintf("GitHub API 访问被拒绝（403）：%s", msg)
	case http.StatusNotFound:
		if msg != "" {
			return fmt.Sprintf("GitHub repository not found（404）：%s", msg)
		}
		return "GitHub repository not found"
	default:
		if msg != "" {
			return fmt.Sprintf("GitHub API error: %d — %s", statusCode, msg)
		}
		return fmt.Sprintf("GitHub API error: %d", statusCode)
	}
}

func logGitHubAPIFailure(op, repoURL string, statusCode int, body []byte) {
	msg := parseGitHubAPIMessage(body)
	if msg == "" {
		msg = "(empty)"
	}
	log.Printf("[taskProjectService] github-api op=%s status=%d repo=%s msg=%s",
		op, statusCode, redactRepoURLForLog(repoURL), msg)
}

// githubRepoResponseAllowsPush reports whether GET /repos JSON grants write
// (admin / maintain / push). Official schema: permissions object on the
// repository resource when the request is authenticated.
// https://docs.github.com/en/rest/repos/repos#get-a-repository
// Missing permissions is treated as unknown (do not false-deny).
func githubRepoResponseAllowsPush(body []byte) bool {
	var payload struct {
		Permissions *struct {
			Admin    bool `json:"admin"`
			Maintain bool `json:"maintain"`
			Push     bool `json:"push"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Permissions == nil {
		return true
	}
	p := payload.Permissions
	return p.Admin || p.Maintain || p.Push
}
