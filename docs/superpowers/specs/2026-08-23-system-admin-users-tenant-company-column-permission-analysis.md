# 角色权限分析 — 用户列表所属租户公司列

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-system-admin-users-tenant-company-column-design.md`

## 权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/users/` | 平台超管 (`requireSuperuser`) | System | 读 | `handleSystemAdminUsers` 入口 `requireSuperuser` | ✅ 充分 | 不新开端点 |
| POST `/api/internal/tenant/members/batch-get/` | 服务间（taskAuth） | System | 读 | `X-Internal-Secret` / `checkInternalSecret` | ✅ 充分 | 沿用既有内部密钥 |
| FE `/system-admin/users/` 新列 | 平台超管（路由守卫） | System | 读 | 既有 system-admin 路由 | ✅ 充分 | 无新写按钮 |

## 结论

- 无新角色、无新公开写接口、无 IDOR 面扩大：租户名只出现在超管已可见的全局用户列表。
- 内部密钥不入日志；响应可含公司名（运营所需，非银行卡/身份证类 PII）。
- 不新增前端副作用按钮（Anti-Replay-OK: 只读列）。
