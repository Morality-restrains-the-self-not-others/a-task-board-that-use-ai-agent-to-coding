package main

import (
	"net/http"
	"strings"
)

// wechatBindNextFromRequest reads the bind-flow return path from ?next=.
// Empty or unsafe values become "" and later fall back to the login bound page.
func wechatBindNextFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	return sanitizeNextPath(r.URL.Query().Get("next"))
}

// wechatBindSuccessPath is the frontend location after a successful WeChat bind.
// Empty next keeps the historical login ?wechat_bound=1 landing page.
func wechatBindSuccessPath(next string) string {
	dest := sanitizeNextPath(next)
	if dest == "" {
		return "/auth/login/?wechat_bound=1"
	}
	if strings.Contains(dest, "wechat_bound=") {
		return dest
	}
	if strings.Contains(dest, "?") {
		return dest + "&wechat_bound=1"
	}
	return dest + "?wechat_bound=1"
}
