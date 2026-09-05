# 权限分析：自建 GitLab 同专有网络提示

- **日期**: 2026-08-26
- **设计**: `docs/superpowers/specs/2026-08-26-gitlab-selfhosted-same-vpc-hint-design.md`

页面已要求 `company:manage`（侧栏 `settings.gitlab`）。本增量不新增角色或 endpoint。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/cloud/server-config-default/tenant_id/{tid}/` | 租户成员 | Tenant | read | `ensureTenantMember` | ✅ 充分 | 列表仅本 tenant `company_id` |
| POST `/api/cloud/create-vpc/` | 租户成员（有云授权） | Tenant | write | `loadCloudAuthByID` 属本租户授权 | ✅ 充分 | 不改 handler |
| POST create-vswitch | 同上 | Tenant | write | 同上 | ✅ 充分 | 不改 handler |
| 前端路由 `/tenant/:tenant/settings/gitlab-connection/` | `company:manage` | Tenant | read UI | 既有 nav anyOfPerms | ✅ 充分 | 无新路由 |
| 链到 `/settings/task-panel/` | `settings.task_panel` | Tenant | navigate | 既有侧栏 | ✅ 充分 | 真实 `<a href>`，无静默 replace |

无 IDOR：tenant_id 来自路由且 API 以 auth tenant 为准。无新密钥字段。创建弹窗不在日志中打印 secret。

**结论**：不引入新权限模型；不新增 Python/Go 鉴权代码。
