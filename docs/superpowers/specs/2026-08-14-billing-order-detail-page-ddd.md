# 订单详情页 — DDD

限界上下文：Billing（taskBill）已有 `ResourceOrder` + `ResourceOrderItem`。本增量**不新增领域对象**。

## 复用

- 聚合：ResourceOrder（根）含 items
- 读端口：既有 `loadOrder` / `handleGetOrder`
- 写：既有 `createOrder`（创建页）

## 事件

无新领域事件。书面例外：纯查询 + SPA 导航。

## 前端应用服务（非领域层）

- OrderCreate：组 items → POST → `router.push`
- OrderDetail：GET → 展示 items → 待支付走既有 pay/SSE
