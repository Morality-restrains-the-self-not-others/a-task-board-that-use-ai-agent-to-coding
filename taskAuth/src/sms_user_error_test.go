package main

import (
	"strings"
	"testing"
)

func TestFormatSMSSendUserError_AliyunAmountNotEnough(t *testing.T) {
	msg := formatSMSSendUserError(smsSendResult{
		Success:      false,
		Provider:     "aliyun",
		RequestID:    "019FEB8B-E113-5654-BA44-1D97192278F4",
		ErrorCode:    "isv.AMOUNT_NOT_ENOUGH",
		ErrorMessage: "账户余额不足",
	})
	if msg == "短信发送失败，请稍后重试" {
		t.Fatalf("expected detailed error, got generic: %q", msg)
	}
	for _, part := range []string{"账户余额不足", "isv.AMOUNT_NOT_ENOUGH", "aliyun", "019FEB8B-E113-5654-BA44-1D97192278F4"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %q", part, msg)
		}
	}
}

func TestFormatSMSSendUserError_ConfigIncomplete(t *testing.T) {
	msg := formatSMSSendUserError(smsSendResult{
		Success:      false,
		Provider:     "aliyun",
		ErrorCode:    "config_incomplete",
		ErrorMessage: "短信服务配置不完整",
	})
	for _, part := range []string{"短信服务配置不完整", "aliyun"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %q", part, msg)
		}
	}
}

func TestFormatSMSSendUserError_GenericFallback(t *testing.T) {
	msg := formatSMSSendUserError(smsSendResult{Success: false, Provider: "mock"})
	if !strings.Contains(msg, "短信发送失败") {
		t.Fatalf("got %q", msg)
	}
	if !strings.Contains(msg, "mock") {
		t.Fatalf("expected provider in fallback message, got %q", msg)
	}
}

func TestFormatSMSSendUserError_TencentError(t *testing.T) {
	msg := formatSMSSendUserError(smsSendResult{
		Success:      false,
		Provider:     "tencent",
		RequestID:    "req-1",
		ErrorCode:    "FailedOperation.InsufficientBalanceInSmsPackage",
		ErrorMessage: "短信套餐包余额不足",
	})
	for _, part := range []string{"短信套餐包余额不足", "FailedOperation.InsufficientBalanceInSmsPackage", "tencent", "req-1"} {
		if !strings.Contains(msg, part) {
			t.Fatalf("missing %q in %q", part, msg)
		}
	}
}
