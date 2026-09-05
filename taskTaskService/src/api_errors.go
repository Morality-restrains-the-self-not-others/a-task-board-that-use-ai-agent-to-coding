package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"tracelog"
)

// 用户可见 API 错误文案（中文直出，减少前后端双份维护）。
const (
	errMsgForbidden                     = "您没有权限修改此任务"
	errMsgForbiddenRead                 = "您没有权限查看此任务"
	errMsgUnauthorized                  = "未登录或登录已过期，请重新登录"
	errMsgNotFound                      = "资源不存在或已被删除"
	errMsgInvalidJSON                   = "请求数据格式无效，请刷新页面后重试"
	errMsgMethodNotAllowed              = "请求方法不被允许"
	errMsgTaskIDRequired                = "缺少任务 ID"
	errMsgWorkspaceIDRequired           = "缺少工作空间 ID"
	errMsgTitleRequired                 = "标题不能为空"
	errMsgIDsRequired                   = "请选择要操作的任务"
	errMsgProjectsRequired              = "请关联项目"
	errMsgContentRequired               = "评论内容不能为空"
	errMsgWorkspaceUnavailable          = "工作空间不可用"
	errMsgWorkspaceNotFound             = "工作空间不存在"
	errMsgWorkspaceAccessUnavailable    = "无法访问该工作空间"
	errMsgSearchAccessLookupFailed      = "无法校验工作空间访问权限，请稍后重试"
	errMsgProjectNotFound               = "项目不存在"
	errMsgInstalledImageNotFound        = "未找到已安装镜像"
	errMsgInstalledImageLookupFailed    = "查询已安装镜像失败"
	errMsgFailedToCreateComment         = "创建评论失败"
	errMsgParentCommentNotFound         = "父评论不存在或不属于该任务"
	errMsgCommentContentIsTaskID        = "评论内容不能为任务 ID"
	errMsgLimitPositiveInteger          = "分页数量须为正整数"
	errMsgRepoCloneIdentitiesObject     = "仓库克隆 Git 身份配置格式无效"
	errMsgInvalidFeatureParamsSource    = "特性参数来源无效"
	errMsgInvalidProgressColumn         = "进度列无效"
	errMsgInvalidAssignee               = "协作者无效"
	errMsgInvalidDeliverable            = "交付物无效"
	errMsgValidationFailed              = "校验失败"
	errMsgMemberNotFound                = "成员不存在"
	errMsgOneProjectOnly                = "一个任务只能关联一个项目"
	errMsgTitleEmptyZH                  = "title 不能为空"
	errMsgTranslateTitleFailed          = "任务标题翻译失败"
	errMsgWorkspaceAutoScheduleDisabled = "当前工作空间尚未启用自动调度。请先前往「自动调度安排」开启后再加入队列。"
)

// httpStatusError carries an HTTP status for API-facing domain errors.
type httpStatusError struct {
	status int
	msg    string
}

func (e *httpStatusError) Error() string {
	if e == nil {
		return ""
	}
	return e.msg
}

func (e *httpStatusError) HTTPStatus() int {
	if e == nil || e.status == 0 {
		return http.StatusInternalServerError
	}
	return e.status
}

type httpStatuser interface {
	HTTPStatus() int
}

func httpStatusOf(err error, fallback int) int {
	var hs httpStatuser
	if errors.As(err, &hs) {
		return hs.HTTPStatus()
	}
	return fallback
}

func errWorkspaceAutoScheduleDisabled() error {
	return &httpStatusError{status: http.StatusConflict, msg: errMsgWorkspaceAutoScheduleDisabled}
}

// traceIDForError resolves a trace id for error-body injection:
// X-Trace-Id header first (gateway-bridged), then context trace id.
func traceIDForError(r *http.Request) string {
	if r == nil {
		return ""
	}
	if tid := strings.TrimSpace(r.Header.Get("X-Trace-Id")); tid != "" {
		return tid
	}
	return strings.TrimSpace(tracelog.TraceIDFromContext(r.Context()))
}

// writeError writes a uniform {status, error, message, trace_id?} error body.
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	body := map[string]interface{}{"status": "error", "error": message, "message": message}
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorDetail writes a {status, error, detail, message, trace_id?} error body.
// Frontend reads data.detail first (taskAuth FE error contract), so detail-first.
func writeErrorDetail(w http.ResponseWriter, r *http.Request, status int, detail string) {
	body := map[string]interface{}{
		"status":  "error",
		"error":   detail,
		"detail":  detail,
		"message": detail,
	}
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	writeJSON(w, status, body)
}

// writeErrorMap writes a composite error body, injecting trace_id and filling
// in status/message (from error or detail) when absent while preserving any
// extra keys the caller provided.
func writeErrorMap(w http.ResponseWriter, r *http.Request, status int, body map[string]interface{}) {
	if tid := traceIDForError(r); tid != "" {
		body["trace_id"] = tid
	}
	if _, ok := body["status"]; !ok {
		body["status"] = "error"
	}
	if _, ok := body["message"]; !ok {
		if msg, ok := body["error"]; ok {
			body["message"] = msg
		} else if detail, ok := body["detail"]; ok {
			body["message"] = detail
		}
	}
	writeJSON(w, status, body)
}

func errProjectsMustBeObject(index int) error {
	return fmt.Errorf("关联项目第 %d 项数据格式无效", index+1)
}

func errProjectsBaseBranchRequired(index int) error {
	return fmt.Errorf("请为关联项目第 %d 个仓库配置基准分支", index+1)
}

func errProjectsDuplicateRepoIndex(index, repoIndex int) error {
	return fmt.Errorf("关联项目第 %d 项仓库索引重复（repo_index=%d）", index+1, repoIndex)
}
