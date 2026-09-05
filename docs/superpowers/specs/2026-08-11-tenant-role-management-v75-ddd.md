# DDD — tenant-role-management-v75

## Aggregates

| Aggregate | Root | BC |
|-----------|------|-----|
| ReusableRole | auth_role | taskAuth |
| RoleGrantSet | auth_role_resource_group | taskAuth |
| MemberRoleSet | tenant_member_role (by member_id) | taskTenant |
| GroupRoleSet | tenant_group_role (by group_id) | taskTenant |

## Domain events

| Intent | Event | Publish | Consumer |
|--------|-------|---------|----------|
| Role CRUD / grants changed | RoleChanged | taskAuth | PDP cache invalidate |
| Member/group roles replaced/removed | TenantRoleChanged | taskTenant | membership rev++ |

## Invariants

1. Custom role delete refused if any member/group binding exists
2. Replace-all is transactional (delete then insert)
3. Effective perms = union of direct + group-inherited roles (PDP)
4. System roles immutable grants
