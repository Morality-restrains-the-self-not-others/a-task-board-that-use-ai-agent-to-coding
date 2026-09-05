# DDD — 管理端分账表 AppID / OpenID

- **日期**: 2026-08-26
- **NFR**: `docs/superpowers/plans/2026-08-26-admin-profit-sharing-appid-openid-columns-nfr-clarification.md`

## 限界上下文

`taskBill` 资金/分账。不跨服务。

## 既有实体（不新增）

- `billing_profit_sharing`：分账台账；`referrer_openid` 为出站快照
- `billing_profit_sharing_receiver`：接收方登记；`appid` + `openid`

## 查询 DTO

管理端队列项增加只读 `app_id`、`openid`。有接收方时成对取自登记行（`wechat_identity` 支付/mp 投影）；禁止与台账快照混源。

## 事件

无。纯查询例外，意图文档已书面标注。
