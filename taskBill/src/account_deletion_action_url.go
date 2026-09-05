package main

import (
	"fmt"
	"strconv"
	"strings"
)

// 注销阻断项的 action_url 必须是 taskFE 已注册的页面路由。
// 未注册路径会被 router.js catch-all 重定向到首页，用户无法「前往处理」。

func billingDashboardActionURL(tenantID string) string {
	return fmt.Sprintf("/tenant/%s/billing/", strings.TrimSpace(tenantID))
}

func billingOrdersActionURL(tenantID string) string {
	return fmt.Sprintf("/tenant/%s/billing/orders/", strings.TrimSpace(tenantID))
}

func gitlabResourceActionURL(tenantID string) string {
	return fmt.Sprintf("/tenant/%s/settings/gitlab-connection/", strings.TrimSpace(tenantID))
}

func pendingPaymentActionURL(tenantID, orderID int64) string {
	if tenantID <= 0 {
		return "/profile/"
	}
	tid := strconv.FormatInt(tenantID, 10)
	if orderID > 0 {
		return fmt.Sprintf("/tenant/%s/billing/orders/%d/", tid, orderID)
	}
	return billingDashboardActionURL(tid)
}
