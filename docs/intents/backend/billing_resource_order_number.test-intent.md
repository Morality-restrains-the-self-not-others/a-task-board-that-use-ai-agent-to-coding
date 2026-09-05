# 测试意图：资源订单展示号全局唯一

## 覆盖

1. `generateOrderNumber(tenantID, id)` 返回 `ORD-{UTC日期}-{tenantID}-{id}`。
2. `orderID<=0` 返回错误。
3. 两个不同 `tenant_id` 调用 `createOrder`，得到不同 `order_number`，各自含自己的 `tenant_id` 与 `id`。
4. 既有 1062 重试与并发唯一测例仍通过。

## 不覆盖

- 存量 NNN 号码迁移改写。
- 匿名跨租户按号搜索（明确不做）。staff 等值查询见 `billing_admin_order_trade_no_query`。
