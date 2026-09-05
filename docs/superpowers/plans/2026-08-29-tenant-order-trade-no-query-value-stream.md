# 价值流：租户订单列表按交易单号 / 商户单号查询

- **日期**: 2026-08-29
- **设计**: `docs/superpowers/specs/2026-08-29-tenant-order-trade-no-query-design.md`

Mapping the approved design into a value stream.

## 端到端价值流

```
用户打开租户订单列表
  → 从微信凭证粘贴交易单号或商户单号
  → 点击查询
  → GET 本租户 orders/?order_number=
  → 看到自己租户的匹配订单（或空列表）
```

## 最小可行增量

单一增量即可交付：前端查询 UI + 既有 API 契约测试 + 越权回归。不拆第二期。

| 增量 | 价值 | 测试 |
|------|------|------|
| VS-TO-1 交易单号查询 | 粘贴微信支付交易单号得到本租户订单 | FE: 请求带 `order_number`；Go: 租户 GET 命中 `wechat_transaction_id` |
| VS-TO-2 商户单号查询 | 粘贴商户订单号得到本租户订单 | FE: 商户单号输入发出 `order_number`；Go: 命中 `out_trade_no` |
| VS-TO-3 空查询 | 不填单号保持原列表 | FE: URL 无 `order_number` |
| VS-TO-4 越权 | 他租户单号查不到 | Go: 跨租户 `total=0` |

## 事件

无。纯 GET。
