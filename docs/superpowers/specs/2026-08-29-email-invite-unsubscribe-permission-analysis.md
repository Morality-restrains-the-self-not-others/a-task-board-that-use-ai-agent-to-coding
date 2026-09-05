# 邮件邀请退订 — 权限分析

- 日期：2026-08-29
- 设计：`docs/superpowers/specs/2026-08-29-email-invite-unsubscribe-design.md`

## 端点鉴权矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET/POST `/api/public/email-unsubscribe/` | 持有 HMAC token 的收件人 | System / Email | write | 无登录 | ✅ 以 token 代替会话 | 禁止无 token；不回显完整邮箱 |
| GET `/api/internal/email-unsubscription/` | 内部服务 | System | read | `requireInternalSecret` | ✅ | 开发 secret 空 fail-open 与既有一致 |
| POST 超管 email-invitations | super admin | System | write | `isSuperAdminUser` | ✅ | 退订只改响应，不放宽鉴权 |
| POST 租户 members/invite | `member:manage` | Tenant | write | `requireCompanyAdmin` | ✅ | 同上 |
| SPA `/auth/unsubscribe/` | 匿名 | — | read | 公开路由 | ✅ | 仅展示 query `ok`，不接受裸 email |

无新角色、无新权限码。

## IDOR / 枚举

- Token 不含可预测自增 ID；MAC 防伪造。
- 内部查询按 email，仅内网 + internal secret。
- 公开错误统一「链接无效」，不区分「从未退订 / token 错」。

## 公开写路径

one-click 无 CSRF cookie；RFC 8058 POST body 固定。GET 会改变状态（邮件客户端常见），以 HMAC 约束。
