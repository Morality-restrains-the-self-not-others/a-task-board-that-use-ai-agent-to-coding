# 角色权限分析：登录历史

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-login-history-design.md`

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| 写入 `auth_login_history` | 认证成功的账号自身 | User | write | 登录路径已通过凭据校验 | ✅ | 仅成功路径；模拟登录禁止写目标 |
| GET `/api/auth/login-history/` | 已登录用户 | User（仅自己） | read | `requireAuthenticatedUser` | ✅ | 查询强制 `WHERE user_id = 会话用户`；禁止 query user_id |
| GET `/api/system-admin/users/{id}/login-history/` | 平台超管 | System | read | `requireSuperuser` | ✅ | 与用户管理同闸；目标不存在 404 |
| 侧边栏/超管链接 | 已登录 / 超管 SPA | User / System | read | 路由守卫既有 | ✅ | 真实 href，无点击拦截 |

无新角色。IDOR 风险：用户 API 不得接受他人 user_id。IP/UA 对本人与超管可见，不对其他租户成员可见。
