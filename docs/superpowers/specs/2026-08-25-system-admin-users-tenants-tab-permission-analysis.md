# 角色权限分析 — 超管用户页租户 Tab

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-system-admin-users-tenants-tab-design.md`

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/accounts/admin/tenants/` | 平台员工（super_admin / employee） | System | read | 与 tenant-options 相同：`X-Gateway-Auth-Verified` + `authz.IsPlatformStaff` | ✅ 充分 | 禁止对租户成员开放；响应不含密钥 |
| FE `/system-admin/users/?tab=tenants` | 平台员工 | System | read | 现有 system-admin 路由守卫 | ✅ 充分 | Tab 不单独鉴权，与页面同门 |
| `POST /api/internal/users/batch/details/`（出站） | taskTenantService 内部 | Identity | read | Internal secret | ✅ 充分 | fail-open；日志禁明文邮箱/手机 |
| `GET /api/internal/users/?q=`（出站搜索创建者） | taskTenantService 内部 | Identity | read | Internal secret | ✅ 充分 | 与 tenant-options 共用 |

## 角色建模

不新增角色。沿用平台员工。不引入租户管理员访问他租户目录。

## IDOR / 越权

- 列表是平台全局目录，无「按 caller 的 tenant_id 过滤」——这是有意的超管能力。
- 普通登录用户 / 租户管理员不得调用；403 且空 body 不泄漏公司名。
- 前端 Tab 对非超管不可达（进不了 `/system-admin/users/`）。

## 数据暴露

- 返回创建者邮箱/手机：仅平台员工可见，与 tenant-options 已批准的暴露面一致。
- 日志：`count` / `limit` / `offset` / `search_len`，禁止记录 search 原文（可能含手机号）。
