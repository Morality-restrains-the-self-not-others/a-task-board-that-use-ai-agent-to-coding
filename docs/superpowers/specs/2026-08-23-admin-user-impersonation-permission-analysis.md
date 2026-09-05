# 角色权限分析：管理员模拟用户登录

- **Date:** 2026-08-23
- **Design:** `docs/superpowers/specs/2026-08-23-admin-user-impersonation-design.md`

## 权限模型

沿用既有平台权限码 `user:impersonate`（不新增角色）。super_admin 与 employee 静态映射已包含该码。

本能力是**平台级**操作，不走租户 page/region（元规则 45 不触发）。

## 端点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `/api/system-admin/users/{id}/impersonate/` | 持有 `user:impersonate` 的平台角色 | System / User | 写（签发模拟会话） | 整页 `requireSuperuser` 过粗 | ⚠️ 须特判 | `RequirePlatformPerm(PermUserImpersonate)`；禁止套用 `platform:manage` |
| POST `/api/auth/impersonation/stop/` | 当前模拟会话的 actor | System | 写（结束会话） | 无 | ⚠️ | 仅当 token 属于未结束模拟会话 |
| GET `/api/auth/impersonation/status/` | 已登录用户 | System | 读 | 无 | — | 已登录即可；非模拟返回 `impersonating:false` |
| 编辑页按钮 | 同上 | UI | 可见性 | 无 | ⚠️ | `hasPlatformPerm('user:impersonate')` |
| Navbar 退出 | 模拟中的浏览器会话 | UI | 写 | 无 | — | 仅 `impersonating` 时渲染 |

## 额外授权条件

| 条件 | 原因 |
|------|------|
| 目标 active 且未归档 | 停用账号不可登录 |
| actor ≠ target | 无意义且易混淆审计 |
| 禁止嵌套 | 防止 restore cookie 被覆盖丢失管理员会话 |
| 目标为超管 / `platform:manage` 时 actor 必须有 `platform:manage` | 防 employee 提权 |

## IDOR

路径中的 `{id}` 是任意用户。平台权限本身即跨租户；不以 tenant 过滤。仍须校验目标存在，避免对幽灵 id 签发会话。

## 审计

每次 start/stop 打 `actor_user_id` + `target_user_id` + `session_id`（非 token）+ trace_id。
