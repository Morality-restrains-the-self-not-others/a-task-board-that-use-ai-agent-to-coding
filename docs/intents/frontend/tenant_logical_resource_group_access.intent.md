# 意图：租户控制台访问管理（逻辑资源组 Page ⊃ Region）

## 用户故事

作为公司管理员，我希望在「人员管理 → 访问管理」中为成员或小组勾选**页面组下的 UI 组件区域**（资源组），以便前后端同源控制页面可见性与 API 访问；`tenant_admin` 拥有全部系统资源组。

## 业务规则

1. 入口保留：人员管理子菜单「访问管理」；配置能力依赖 `member:manage`（过渡）或等价管理 region。
2. **页面组 ≠ 资源组**：页面组是载体；**UI 组件区域（ui_region）** 才是默认可授予的资源组（ADR-0003 / A1）。
3. UI 组件与 API endpoint 仅挂在 **ui_region** 上；角色可绑 region，或绑 page 由 PDP 展开为子 region。
4. PDP 注入 `region:<key>` / `page:<key>`；本授权链路**不**展开旧粗码（B2）。
5. 前端：侧栏用 `page:*`；页内控件用 `hasRegion`；直链无 page → 无权限空态。
6. 后端：关键 API 使用 `RequireRegion`；仅前端隐藏不算验收通过。
7. **不**复用业务表 `tenant_resource_group_assignment`。
8. 权威设计：`docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md`。
9. 旧意图 `tenant_menu_access_management.*`（菜单↔粗码）已 superseded。

## 服务端事件

- 角色↔资源组变更：可复用/扩展既有 RoleChanged；实现阶段在 value-stream 定稿
- 成员/组角色绑定：既有 TenantRoleChanged（taskTenant）

## 验收

见同名 `.test-intent.md`。
