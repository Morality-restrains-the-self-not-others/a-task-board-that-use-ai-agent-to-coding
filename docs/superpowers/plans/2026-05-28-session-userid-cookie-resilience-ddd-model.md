# DDD 轻量模型：Session userId Cookie 业务韧性

**上下文：** 用户与认证（前端）

| 类型 | 名称 | 说明 |
|------|------|------|
| 值对象 | SessionUserId | numeric string user id |
| 领域服务 | SessionUserIdResolver | resolveAuthenticatedUserId |
| 领域服务 | UserIdCookieSync | syncUserIdCookieFromProfile |
| 领域服务 | AuthSessionGuardService | profile 校验 + sync/clear |

**不变量：**

1. profile 非 200 → 不得 sync cookie
2. sync 仅当 extracted user_id 非空
3. resolver：cookie 优先，profile 回退，失败返回 ''

**文件映射：**

- `utils/sessionUserIdUtils.js` — Resolver + Sync
- `domain/auth/services/auth_session_guard_service.js` — Guard
