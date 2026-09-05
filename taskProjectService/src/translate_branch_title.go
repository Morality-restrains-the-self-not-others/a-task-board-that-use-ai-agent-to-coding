package main

import (
	"net/http"
	"regexp"
	"strings"
)

var chineseTitleRe = regexp.MustCompile(`[\x{3400}-\x{9fff}]`)
var branchTitleSpaceRe = regexp.MustCompile(`[\s_]+`)
var branchTitleInvalidRe = regexp.MustCompile(`[^a-z0-9.\-]+`)
var branchTitleDashRe = regexp.MustCompile(`-{2,}`)

// Fallback 502 copy when the failure cannot be classified. Technical details stay in logs.
const translateTitleUserError = "任务标题自动翻译失败：翻译服务暂时不可用"

// translateTitleUserFacingError maps internal fanyi errors to a short user-safe "why"
// without leaking fanyi_agent / JSON / status-byte dumps.
func translateTitleUserFacingError(err error) string {
	if err == nil {
		return translateTitleUserError
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "配置不完整"):
		return "任务标题自动翻译失败：翻译服务暂未就绪"
	case strings.Contains(msg, "请求超时"):
		return "任务标题自动翻译失败：翻译服务响应超时"
	case strings.Contains(msg, "返回内容为空"):
		return "任务标题自动翻译失败：翻译服务未返回可用译文"
	case strings.Contains(msg, "响应为空"), strings.Contains(msg, "响应无效"):
		return "任务标题自动翻译失败：翻译服务响应异常"
	case strings.Contains(msg, "请求失败"), strings.Contains(msg, "读取响应失败"):
		return "任务标题自动翻译失败：翻译服务暂时不可用"
	default:
		return translateTitleUserError
	}
}

// handleTranslateBranchTitle POST /api/tenant/{tid}/projects/translate-branch-title/
// 无中文时本地规范化；含中文时直连 fanyi_agent（不再经 Django）。
func handleTranslateBranchTitle(w http.ResponseWriter, r *http.Request, tenantID string) {
	_ = tenantID
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	body, err := readJSONBody(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid json")
		return
	}
	sourceTitle := strField(body, "title")
	if sourceTitle == "" {
		writeError(w, r, http.StatusBadRequest, "title 不能为空")
		return
	}
	if !chineseTitleRe.MatchString(sourceTitle) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"source_title":     sourceTitle,
			"translated_title": sanitizeBranchTitleSegment(sourceTitle),
			"used_ai":          false,
		})
		return
	}

	traceID := r.Header.Get("X-Trace-Id")
	logInfo("translate-branch-title: calling fanyi_agent", traceID)
	translated, err := translateTitleWithFanyiAgent(r.Context(), sourceTitle)
	if err != nil {
		logError("translate-branch-title failed: "+err.Error(), traceID)
		writeError(w, r, http.StatusBadGateway, translateTitleUserFacingError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"source_title":     sourceTitle,
		"translated_title": translated,
		"used_ai":          true,
	})
}

func sanitizeBranchTitleSegment(rawTitle string) string {
	s := strings.ToLower(strings.TrimSpace(rawTitle))
	s = branchTitleSpaceRe.ReplaceAllString(s, "-")
	s = branchTitleInvalidRe.ReplaceAllString(s, "-")
	s = branchTitleDashRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		return "task"
	}
	return s
}
