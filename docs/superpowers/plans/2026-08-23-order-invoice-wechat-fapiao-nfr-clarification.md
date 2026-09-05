# 订单电子发票 NFR 澄清

价值流：`docs/superpowers/plans/2026-08-23-order-invoice-wechat-fapiao-value-stream.md`

资金/税务路径默认 **L3**。

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 动作 |
|------|---------|----------|------|
| POST /api/tenant/{tid}/billing/orders/{oid}/invoice-applications/ | tenant_id + order_id | 是，tenant 与库分片一致 | 查询必带 tenant_id |
| GET 同上 invoices / 订单 GET | tenant_id + order_id | 是 | |
| GET /api/system-admin/invoice-applications/ | 无租户键 | 跨租户员工队列 | L1：分页 + status 过滤；禁止无 limit |
| POST /api/system-admin/invoice-applications/{id}/approve\|reject | 申请主键 | 员工路径可接受 | 加载后用行上 tenant_id 写库 |
| POST /api/billing/wechat/fapiao/notify/ | fapiao_apply_id / fapiao_id | 按发票主键定位 | 不扫全表 |
| 前端 /tenant/{tid}/billing/orders/{oid}/ | tenant_id | 是 | |
| 事件 BILLING_INVOICE_* | tenant_id + invoice/application id | 禁止只用 tenant_id 作幂等键 | 键=申请或发票 ID |

升级触发：员工列表 QPS 持续 > 50 再考虑分区。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| POST 申请 | 写申请 | 双击 | 一单一 pending | Idempotency-Key + unique pending(order_id) | 同键回放 201/200；无键 409 |
| POST approve | 微信开票 | 双击/网关重试 | 一申请一次开具 | 申请 id；微信 RESOURCE_ALREADY_EXISTS 转查询 | 已 approved 再 approve 返回当前票 |
| POST reject | 改状态 | 双击 | 申请 id | 申请 id | 非 pending 400 |
| 微信开具 | 出站写 | 回调+重试 | fapiao_id | fapiao_id 商户唯一 | 已存在则查单 |
| 微信冲红 | 出站写 | 退款重试 | fapiao_apply_id + fapiao_id | 同上 | 已 REVERSED 则跳过 |
| 重开 | 出站写 | 退款重试 | 新 fapiao_id 由 refund_application_id 派生 | `reissue-{refundAppID}` | 已有 reissue 行则跳过 |
| 回调 notify | 改发票状态 | 微信重放 | 通知 id / 发票状态机 | 状态机单向 | ISSUED/REVERSED 不回退 |
| 只读 GET | 无 | — | — | L0 | |
| GET 订单 invoices / invoice_reverse_confirm | 无（计算截止时间） | 刷新 | 无写 | L0 | 禁止 GET 改库；超时只改展示 status |
| FAPIAO.REVERSED 回调 | 蓝票 reversed / 红票 issued | 微信重放 | 同一 apply_id 状态机 | wechat_apply_id | 已 reversed 不回退；禁止进程内扫 72h |

前端申请/审批按钮：`createClickGuard` + 同一意图同一 Idempotency-Key。

## 类别支撑程度

| 类别 | 等级 | 说明 |
|------|------|------|
| 数据一致性 | L3 | 申请唯一约束 + 微信查单；退款本地提交后发票最终一致 |
| 容错 | L3 | 开具 202 后靠回调/查询；冲红失败记 reverse_failed 可重试 |
| 安全 | L3 | 验签回调；PII 不进日志 |
| 可伸缩性 | L1 | 员工列表分页 |
| 可观测性 | L2 | 结构化事件名 + order_id/invoice_id |

## 领域模型影响

- 聚合：InvoiceApplication、Invoice（按 order_id 归属）。
- 退款聚合不内嵌发票，退款成功后领域服务 `ReconcileInvoicesAfterRefund`。
- 幂等键与申请/发票 ID 同粒度。
