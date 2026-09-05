# Plan — tenant-role-management-v75

## Tasks

- [x] T1: dataMigrate/taskAuth/034_people_roles_page.sql — page + regions + members + tenant_admin bind
- [x] T2: taskTenant replace-all `role_names` + DELETE single role (member & group) + unit tests
- [x] T3: taskAuth delete role — cascade resource_groups/permissions
- [x] T4: FE nav `people.roles` + route + PeopleRoles.vue (CRUD + matrix)
- [x] T5: FE PeopleAccess — multi-select roles, save via role_names, no 访问· create
- [x] T6: domain helper assignSubjectRoles + tests
- [x] T7: register precise restart taskFE taskTenant taskAuth
- [x] T8: Review + OPT notes

## Events in plan

- RoleChanged / TenantRoleChanged already exist — wired on DELETE/replace-all paths
