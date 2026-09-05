# 角色权限分析：可插拔多区域 gitService

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-pluggable-multi-region-gitservice-design.md`
- **结论**: 无新角色；沿用 system_admin / tenant_admin / 租户成员；强化 region 必填与 token 脱敏

## 端点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `/api/system-admin/gitlab-regions/` | system_admin | System | read | SystemAdmin 门禁 | ✅ | 列表脱敏 token |
| POST/PATCH/DELETE `/api/system-admin/gitlab-regions/` | system_admin | System | manage | SystemAdmin | ✅ | 写 token 后响应脱敏 |
| GET `/api/tenant/{tenant_id}/billing/gitlab-regions/` | tenant 成员 | Tenant | read | 租户归属 | ✅ | 仅 `is_active=1` |
| GET/POST 租户 GitLab 配额（带 region） | tenant_admin / billing | Tenant | read/write | 既有 billing 门禁 | ⚠️ | **region 必填**；无默认 |
| 内部开通 `ensureTenantGitlabGroupForRegion` | 服务内部 | System→Region | write | Admin PAT per region | ⚠️ | 禁止全局 fallback；缺 token → pending |
| OIDC 多 GitLab callback | 用户浏览器 | Identity | auth | taskAuth bootstrap | ✅ | 独立 client，不跨实例共享 secret |

## 数据访问

| 表/字段 | 谁可读 | 谁可写 | 备注 |
|---------|--------|--------|------|
| `billing_gitlab_region.admin_private_token` | system_admin（脱敏） | system_admin | 永不写入前端日志 |
| `billing_tenant_gitlab_resource` | 本租户 | 本租户购买流 + system 开通 | PK `(tenant_id, region)` |
| GitLab Admin API | taskBill 服务账号 | 同上 | 每 region 独立 PAT |

## IDOR / 越权风险

- 租户 A 不得用 region 参数操作租户 B 配额（既有 tenant_id 路径校验保留）。
- 停用区域后禁止新购；已购只读（设计 D5/D6）。
- 无新角色建模需求。
