# NFR — tenant-role-management-v75

| Category | Level | Notes |
|----------|-------|-------|
| Security | L3 | AuthZ on all mutate paths; IDOR checks company_id |
| Reliability | L2 | RoleExists via taskAuth; 502 on auth down |
| Performance | L2 | Replace-all in one TX; PDP already unions roles |
| Observability | L2 | INFO on role bind/unbind + TenantRoleChanged |
| Usability | L2 | Role-first UX; orphan cleanup retained |
| Compatibility | L2 | PUT accepts legacy `role_name`; coarse dual-write silent |

Blocked: none. Quality scenarios feed DDD (multi-role union, delete with refs).
