# 功能意图：租户购买 GitLab 资源须按行指定区域

## 意图

租户在「购买资源」页购买 GitLab 磁盘或流量时，必须显式选择 GitLab 区域；订单行写入该 slug，禁止静默默认区。购买页将磁盘与流量放在同一卡片内、共用一个区域下拉（见 [order_create_gitlab_shared_region](../frontend/order_create_gitlab_shared_region.intent.md)）。后端 API 仍按行接收 `region`，同一请求两行可带相同 slug。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 租户创建含 GitLab 行的订单 | — | — | `createOrder` 写 pending 订单 | 待支付 | 下单本身不发放配额；观测用 `resource_order_created` 日志 |
| 订单支付成功发放配额 | BillingTransactionCreated | BILLING_TRANSACTION_CREATED | `markOrderPaid` outbox（既有） | 账单流水 | 本增量不改支付路径，仅保证行项 region 在下单时已合法 |

## 验收

- GitLab 行缺 region 或 slug 未启用 → 400，不写默认 `tencent-shanghai-5`
- 购买页磁盘与流量合卡，共用一个区域下拉；空区域不 POST
- 同时购买时两行写入同一所选 slug；后端仍按行存储 region
- 订单详情展示 GitLab 行 region
- GitLab 设置页购买入口指向 `/tenant/{tid}/billing/orders/create/`
