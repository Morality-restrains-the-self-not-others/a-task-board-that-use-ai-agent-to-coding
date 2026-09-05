# 租户 RBAC 资源授予效果层（view / operate）v73 设计

- **Status:** accepted（goal-mode 自动采纳）
- **Date:** 2026-08-11
- **Author:** cursor-grok
- **Base:** ADR-0003 / v72 逻辑资源组
- **ADR:** ADR-0004

## 1. 问题

v72 将「能否使用某 ui_region」建模为**二元绑定**（有/无）。产品要求同一资源组区分：

| 效果 | 中文 | 典型能力 |
|------|------|----------|
| `view` | 可访问 | 看见区块、读列表/详情（GET） |
| `operate` | 可编辑执行 | 创建/更新/删除、触发危险操作（写 API + 写按钮） |

当前用「拆更多 region」（如 `save_actions`）表达写能力，认知成本高，且无法在任意 region 上统一表达只读 vs 可写。

## 2. 决策（锁定）

**We will** 在 `auth_role_resource_group` 增加 **`effect ENUM('view','operate') NOT NULL DEFAULT 'operate'`**：

1. **一角色 × 一资源组 = 一行**，存最高效果（`operate` ⊃ `view`）。
2. PDP 注入：
   - `view` → `region:<key>:view`（page 同理 `page:<key>:view`）
   - `operate` → `region:<key>:view` + `region:<key>:operate` + **遗留** `region:<key>` / `page:<key>`（兼容存量 `RequireRegion`）
3. Enforce：
   - `HasRegionView` / `RequireRegionView` — 读路径
   - `HasRegionOperate` / `RequireRegionOperate` — 写/执行路径
   - 存量 `HasRegion` / `RequireRegion` — 仍匹配遗留 `region:<key>`（即 operate 授予）；**新写 API 应显式 `RequireRegionOperate`**
4. 访问管理 UI：每个 region 两档勾选（可访问 / 可编辑执行）；勾选 operate 自动蕴含 view。
5. API：`PUT` 接受 `grants:[{group_key,effect}]`；旧 `group_keys` 视为 `effect=operate`。
6. **不**在本迭代强制合并既有 `*.save_actions` region（保留）；效果层与细分 region **可并存**。合并列为 OPT。

## 3. 方案对比

| 方案 | 说明 | 结论 |
|------|------|------|
| A. 绑定 effect（本决策） | 同一 region 上 view/operate | ✅ 选中：与用户「资源分权限」语义一致 |
| B. 继续只拆 region | `*.main` / `*.save` | ❌ 无法普适；爆炸式种子 |
| C. 叶子 action CRUD | 每 api 成员绑 create/read/… | ❌ 过重；P2 可演进 `required_effect` |

## 4. 数据模型

```sql
ALTER TABLE auth_role_resource_group
  ADD COLUMN effect ENUM('view','operate') NOT NULL DEFAULT 'operate'
  AFTER resource_group_id;
-- UNIQUE (role_id, resource_group_id) 保持不变
```

存量行默认 `operate` → 行为与 v72 全权授予一致。

## 5. API 契约

### GET `/api/auth/roles/role_id/{rid}/resource-groups/`

```json
{
  "role_id": "...",
  "resource_groups": [
    {"id":"...","group_key":"settings.cloud.main","kind":"ui_region","effect":"view"}
  ]
}
```

### PUT 同路径

```json
{
  "grants": [
    {"group_key":"settings.cloud.main","effect":"view"},
    {"group_key":"settings.gitlab.main","effect":"operate"}
  ]
}
```

兼容：`group_keys: ["..."]` / `group_ids` → 全部 `operate`。

## 6. FE

- `selectedGrants: Map|Array<{group_key, effect}>`
- 矩阵：可访问 checkbox + 可编辑执行 checkbox；operate 勾选 ⇒ view 勾选且禁用取消 view（或自动保持）
- `saveSubjectResourceAccess` 传 `grants` 而非仅 keys
- 页内：`hasRegionView` / `hasRegionOperate`（composables）

## 7. 验收标准

- [ ] DDL `033_*` 经 dataMigrate；业务启动不跑迁移
- [ ] 单元测试：PDP 注入、HasRegionView/Operate、PUT grants 往返
- [ ] FE 单测：grant 归一化（operate⊃view）、旧 selectedGroupKeys 升级
- [ ] 元规则 45 / companion / ADR-0004 已更新
- [ ] 架构 v73 四件套已落盘

## 8. 非目标（本迭代）

- 三档 `execute` 独立于 `edit`
- 自动按 HTTP method 推断 required_effect（OPT）
- 合并 `people.access.save_actions` 进主 region+effect（OPT）

## 9. 架构制品

- `docs/architecture/v73-application-integration-20260811-1500-cursor.*`
