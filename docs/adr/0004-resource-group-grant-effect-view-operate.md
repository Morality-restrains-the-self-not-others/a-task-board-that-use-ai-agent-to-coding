# ADR-0004: 租户资源组授予效果（view / operate）

- **Status:** accepted
- **Date:** 2026-08-11
- **Author:** cursor-grok
- **Deciders:** goal-mode 自动采纳（用户目标：资源分可访问 / 可编辑执行）
- **Relates to:** ADR-0003（逻辑资源组 page ⊃ ui_region；本 ADR 在绑定层增加效果，不推翻 0003）

---

## Context

ADR-0003 解决了「页面 ≠ 资源组、页内分区、前后端同源 Enforce」。绑定仍是二元：角色要么拥有某 region，要么没有。产品需要在**同一资源**上区分只读访问与编辑/执行，避免只能靠无限拆分 `*.save_actions` region 表达写能力。

## Decision

We will:

1. 在 `auth_role_resource_group` 增加 `effect ENUM('view','operate') NOT NULL DEFAULT 'operate'`；一角色一资源组一行，存最高效果。
2. **蕴含**：`operate` ⊃ `view`（PDP 与 FE 均保证）。
3. PDP 注入 `region:<key>:view` / `region:<key>:operate`；对 `operate` 额外注入遗留码 `region:<key>`（及 page 同理），兼容存量 `RequireRegion`。
4. `shareLib/authz` 提供 `HasRegionView` / `HasRegionOperate`（及 Require*）；新写路径优先 `RequireRegionOperate`。
5. 访问管理 API/UI 读写 `effect`；旧 `group_keys` 兼容为 `operate`。
6. 不强制合并既有细分 region；与 effect 可并存。

## Alternatives Considered

### Alternative 1: 仅继续拆 ui_region

- **Pros:** 无 schema 变更
- **Cons:** 种子爆炸；「只读同一区块」无法自然表达
- **Why rejected:** 不满足「资源上分权限」的产品语义

### Alternative 2: 叶子级 CRUD action

- **Pros:** 最细
- **Cons:** UX/运维过重；与 v72 region 粒度冲突
- **Why rejected:** 可后续在 `auth_resource_member.required_effect` 演进（非本决策）

### Alternative 3: 三档 view / edit / execute

- **Pros:** 更细
- **Cons:** 访问管理矩阵复杂度高；首期收益不足
- **Why rejected:** 首期锁定两档；`operate` 覆盖编辑+执行

## Consequences

### Positive

- 任意 region 可只读授予；写按钮/写 API 可独立要求 operate
- 存量全权绑定默认 operate，行为连续

### Negative / Trade-offs

- PDP 码空间扩大（`:view` / `:operate` + 遗留码）
- FE 矩阵从单勾选变为双档
- 存量 `RequireRegion` 在仅 view 授予时为 false（符合安全预期；读路径需逐步改 View）

### Mitigations

- 遗留 `region:<key>` 仅由 operate 注入
- 文档与元规则 45 明确 Enforce 选用表
- 关键读 API 迁移 `RequireRegionView` 列入 OPT / 本迭代覆盖访问管理读路径

## References

- 设计: `docs/superpowers/specs/2026-08-11-rbac-resource-grant-effect-v73-design.md`
- 架构: `docs/architecture/v73-application-integration-20260811-1500-cursor.*`
- 元规则: `.ai/01_project_constraints/45_tenant_logical_rbac_resource_groups.md`
