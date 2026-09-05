# 零额订单禁止开票 — NFR 澄清

价值流：`docs/superpowers/plans/2026-08-29-zero-amount-invoice-block-value-stream.md`

资金/税务路径 **L3**。

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| POST /api/tenant/{tid}/billing/orders/{oid}/invoice-applications/ | tenant_id + order_id | 是，与库分片一致 | 零额判断读本行 total_yuan_cents，禁止跨单扫描 |
| POST /api/system-admin/invoice-applications/{id}/approve/ | 申请主键 | 员工路径可接受 | 加载申请后用行上 tenant_id+order_id 读订单金额 |
| 前端 /tenant/{tid}/billing/orders/{oid}/ | tenant_id + order_id | 是 | 用订单 JSON 已有金额字段，无新路径 |
| 拒绝路径（无 MQ） | — | L0 | 失败意图不投递；无消费键 |

升级触发：无新列表/无新全局索引。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| POST 申请（金额 > 0） | 写申请 | 双击 | 一单一 pending | Idempotency-Key + unique pending(order_id) | 既有：同键回放；无键 409 |
| POST 申请（金额 ≤ 0） | **无**（校验失败） | 双击/脚本重试 | 该 order_id 零额 | 无写则无键 | 每次 400，不插入、不发事件 |
| POST approve（金额 ≤ 0） | **无** | 员工连点 | 该申请+订单金额 | 申请仍 pending | 每次 400，不落蓝票 |
| 只读订单 GET / 前端展示 | 无 | 刷新 | — | L0 | |

前端：零额路径无写按钮（display-only）。正额申请仍 `createClickGuard` + `Idempotency-Key`。

禁止用 `tenant_id` / `user_id` 作消费幂等键（本增量无新消费者）。

## 类别支撑程度

| 类别 | 等级 | 说明 |
|------|------|------|
| 数据一致性 | L3 | 写前校验金额；拒绝不留半截申请 |
| 容错 | L2 | 校验失败立即 400，无需补偿 |
| 安全 | L3 | 前端隐藏不替代后端；鉴权顺序不变 |
| 可伸缩性 | L0 | 单行主键读取 |
| 可观测性 | L2 | warn：`invoice_application_rejected_zero_amount` / `invoice_approve_rejected_zero_amount` |

## 领域模型影响

- InvoiceApplication 不变量：所属订单 `TotalYuanCents > 0`。
- 无新聚合、无新事件。失败意图书面例外：不 publish。
