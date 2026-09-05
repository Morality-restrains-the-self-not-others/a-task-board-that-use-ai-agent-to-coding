# 价值流 — 管理员查看待分账队列

- **日期**: 2026-08-22
- **设计**: `docs/superpowers/specs/2026-08-22-admin-pending-profit-sharing-list-design.md`

## Related Value Streams

- `2026-08-22-admin-order-profit-sharing-value-stream.md` — 本增量**扩展**同一页：从「逐单展开看分账」变为「队列总览」。
- `tenant-refund-application` — 同页退款 Tab 不变。

## 增量

超管打开订单与退款页 → 点「待分账」→ 看到未完成分账订单（默认 open）→ 可点订单号回到订单 Tab。

```
超管 /system-admin/order-records/?tab=profit-sharing
  → GET /api/system-admin/profit-sharing/?status=open
  → 表格渲染
  → 订单号 href → /system-admin/order-records/?tenant_id=&order_id=
租户
  → 无入口；调 admin API → 403
```

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| VS-PSQ-1 | staff 打开待分账 Tab | 见 pending 行 |
| VS-PSQ-2 | 默认筛选 | 不含 finished |
| VS-PSQ-3 | 空队列 | 「暂无待分账订单」 |
| VS-PSQ-4 | 订单号链接 | 指向订单 Tab 深链 |
| VS-PSQ-5 | member GET | 403 |
| VS-PSQ-6 | JSON | 无 openid |

测试落点：Go `handlers_admin_list_profit_sharing_test.go` + Vue tabs/panel 单测。不写入 Django `conf/value-stream.yaml`。
