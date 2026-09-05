# 测试意图：资源订单展示号带租户基因

## 覆盖（批准实施后）

1. `generateOrderNumber(tenantID, orderID)` 返回 `ORD-{UTC日期}-{tenantID}-{orderID}`。
2. `tenantID<=0` 或 `orderID<=0` 返回错误。
3. 两租户 `createOrder` 得到不同 `order_number`，各自含自己的 `tenant_id` 与 `id`。
4. `ParseResourceOrderNumber`：四段得到 tenant+id；三段视为无租户基因。
5. 基因 tenant 与行不一致 → 加载失败（不存在）。
6. `loadOrder(tid, oid)` 同租户命中、错租户 404；pay/cancel 错租户同样 404。
7. 既有 1062 重试仍通过。
8. 订单 PK 仍为标准 Snowflake，不是「tenant+id」拼出来的整数。
9. `markOrderPaid(..., tenantID=0)` 使用行上 `tenant_id` 发放配额；错租户调用失败且订单仍 pending。

## 非覆盖

- 存量号码改写。
- 匿名公网按 order_number 查询 API。
- 真实分库路由。

## 变更记录

- 2026-08-19：与 ADR-0018 四段展示号提案同步。
