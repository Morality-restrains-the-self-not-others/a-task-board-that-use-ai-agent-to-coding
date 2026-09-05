# 角色权限分析：模拟登录审计标识与收信箱

- **日期**: 2026-08-23
- **对应设计**: `docs/superpowers/specs/2026-08-23-impersonation-audit-inbox.md`

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST /api/system-admin/users/{id}/impersonate/ | 持有 `user:impersonate` 的平台员工/超管 | System | write | RequirePlatformPerm + ValidateStart | ✅ 充分 | 增加必填 reason |
| GET /api/auth/inbox/ | 登录用户 | User（本人） | read | requireAuthenticatedUser + WHERE recipient=self | ✅ 充分 | 禁止按他人 userId 查询 |
| PATCH /api/auth/inbox/{id}/read/ | 登录用户 | User（本人） | write | 信件 recipient 必须等于登录用户 | ✅ 充分 | 防 IDOR |
| 写 auth_user_inbox_message | 仅 impersonate 成功路径内部 | User | write | 无公开写信 API | ✅ 充分 | 禁止任意用户给他人写信 |
| 读模拟会话 reason | 被模拟用户经收信箱；管理员经会话表运维 | System/User | read | 收信箱仅本人 | ✅ 充分 | 日志不打理由全文 |

不新增租户 region 或平台权限码。收信箱是个人能力，不是租户控制台功能。
