# 价值流 — 任务帖配额来源与订单消耗归属

- **Date:** 2026-08-20
- **Design:** `docs/superpowers/specs/2026-08-20-task-post-quota-source-and-order-consumption-design.md`

## 增量切片（按用户价值）

1. **看见来源**：账单卡赠送/购买剩余（只读 API + UI）。
2. **批次落地**：支付写入 purchase grant；存量 LIFO 回填。
3. **消耗可追溯**：消耗扣批次并写 related_order_id；订单展开/详情展示消耗。

## 步骤顺序

配额展示不依赖批次回填（purchased = total − gifted）。订单消耗数字依赖批次；events 依赖新消耗。故可先交付切片 1，再 2+3 同发以免「有拆分无归属」。

## YAML 增量

见 `conf/value-stream.yaml` 流 `task-post-quota-source-and-order-consumption`。
