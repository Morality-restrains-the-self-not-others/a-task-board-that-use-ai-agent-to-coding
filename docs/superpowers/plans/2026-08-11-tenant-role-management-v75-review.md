# Review notes — tenant-role-management-v75 (2026-08-11)

## Verdict
**Approve to ship** (minor follow-ups tracked as OPT).

## Against plan / design
| Decision | Implemented |
|----------|-------------|
| Q1=C people.roles page | `PeopleRoles.vue` + nav + router + seed 034 |
| Q2=A role-first, no 访问· | Access uses `assignSubjectRoles` / `role_names[]` |
| Q3=B multi-role union | taskTenant replace-all + FE multi-select |
| Q4=A page/region only in UI | Role grants via resource-groups; coarse codes not exposed |
| RequirePerm stay | Not removed; OPT-074 tracks retirement |

## Tests
- taskTenant: `TestParseRoleNamesBody` / member-role (commit `7207109`)
- taskAuth: `TestHandleDeleteRoleCascadesResourceGroupsAndPermissions` (commit `7be2102`)
- taskFE: vitest PeoplePageAccess (12) + assignSubjectRoles + tenantConsoleNav — pass

## Residual risk
1. **Deploy**: need 9999 migrate for `034_people_roles_page.sql` + precise restart (taskFE / task-auth / task-tenant-service).
2. **Invite UI** still pending_grants oriented — follow-up OPT-081.
3. **Matrix duplication** — OPT-075 / OPT-063.
4. Concurrent **v76** remains target; do not confuse with v75 current.

## CRG
`CRG unavailable` / skipped — fail-open per review skill.
