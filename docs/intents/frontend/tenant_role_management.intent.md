# 意图：租户角色管理（可复用角色 + 多角色并集）

## 用户故事

作为公司管理员，我希望在「人员 → 角色管理」中创建可复用角色并配置页面/区域权限，再在「访问管理」中为成员或小组分配**一个或多个**角色，以便多人共享同一权限模板，且有效权限为多角色并集。

## 业务规则

1. 侧栏独立入口 `people.roles`（角色管理）；访问管理 `people.access` 负责主体↔角色分配，不再隐式创建 `访问·…` 角色。
2. 角色可配内容仅为 page/region 的 view|operate（ADR-0003/0004）；不对管理员暴露粗码勾选。
3. 系统角色只读；自定义角色可创建/改名/删（有主体引用时拒绝删除）。
4. 成员/组支持多角色；权限 = 直接角色 ∪ 组继承角色（PDP 并集）。
5. 前后端 Enforce 新页用 region；过渡期可并存 `member:manage`。
6. 租户粗码 `RequirePerm` **本迭代不移除**（见设计 §8）；新路径以 region 为真源，粗码可静默双写兼容。
7. 权威设计：`docs/superpowers/specs/2026-08-11-tenant-role-management-v75-design.md`。

## 服务端事件

| 业务意图 | 事件 | 发布点 |
|---------|------|--------|
| 角色 CRUD / 改 resource-groups | RoleChanged | taskAuth |
| 成员/组多角色变更 | TenantRoleChanged 族 | taskTenant |

## 验收

见同名 `.test-intent.md`。
