# v75 — Tenant Role Management (reusable roles + multi-role union)

```mermaid
flowchart LR
  Roles[PeopleRoles NEW] -->|CRUD + resource-groups| TA[taskAuth]
  Access[PeopleAccess MOD] -->|role_names replace-all / DELETE| TT[taskTenantService]
  TA --> AR[(auth_role)]
  TA --> RRG[(auth_role_resource_group)]
  TA --> Seed[(auth_resource_group people.roles*)]
  TT --> MR[(tenant_member_role multi)]
  TT --> GR[(tenant_group_role multi)]
  Invite[PeopleInvite] -->|pending_grants compat| TT
```

- **Version:** 75 current (shipped 2026-08-11)
- **Iteration:** tenant-role-management-v75
- **Based on:** v74
- **Decisions:** Q1=C, Q2=A, Q3=B, Q4=A; RequirePerm coarse codes not removed in this version
