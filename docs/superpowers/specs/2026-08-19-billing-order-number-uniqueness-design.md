# 资源订单号全局唯一 — 设计

- **Date:** 2026-08-19
- **Status:** accepted（goal-mode 自动采用）
- **Architecture change:** 否（无新服务 / 新 API / 新事件；不更新 ArchiMate）
- **ADR:** [ADR-0017](../../adr/0017-globally-unique-resource-order-number.md)

## Goal / 成功标准

1. 多租户同日创建订单，列表展示的 `order_number` **互不相同**。
2. 新单格式为 `ORD-YYYYMMDD-{tenantId}-{snowflakeID}`（[ADR-0018](../../adr/0018-order-id-tenant-shard-routing.md)），**末段**等于主键 `id`。
3. 分配 **不** `SELECT MAX` 日序号；不再受每日 999 上限约束。
4. `UNIQUE(order_number)` 仍在；1062 仍重试换新 ID。
5. 存量 `ORD-YYYYMMDD-NNN` 仍可展示与支付；URL 仍用 Snowflake `id`。
6. 前端列表长号码不撑破表格（`break-all`）。

## 现状

页面 `https://www.daydaymoney.com/tenant/{tid}/billing/orders/` 第二列链接文案为 `ORD-20260818-003`：

```html
<a class="text-gray-900 hover:text-primary hover:underline"
   href="/tenant/877397588196749312/billing/orders/877596007691485184/">ORD-20260818-003</a>
```

- **路由身份** = Snowflake `id`（已全局唯一，租户路径另有 `tenant_id`）。
- **展示身份** = `order_number`，当前为 **全平台** 日序号 `NNN`。

会冲突吗？

| 维度 | 当前 | 多租户后果 |
|------|------|------------|
| 字符串 UNIQUE | 全局唯一，两租户不会落同一号码 | 号码本身不重复，但 **争抢同一计数器** |
| 生成算法 | `SELECT MAX` + `%03d` | 并发 1062（已生产发生）；日单 >999 格式失控 |
| 若改成每租户 001 | 短号好看 | **会冲突**：管理端/客服看到两个 `ORD-20260818-003` |

## 决策

采用 **Snowflake 派生的全局展示号**（ADR-0017），拒绝每租户短序号。

```
createOrder / insertOrderWithRetry
  → orderID = generateSnowflakeID()
  → orderNumber = ORD-{UTC日期}-{tenantID}-{orderID}
  → INSERT UNIQUE(order_number)
```

不新增 HTTP、不改支付 `out_trade_no`、不发新领域事件（沿用现有下单日志 `resource_order_created`）。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：增量 23 文件，本主题符号尚未入图（生成函数为包级 `main.generateOrderNumber`）。
- 影响面（手工）：`createOrder`、`insertOrderWithRetry`（`adminGrantResources` / `backfillGrantOrders`）、测试 `orders_duplicate_test.go`、FE `BillingOrders.vue` / `SystemAdminOrderRecords.vue`。
- `CRG unavailable for symbol-level query on unindexed generateOrderNumber`：以源码调用链为准。

## API（无契约变更）

`order_number` 仍是字符串字段；只是新值更长。客户端不得假设 `\d{3}$`。

## 权限

无新 endpoint。租户列表仍按 `tenant_id` 过滤；管理员全量列表依赖全局唯一号码避免歧义。见 permission 分析。
