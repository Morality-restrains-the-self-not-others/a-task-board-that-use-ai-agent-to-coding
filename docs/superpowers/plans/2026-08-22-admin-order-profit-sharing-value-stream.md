# 价值流 — 管理员查看订单分账

- **日期**: 2026-08-22
- **设计**: `docs/superpowers/specs/2026-08-22-admin-order-profit-sharing-design.md`

## Related Value Streams

- `2026-08-21-system-admin-order-trade-no-query-value-stream.md` — 本增量**扩展**订单记录页展开内容，不改交易单号查询。
- `2026-08-20-task-post-quota-source-and-order-consumption-value-stream.md` — 展开区已有消耗；本增量在其下增加分账块。
- `2026-08-14-billing-order-detail-page-value-stream.md` — 租户详情保持无分账。

## 增量

超管打开订单记录 → 展开一行 → 看到分账接收方与金额；租户展开同一组件看不到该块。

```
超管 /system-admin/order-records/
  → 展开订单行
  → GET /api/system-admin/orders/{id}/
  → profit_sharing[] 渲染
租户 /tenant/{tid}/billing/orders/
  → 展开
  → GET /api/tenant/{tid}/billing/orders/{id}/
  → 无 profit_sharing 键、无分账 UI
```

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| VS-PS-1 | staff 展开有分账单 | 见接收方与金额 |
| VS-PS-2 | staff 展开无分账单 | 「本订单无分账记录」 |
| VS-PS-3 | 租户展开 | 无「分账」标题 |
| VS-PS-4 | 租户 GET | JSON 无 profit_sharing |
| VS-PS-5 | member 调 admin GET | 403 |

测试落点：Go `handlers_admin_get_order_test.go` + Vue `OrderExpandDetail.profitSharing.test.js`。不写入 Django `conf/value-stream.yaml`（本增量无 pytest 路径）。
