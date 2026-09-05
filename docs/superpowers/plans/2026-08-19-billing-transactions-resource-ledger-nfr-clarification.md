# NFR 澄清 — 交易记录资源流水账

- **Date:** 2026-08-19
- **NFR 默认档：** 资金展示 L3 准确；查询性能 L2

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| `GET /api/tenant/{tenant_id}/billing/transactions/` | `tenant_id` | 是：与 `billing_account.tenant_id` 一致 | 保持 |
| `GET /api/tenant/{tenant_id}/billing/transactions/list_filtered/` | `tenant_id` | 是 | 保持；回放 SQL 必须带该租户 `account_id` |
| 前端 `/tenant/:tenant/billing/transactions/` | `tenant` | 是 | 保持 |
| Kafka | 无本增量新路径 | — | — |

可伸缩性：路径已带租户键 → **L2**（租户级列表 + 上限 5000 / 分页 page_size≤200）。升级触发：单租户流水超百万且列表 P95>500ms → 考虑按时间分区裁剪（表已 RANGE 分区）。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 幂等键 | 判定 | 等级 | 重放语义 / 动作 |
|------|--------|------------|--------|------|------|-----------------|
| GET `/billing/transactions/` | 无（只读；`getOrCreateBillingAccount` 对已存在账户无插入） | — | — | 只读 | L0 | 无副作用；重复 GET 结果随数据自然变化 |
| GET `/billing/transactions/list_filtered/` | 无（只读回放 SQL，不写流水） | — | — | 只读 | L0 | 无副作用 |

本增量无写路径 → 不引入消费幂等键。禁止用 `tenant_id` 当未来写事件幂等键（本增量无写事件）。

## 类别支撑程度

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全性 | L3 | 租户隔离；SQL 参数化 |
| 数据准确性（账目） | L3 | 现金用落库 before/after；任务帖剩余用未过滤回放 |
| 性能 | L2 | 每页一次回放查询（从最旧行时间起），禁止 N+1 grant |
| 可观测性 | L2 | enrich 失败 slog warn + tenant_id + trace |
| 可伸缩性 | L2 | 见上表 |

## 质量场景

1. **刺激**：筛选只看消耗。**响应**：现金瞬时仍正确；任务帖剩余与未过滤历史一致，不因筛选跳行而错。
2. **刺激**：`amount_points=0` 的配额消耗。**响应**：变动列显示数量+单元，不是 0.00 元。
3. **刺激**：grant enrich SQL 失败。**响应**：列表仍 200，缺 `change_display` 时前端回退；打 warn 日志。

## 领域模型影响

`LedgerSnapshot` 为查询侧值对象，不进入写聚合一致性边界。不新增领域事件。
