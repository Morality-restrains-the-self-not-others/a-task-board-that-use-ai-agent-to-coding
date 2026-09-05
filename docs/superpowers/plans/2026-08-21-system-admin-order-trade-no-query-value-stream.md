# 价值流 — 按交易单号查询订单

- **日期**: 2026-08-21
- **设计**: `docs/superpowers/specs/2026-08-21-system-admin-order-trade-no-query-design.md`

## 增量

客服/超管拿着展示订单号、订单主键或支付渠道交易号，在订单记录页一次查询定位订单。

```
超管打开 /system-admin/order-records/
  → 输入交易单号 → 查询
  → GET /api/system-admin/orders/?order_number=
  → 命中：列表展示并展开
  → 未命中：空态
```

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| VS-TN-1 | 四段 order_number | 返回该单 |
| VS-TN-2 | 纯 id | 返回该单 |
| VS-TN-3 | payment_ref | 返回该单 |
| VS-TN-4 | 未知单号 | total=0 |
| VS-TN-5 | 空输入（前端） | 不请求 |
| VS-TN-6 | 非 staff | 403 |
