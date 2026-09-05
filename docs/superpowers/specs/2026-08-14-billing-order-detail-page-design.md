# 创建订单后独立详情页 — 设计

- **Date:** 2026-08-14
- **Status:** accepted（goal-mode 自动采用）
- **Architecture change:** 否（无新服务 / 新 API / 新事件；不更新 ArchiMate）

## Goal / 成功标准

1. 点击「创建订单」且 POST 成功 → URL 变为 `/tenant/{tid}/billing/orders/{orderId}/`。
2. 详情页展示订单头（订单号、金额、状态）+ **资源行项表**。
3. 待支付可在详情页支付；失败留在创建页并带 `data-traceId`。
4. 不新增 HTTP API：复用 `GET/POST .../billing/orders/` 与既有 pay（无 mock-complete）。

## 现状

`OrderCreate.vue` 在 POST 成功后把 `currentOrder` 设在**同一页**，内嵌「订单详情」仅有订单号/金额/状态，**不展示 items**。后端 `orderJSON` 已返回 `items[]`（`resource_type/quantity/unit_price_yuan/subtotal_yuan`）。

## 决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 详情形态 | 独立路由页面 | 用户明确要求单独页面 |
| 数据源 | GET 订单详情 | 可刷新、与列表展开同源 |
| 支付落点 | 迁到详情页 | 离开创建页后否则无法支付 |
| 权限 | 复用 `billing.orders` / `billing.orders.main` | 非新侧栏页 |
| 架构图 | 不升级 | 无应用组件/事件拓扑变化 |

## 方案

```
购买资源页 --POST create--> 独立详情页 --GET order--> 渲染 items + 支付
```

- 路由：`/tenant/:tenant/billing/orders/:orderId/`，`name=billing_order_detail`，注册在 `create` 之后。
- 创建成功：`router.push({ name, params })`（表单提交后的编程导航，非伪链接）。
- 列表订单号用真实 `<a href>` 进入同一详情页。
- `router.js` 已超 500 行：拆为 `router/publicRoutes.js` + `tenantRoutes.js` + `adminRoutes.js`。

## 备选（拒绝）

- 创建页内嵌补 items：不满足「单独一个页面」。
- 新建 GET 聚合 API：现有 GET 已含 items。
- 详情只用 PayOrderModal、创建页保留支付：会双路径，用户离开创建页即丢失支付。

## API（无变更）

`GET /api/tenant/{tenant_id}/billing/orders/{order_id}/` → `orderJSON`（含 `items`）。

## 权限

见 `*-permission-analysis.md`。无新 endpoint。
