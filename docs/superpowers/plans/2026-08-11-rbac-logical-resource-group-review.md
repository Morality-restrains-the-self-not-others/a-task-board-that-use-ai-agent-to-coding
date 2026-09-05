# Review — RBAC 逻辑资源组 v72 P1

- **Date:** 2026-08-11
- **Plan:** `2026-08-11-rbac-logical-resource-group-plan.md`

## 五轴

| 轴 | 结论 |
|----|------|
| Correctness | PDP 注入 region/page；tenant_admin 全量；绑 page 展开 region；B2 无粗码展开 |
| Readability | `rbac_resource_groups.go` 集中目录/绑定；FE `saveSubjectResourceAccess` |
| Architecture | 与 ADR-0003 / v63 PDP 头格式兼容（Cut 首个 `:`） |
| Security | PUT 禁改系统角色；公司归属校验；RequireRegion + 过渡 member:manage |
| Performance | 批量 SQL；header 仍受 7KB 截断保护 |

## 测试

- shareLib/authz：含 HasRegion/RequireRegion — pass
- taskAuth：expand + computePermSets merge — pass
- taskFE：nav / saveResource / PeopleAccess — pass
- 032 migrate：已应用到 task_auth

## Intent→Event

- PUT resource-groups → `publishRoleChanged`（缓存失效）
- 显式 Kafka `RoleResourceGroupsChanged` → OPT

## Critical

无阻断项。部署须：9999 精准编译重启 task-auth + taskFE（032 已落库）。
