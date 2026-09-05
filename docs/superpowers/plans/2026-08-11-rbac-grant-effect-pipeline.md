# Value Stream / NFR / DDD / Plan — 资源组授予效果 v73（合并）

## Value Stream

管理员打开访问管理 → 勾选「可访问」或「可编辑执行」→ 保存 grants → PDP 注入效果码 → 目标用户刷新 → FE/BE 按 view/operate Enforce。

测试点：view-only 不能 PUT；operate 可 PUT；侧栏 page:view 仍可见。

## NFR（L3 auth）

- 效果码解析 O(1)；PDP 缓存失效沿用 membershipRev
- 存量 DEFAULT operate 行为连续
- 迁移幂等 guarded_add_column

## DDD

- 聚合：RoleResourceGrant(role_id, resource_group_id, effect)
- 领域事件：既有 role_resource_groups_changed（payload 可含 effect；本迭代不改 topic）
- 纯查询例外：GET resource-groups（无新事件）

## Build Plan

1. DDL 033 ✅
2. authz Emit/Has* ✅
3. PDP fetch effect ✅
4. PUT/GET grants ✅
5. FE 双档 + saveSubjectResourceAccess ✅
6. 测试 + 元规则 + ADR ✅
