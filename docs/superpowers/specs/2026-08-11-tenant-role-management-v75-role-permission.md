# Role-Permission Analysis — tenant-role-management-v75

- **Date:** 2026-08-11
- **Design:** `docs/superpowers/specs/2026-08-11-tenant-role-management-v75-design.md`
- **goal-mode:** auto-adopted

## Actors

| 角色 | 能力 |
|------|------|
| tenant_admin | 全部 page/region；可管理角色与分配 |
| 持有 `people.roles.*` operate / `member:manage` | 角色 CRUD + 矩阵保存 |
| 持有 `people.access.save_actions` operate / `member:manage` | 主体多角色分配 |
| 普通 member | 只读自身权限（无角色管理入口） |

## New / Changed Endpoints

| Method | Path | Auth |
|--------|------|------|
| PUT | `/api/tenant/member-role/.../member_id/{mid}/` | member:manage（过渡）/ region people.access.save_actions；body `role_names[]` replace-all；兼容 `role_name` |
| DELETE | `/api/tenant/member-role/.../member_id/{mid}/role_name/{name}/` | 同上 |
| PUT/DELETE | group-role 对称 | group:manage 过渡 |
| 既有 | `/api/auth/roles*` + resource-groups | member:manage / people.roles.save_actions |

## Data access

- taskAuth owns `auth_role` / `auth_role_resource_group` / `auth_resource_group`
- taskTenant owns `tenant_member_role` / `tenant_group_role`
- No cross-DB writes

## Risks

- Multi-role replace-all mis-clear → confirm UI + audit log
- Coarse RequirePerm retained (design §8)
