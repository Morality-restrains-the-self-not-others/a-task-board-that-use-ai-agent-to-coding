# 角色权限分析 — 用户列表「是否获得分账资格」列

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-system-admin-users-profit-sharing-qualification-column-design.md`

## 权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/users/` 新增字段 | 平台超管 (`requireSuperuser`) | System | 读 | `handleSystemAdminUsers` 入口 `requireSuperuser` | ✅ 充分 | 不新开公开端点 |
| POST `/api/internal/referral/qualification/active/batch/` | 服务间（taskAuth） | System | 读 | `X-TaskReferral-Internal-Secret` / `requireInternalSecret` | ✅ 充分 | 与单用户 active 接口同密钥 |
| FE `/system-admin/users/` 新列 | 平台超管（路由守卫） | System | 读 | 既有 system-admin 路由 | ✅ 充分 | 无新写按钮 |

## 结论

- 无新角色、无新公开写接口、无 IDOR 面扩大：资格布尔值只出现在超管已可见的全局用户列表。
- 内部密钥不入日志；响应仅为 true/false/null，不含 openid/收款账号。
- 不新增前端副作用按钮（Anti-Replay-OK: 只读列）。
