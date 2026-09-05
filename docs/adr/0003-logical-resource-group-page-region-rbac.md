# ADR-0003: 逻辑资源组（页面组 ⊃ UI 组件区域）前后端统一 RBAC

- **Status:** accepted
- **Date:** 2026-08-11
- **Author:** claude
- **Deciders:** 用户确认 Q1=A1、Q2=B2、Approve（2026-08-11）

---

## Context

租户控制台需要按「谁能访问哪些能力」做授权。先前将「页面」直接当作资源组、仅做前端菜单隐藏，无法表达页内分区，且 API 可被绕过。产品澄清：**资源组 ≠ 页面组**；页面组是载体；UI 组件区域才是资源组；UI/API 挂在区域上。

既存 `tenant_resource_group_assignment` 表示业务实体→人员小组的数据范围，与本决策正交。

## Decision

We will:

1. 引入 `auth_resource_group`（`kind=page|ui_region`），页面组为载体，**UI 组件区域为默认可授予资源组**。
2. `auth_resource_member`（`ui|api`）**仅**挂在 `ui_region` 下。
3. `auth_role_resource_group` 绑定角色↔page 或 region；绑 page 时 PDP **展开**为全部子 region（Q1=A1）。
4. PDP 注入 `region:<key>` / `page:<key>`；**不再**以旧粗码（`project:view` 等）作为本授权链路的展开目标（Q2=B2）。过渡期存量 `RequirePerm(粗码)` 可并存，但不作为 Page/Region 授权的完成标准。
5. 前后端均用 `RequireRegion` / `hasRegion`（及 page 侧栏）同源 Enforce；仅前端隐藏不算完成。
6. 不复用、不混名 `tenant_resource_group_assignment`。

## Alternatives Considered

### Alternative 1: 页面=资源组，叶子直接挂 page

- **Pros:** 模型简单
- **Cons:** 无法表达页内分区（只读区 vs 危险操作区）
- **Why rejected:** 产品明确否定

### Alternative 2: 继续仅粗码 + FE 菜单映射

- **Pros:** 改动小
- **Cons:** 前后端漂移；可绕过
- **Why rejected:** 无法防绕过

### Alternative 3: 新平行 ACL 绕过 PDP

- **Pros:** —
- **Cons:** 与 v63 双轨冲突
- **Why rejected:** 违反集中 PDP 架构

## Consequences

### Positive

- 页内细粒度与整页快捷授权兼得
- FE/BE 同一 registry，可 CI 门禁新 API 必须归属 region

### Negative / Trade-offs

- 需维护 page/region/member 种子与 handler 挂载
- 旧粗码路径与新 region 路径短期并存，认知负担

### Mitigations

- P1 先覆盖控制台人员/设置/计费；P2 CI 收敛；文档与 ADR 明确双轨语义

## References

- 设计: `docs/superpowers/specs/2026-08-11-rbac-page-resource-group-v72-design.md`
- 架构: `docs/architecture/v72-application-integration-20260811-1136-claude.*`
- v63: `docs/superpowers/specs/2026-08-05-rbac-merged-v63-design.md`
- **项目元规则（后续租户改动强制清单）**: `.ai/01_project_constraints/45_tenant_logical_rbac_resource_groups.md`；Cursor `.cursor/rules/tenant-logical-rbac-resource-groups.mdc`
