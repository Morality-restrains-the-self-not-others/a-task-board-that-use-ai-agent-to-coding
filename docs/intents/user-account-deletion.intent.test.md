# Test Intent: 用户账号注销（PIPL）

- **Status:** accepted
- **Date:** 2026-08-20
- **Paired intent:** `docs/intents/user-account-deletion.intent.md`

## 覆盖点

1. Profile 危险区展示 blockers 或注销表单
2. 「前往处理」为真实 `<a href>`，且 href 为已注册 Vue 路由
3. GitLab 有效订阅不得链到 `/billing/gitlab-resources/`（会踢回首页），应到 `/settings/gitlab-connection/`
4. 待完成支付不得链到 `/profile/`，应到租户账单或订单详情
5. 已退款（及已支付/已取消/已过期）订单不得出现在「有待完成的支付」阻断项
6. 待处理成员邀请不得作为注销硬阻断（`TENANT_INVITE_PENDING` 不得出现在 precheck blockers）
7. 冷却期状态可撤回

## 对应测试

- `taskBill/src/account_deletion_action_url_test.go`
- `taskBill/src/account_deletion_blockers_test.go`
- `taskTenantService/src/account_deletion_action_url_test.go`
- `taskTenantService/src/account_deletion_blockers_test.go`
- `taskCloudService/src/account_deletion_action_url_test.go`
- `taskFE/app/src/utils/accountDeletionActionUrl.test.js`
- `taskFE/app/src/components/UserProfileAccountDeletionPanel.actionUrl.test.js`
- `taskFE/app/src/router/tenantRoutes.accountDeletionRedirect.test.js`
