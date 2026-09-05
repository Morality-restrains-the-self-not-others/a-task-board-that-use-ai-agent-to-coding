package main

import "strings"

const startVmErrorNotEnoughBalance = "阿里云账户余额不足，无法创建云实例。请充值后再重试。"

// humanizeStartVmError 将云厂商 SDK 原文转成工作台可读中文。
// 未匹配的原文原样返回，避免吞掉排障信息。
func humanizeStartVmError(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	if isAliyunNotEnoughBalanceError(raw) {
		return startVmErrorNotEnoughBalance
	}
	return raw
}

func isAliyunNotEnoughBalanceError(raw string) bool {
	if strings.Contains(raw, "NotEnoughBalance") {
		return true
	}
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "notenoughbalance") {
		return true
	}
	if strings.Contains(lower, "does not have enough balance") {
		return true
	}
	return strings.Contains(raw, "余额不足")
}
