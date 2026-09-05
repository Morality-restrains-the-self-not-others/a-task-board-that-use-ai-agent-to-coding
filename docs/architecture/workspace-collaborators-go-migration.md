# workspace-collaborators / enrich-permissions：Django → 纯 Go

- 日期：2026-07-20
- 状态：**已落地并收尾**（逻辑在 Go；Django 路由已物理删除）
- 落点：扩展现有 `taskProjectService`（不新建服务）

## 现状

| 层 | 职责 | Owner |
|----|------|--------|
| 公网 `GET …/workspace-collaborators/` | 鉴权、读 access、拼协作人 DTO | **taskProjectService** |
| 公网 `GET …/workspace-permissions/` | 鉴权、读 access、拼 `user_info`/`group_info` | **taskProjectService** |
| 原 Django internal 同名路径 | 已删除（无路由） | — |
| 成员/分组表 | SSOT | **taskTenantService**（`:8020`） |
| `workspace_accesses` / workspaces | SSOT | **taskProjectService** |
| 用户 email | SSOT | **taskAuth**（`GET /api/accounts/users/{id}/`） |
| 公司创建者（`is_tenant`） | SSOT | **taskTenantService** `accounts_company.creator_id` |

实现要点：

- `cfg.TaskTenantURL` / `cfg.TaskAuthURL`
- `tenant_client.go`、`auth_client.go`、`workspace_access_enrich.go`
- `user_info.email` ← taskAuth；`user_info.is_tenant` ← tenant `companies/creator`（且存在公司成员行）
- `set`/`remove` permission：成员解析走 `tenantGetMemberByID`
- 详见 [company-creator-tenant-migration.md](./company-creator-tenant-migration.md)

## 验收

- `GET :8016/…/workspace-collaborators|workspace-permissions/` → **200**，不访问已删 Django 路径
- Django `POST …/workspace-collaborators|enrich-workspace-permissions/` → **404**
- Go：`go test -run 'Workspace|Enrich|Collaborat'`
- 现场：`user_info.email` 非空（有登录邮箱时）、创建者 `is_tenant=true`

## 与元规则对齐

- [20_go_service_first_apis](../../.ai/01_project_constraints/20_go_service_first_apis.md)
- [19_single_service_data_ownership](../../.ai/01_project_constraints/19_single_service_data_ownership.md)
