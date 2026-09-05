# Intent: 用户账号注销（PIPL）

- **Status:** accepted
- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-user-account-deletion-pipl-design.md`

## 用户故事

作为已登录用户，我可以在个人资料页发起账号注销申请；系统在冷却期结束后自动归档账号并脱敏 PII，期间可撤回。

## 验收要点

1. Profile 危险区展示 precheck 阻断项或提交表单（确认「注销」+ 密码）
2. 冷却期默认 15 天（`conf/auth/task-auth/config.yaml` `accountDeletion.cooldownDays`）
3. 三项硬门禁：资金/订单/退款、唯一租户管理员、运行中云资源。**待处理成员邀请不是硬门禁**（属租户级、可随注销失效；不得下发 `TENANT_INVITE_PENDING`）
4. 阻断项「前往处理」的 `action_url` 必须是 taskFE 已注册路由（GitLab → `/tenant/{id}/settings/gitlab-connection/`，待支付 → 账单/订单页）；禁止指向会被 catch-all 踢回首页的假路径
5. `BILLING_PAYMENT_PENDING` 仅包含 `billing_payment_pending.status='pending'` 且关联订单仍为 `pending`（无订单的充值二维码同样仅 pending）；已退款/已支付/已取消/已过期订单不得作为「有待完成的支付」
6. `taskEvents` timer 周期性调用 `POST /api/internal/taskauth/account-deletion/execute-due/`
7. 完成后 `auth_user.is_archived=1`、登录拦截生效、PII 脱敏

## API（taskAuth）

| Method | Path |
|--------|------|
| GET | `/api/accounts/users/me/account-deletion/precheck/` |
| GET | `/api/accounts/users/me/account-deletion/status/` |
| POST | `/api/accounts/users/me/account-deletion/request/` |
| POST | `/api/accounts/users/me/account-deletion/cancel/` |
| POST | `/api/internal/taskauth/account-deletion/execute-due/` |

## 领域事件

- `USER_ACCOUNT_DELETION_REQUESTED`
- `USER_ACCOUNT_DELETION_CANCELLED`
- `USER_ACCOUNT_DELETION_EXECUTION_STARTED`
- `USER_ACCOUNT_DELETION_COMPLETED`
- `USER_ACCOUNT_DELETION_BLOCKED`
