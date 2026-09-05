# 角色权限分析 — 管理端赠送 GitLab 须选区域

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-admin-grant-gitlab-region-design.md`
- **结论**: 无新角色、无新 endpoint；沿用系统管理员赠送 + 区域列表读权限。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST `/api/tenant/{tid}/billing/accounts/admin_grant_points/` | 系统管理员 | System → 指定 Tenant 配额 | write | 网关 system-admin + Billing 代理 | ✅ 充分 | 仅增加 `region` 校验；禁止租户自赠 |
| POST `/api/internal/taskbill/admin-grant-resources/` | 内部服务 | Tenant | write | `requireInternalSecret` | ✅ 充分 | 新用户礼包仍为 task_post，不传 region |
| GET `/api/system-admin/gitlab-regions/` | 系统管理员 | System | read | 网关 system-admin | ✅ 充分 | 赠送页复用，不新开接口 |
| `billing_tenant_gitlab_resource` 按 region 写入 | 同上 | Tenant + Region | write | 仅经上述 API | ✅ | 校验 slug `is_active`，防任意字符串写入 |

## 建模

不引入新角色。IDOR：路径 `tenant_id` 由系统管理员显式选择，与现页一致。`region` 必须是已登记启用区，避免把配额写到不存在的实例。
