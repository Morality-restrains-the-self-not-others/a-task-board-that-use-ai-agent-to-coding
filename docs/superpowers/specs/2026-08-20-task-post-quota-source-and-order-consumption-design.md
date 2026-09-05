# 任务帖配额来源拆分与订单消耗归属 — 设计文档

- **Date:** 2026-08-20
- **Status:** accepted（goal-mode 自动采用）
- **Author:** cursor
- **入口:** `/tenant/{tenantId}/billing/` 配额卡；`/tenant/{tenantId}/billing/orders/` 展开行 / 订单详情
- **TraceId:** 无（页面元素调整需求，非报错排障）

## 问题

1. 账单页「任务帖配额」只展示 `task_post_quota` 合计（现网 83 帖），无法区分**赠送**与**购买**。
2. 订单列表展开/详情只展示行项（类型、数量、单价、小计），消耗任务帖时**不能归属到具体订单**。

## 现状（代码基线）

- 赠送写入 `billing_resource_grant`（`remaining` + FEFO），并同步增加 `billing_account.task_post_quota`。
- 购买只增加 `task_post_quota`，**不建批次**。
- 消耗：优先扣 grant（当前全是赠送），再扣合计计数器；流水 `points_source_type=quota_consumption` **不记录 grant/order**。
- `related_order_id` 已存在于 `billing_transaction`（赠送汇总流水），配额消耗未用。

## 目标（完成标准）

1. 账单配额卡同时展示 **赠送剩余**、**购买剩余**、**合计**（合计字段保持兼容）。
2. 已支付订单详情/列表展开可查看该订单任务帖：**已购、已消耗、剩余**；有消耗流水时列出时间、动作（创建/续存）、`task_id`、来源批次。
3. 新消耗必须写入批次与订单归属；赠送优先于购买（既有 FEFO 语义保留）。
4. 无新 Python 接口；扩展既有 GET；tenant 隔离不变。

## 选定方案

| 方案 | 结论 |
|------|------|
| A. 仅前端用交易流水推算 | 拒：购买无批次，无法拆剩余，也无法归属订单 |
| B. 新建独立 lot 表 | 拒：与 `billing_resource_grant` 重复 |
| **C. 扩展 grant 为统一批次 + 消耗记 related_order_id（采用）** | 赠送/购买同一 FEFO 表；`source_kind` 保证赠送优先 |

### 数据

`billing_resource_grant` 新增：

- `source_kind` VARCHAR(16) NOT NULL DEFAULT `'gift'` — `gift` | `purchase`
- `order_id` BIGINT NULL — 购买/赠送订单

`billing_transaction` 新增：

- `source_grant_id` BIGINT NULL — 本次消耗的批次

存量赠送行 `source_kind='gift'`。购买支付时插入 `source_kind='purchase'`、`expires_at IS NULL`（批次不过期；帖子 12 个月存续仍在任务侧）。

### 消耗顺序

1. `FOR UPDATE` 取 1 行：`resource_type=task_post` 且 `remaining>0` 且未过期。
2. `ORDER BY source_kind='gift' 优先`，再 FEFO（有 `expires_at` 先于 NULL，早到期先扣）。
3. `remaining-1`，账户 `task_post_quota-1`。
4. 流水写入 `source_grant_id`、`related_order_id`（批次上的 `order_id`）。
5. Outbox `BILLING_TRANSACTION_CREATED` payload 增 `source_kind`、`order_id`、`source_grant_id`。

### API（向后兼容，仅增可选字段）

`GET /api/tenant/{tenant_id}/billing/quotas/`

```json
{
  "task_post_quota": 83,
  "task_post_quota_gifted": 50,
  "task_post_quota_purchased": 33
}
```

- `task_post_quota_gifted` = 未过期 gift 批次 `SUM(remaining)`
- `task_post_quota_purchased` = `max(0, task_post_quota - gifted)`（含尚未建批次的历史购买剩余）

`GET /api/tenant/{tenant_id}/billing/orders/{order_id}/` 增：

