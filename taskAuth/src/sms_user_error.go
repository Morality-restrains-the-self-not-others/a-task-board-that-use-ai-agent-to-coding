package main

import (
	"fmt"
	"strings"
)

// formatSMSSendUserError builds a user-visible SMS failure message with
// provider reason/code/request id when available (no secrets).
func formatSMSSendUserError(sms smsSendResult) string {
	reason := strings.TrimSpace(sms.ErrorMessage)
	if reason == "" {
		reason = smsFailureReasonHint(sms.ErrorCode)
	}
	if reason == "" {
		reason = "请稍后重试"
	}

	var parts []string
	parts = append(parts, "短信发送失败："+reason)
	if code := strings.TrimSpace(sms.ErrorCode); code != "" {
		parts = append(parts, "错误码 "+code)
	}
	if provider := strings.TrimSpace(sms.Provider); provider != "" {
		parts = append(parts, "渠道 "+provider)
	}
	if reqID := strings.TrimSpace(sms.RequestID); reqID != "" {
		parts = append(parts, "流水号 "+reqID)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("%s（%s）", parts[0], strings.Join(parts[1:], "，"))
}

func smsFailureReasonHint(code string) string {
	switch strings.TrimSpace(code) {
	case "isv.AMOUNT_NOT_ENOUGH", "FailedOperation.InsufficientBalanceInSmsPackage":
		return "短信服务账户余额不足，请联系管理员充值"
	case "config_incomplete":
		return "短信服务配置不完整，请联系管理员"
	case "unsupported_provider":
		return "不支持的短信渠道，请联系管理员"
	case "http_error", "request_build_error", "marshal_error":
		return "短信服务暂时不可用，请稍后重试"
	default:
		return ""
	}
}
