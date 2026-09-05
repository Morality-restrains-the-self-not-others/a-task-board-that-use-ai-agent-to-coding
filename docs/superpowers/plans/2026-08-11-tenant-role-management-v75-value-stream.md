# Value Stream — tenant-role-management-v75

## Streams touched

| Domain | Stream / Step | Change |
|--------|---------------|--------|
| 组织与成员 | 角色管理 CRUD | **NEW** — define reusable roles + region grants |
| 组织与成员 | 访问管理 | **MODIFY** — assign multi roles; stop implicit 访问· |
| 组织与成员 | 邀请预授 | **COMPAT** — pending_grants kept; prefer roles later |

## Fields

- `taskAuth.auth_resource_group.group_key` — + `people.roles*`
- `taskTenant.tenant_member_role.role_name` — multi rows per member
- `taskTenant.tenant_group_role.role_name` — multi rows per group

## Tests expected

- Go: replace-all / DELETE member & group roles
- FE: PeopleRoles CRUD; PeopleAccess multi-select; nav people.roles
- No new value-stream.yaml (repo root absent)