```json
{
  "resource_consumption": {
    "task_post": {
      "source_kind": "purchase",
      "granted": 10,
      "consumed": 3,
      "remaining": 7,
      "events": [
        {
          "created_at": "2026-08-20T12:00:00Z",
          "task_id": "...",
          "action": "create",
          "quantity": 1,
          "source_kind": "purchase"
        }
      ]
    }
  }
}
```

- `granted/remaining` 来自该 `order_id` 的 task_post 批次；无批次则为 0。
- `consumed = granted - remaining`（下限 0）。
- `events`：该订单 `related_order_id` 且 `points_source_type=quota_consumption` 的流水（`transaction_id` 含 `task_post_renewal` → `action=renewal`，否则 `create`）。
- 2026-09-03：已支付订单的 `gitlab_disk` / `gitlab_traffic` 同步写入 `resource_consumption`（`granted`=本单行项数量，`consumed/remaining` 按该区域当前配额与用量对新订单优先分摊）。无任务帖时不得再发 `task_post` 全 0。

### 存量回填

幂等 `ensureTaskPostPurchaseLots(tenantID)`：对已支付且 `payment_method != admin_grant` 的任务帖行项补 purchase 批次；将「账户购买剩余」按 **LIFO（新订单先留 remaining）** 摊到批次。在配额 GET / 订单 GET 时若发现缺口则执行（幂等，禁止启动路径 DDL）。

### 退款

既有 `revokeOrderResourcesOnRefund` 仍按行项扣账户配额；同时将该订单 task_post 批次 `remaining` 置 0。

### UI

- 账单卡：合计大字；下一行 `赠送 N · 购买 M`。
- 订单展开/详情：行项下增加「资源消耗」区块（无任务帖行项则不渲染）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 查看配额拆分 | — | — | — | 纯 GET |
| 查看订单消耗 | — | — | — | 纯 GET |
| 创建/续存消耗配额 | BILLING_TRANSACTION_CREATED | consumeAndRecord / consumeTaskPostRenewal outbox | 既有 billing 消费者 | 扩展 payload，不新开 topic |
| 支付发放购买批次 | 既有订单支付流水 | markOrderPaid | — | 不另发事件 |

## 🕸️ Code Review Graph 分析

CRG `update --brief` 成功；本增量符号（`handleResourceQuotas`、`consumeTaskPostQuotaTx`、`markOrderPaid`、`orderJSON`）不在近期 FTS 增量节点中。设计基于源码阅读（无 codegraph MCP）。`CRG unavailable for symbol explore: MCP codegraph 未接入；CLI 增量 0 nodes`。

## 价值流影响（Step 1 输入）

- 影响：租户账单配额展示、订单详情/列表展开、配额消耗写路径。
- 新流：`task-post-quota-source-and-order-consumption`（见 Step 4 YAML）。
- 字段：`billing_resource_grant.source_kind`、`order_id`；`billing_transaction.source_grant_id`、`related_order_id`。

## 🏛️ 架构变更影响

- **迭代版本:** v91 🎯 target
- **迭代名称:** 任务帖配额来源与订单消耗归属
- **作者:** cursor
- **设计日期:** 2026-08-20 19:10
- **新增文件**（每个视图四类伴生格式）:
  - `docs/architecture/v91-application-integration-20260820-1910-cursor.puml` + `.diff.archimate` + `.full.archimate` + `.mermaid.md`
  - `docs/architecture/v91-enterprise-landscape-20260820-1910-cursor.puml` + 伴生三件套
- **based_on:** v88 current（v89/v90 为正交 target，不合并）
- **变更明细:** 🟡 taskBill 配额/订单 GET 扩展；🟡 `billing_resource_grant` 批次语义扩展为赠送+购买；无新服务。

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| `.diff.archimate` | Plateau v88 → Gap「配额合计无法区分来源、消耗无订单归属」→ WP → Plateau v91；目标拓扑含 taskFE↔taskBill↔grant/txn |
| `.full.archimate` | 同上全量运行时拓扑（本增量范围） |

## 决策锁定

- 不改消耗单价（仍为预购 0 元流水）。
- GitLab 磁盘/流量本增量不拆赠送/购买（范围外，记 OPT）。
- 历史消耗事件在回填前可能为空；剩余以批次为准。
