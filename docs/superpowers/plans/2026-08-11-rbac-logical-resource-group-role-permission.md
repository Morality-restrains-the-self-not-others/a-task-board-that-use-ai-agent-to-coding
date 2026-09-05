# Role-Permission — 逻辑资源组 v72

- **Date:** 2026-08-11
- **Design:** `docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md`
- **ADR:** ADR-0003（A1/B2）

## 端点权限矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 检查 | 建议 |
|--------|------|----------|------|------|------|
| GET `/api/auth/resource-groups/` | 租户成员（管理访问） | Tenant | read | `RequireRegion(people.access.subject_list)` 或过渡 `member:manage` | P1 双轨：Region **或** member:manage |
| GET/PUT `/api/auth/roles/role_id/{rid}/resource-groups/` | 管理员 | Tenant | write | `RequireRegion(people.access.save_actions)` 或 `member:manage` | 系统角色禁改绑定（tenant_admin 已由 PDP 全量） |
| 既有角色 CRUD | 同上 | Tenant | write | 保持 `member:manage`；保存访问改走 resource-groups | — |
| 业务关键 API（P1 挂载） | 持 region 者 | Tenant | * | `RequireRegion(<key>)` | people.access 相关写路径优先 |

## 角色模型

- 不新增内置角色名；扩展 `auth_role_resource_group`。
- `tenant_admin`：PDP 注入全部 `is_system=1` 的 page/region。
- 自定义角色：仅绑 `ui_region`（默认）或 `page`（展开）。

## IDOR / 缺失风险

- PUT resource-groups 须校验 role.company_id == 请求公司。
- member 不可挂到 page 行（应用层拒绝）。
- 不读写 `tenant_resource_group_assignment`。
