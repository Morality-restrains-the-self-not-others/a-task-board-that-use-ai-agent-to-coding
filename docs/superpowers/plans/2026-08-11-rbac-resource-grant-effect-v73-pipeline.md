# Step 2–7 / 9 压缩产物 — rbac-resource-grant-effect-v73

goal-mode 自动流水线；细节见设计与 ADR-0004。

## Role-Permission
- 角色仍绑 page/region；新增 effect 维度（view|operate）
- tenant_admin = 全量 operate
- 写 API：RequireRegionOperate；读：RequireRegionView

## Value Stream
主体选择 → 勾选可访问/可编辑执行 → PUT grants → PDP 注入 → FE/BE Enforce

## NFR
- L3 安全（授权）
- 向后兼容：旧 group_keys → operate；存量列 DEFAULT operate

## DDD
- 聚合：RoleResourceGrant(role_id, resource_group_id, effect)
- 事件：既有 role_resource_groups_changed（载荷不变；effect 在 DB）

## Plan（已执行）
1. DDL 033 effect
2. authz Emit + View/Operate API + 测试
3. PDP fetch 读 effect
4. PUT/GET grants
5. FE 双档 + save grants
6. 元规则 / ADR / 架构

## Review（自检）
- Correctness: view-only 无 bare region；operate 有 view+operate+bare ✅
- Security: 写路径 HasRegionOperate ✅
- Line limit: PeopleAccess/rbac 文件 ≤500 ✅
