# Step 6 — DDD：访问管理

## 限界上下文

- **Identity & Access（既有）**：Role、Permission、MemberRole、GroupRole、PDP
- **Tenant Console UI（本迭代）**：MenuAccessPolicy（前端值对象：菜单 key ↔ perm codes）

## 聚合 / 值对象

- `MenuAccessCatalog`：静态目录
- `SubjectAccessDraft`：{ subjectType, subjectId, selectedMenuKeys } → `Set<PermCode>`

## 领域事件（复用）

- `RoleChanged` / `TenantRoleChanged` — 无新事件；书面例外：本意图写路径已覆盖。

## 端口

- 无新端口；适配器为 HTTP 调用既有 taskAuth / taskTenant API。
