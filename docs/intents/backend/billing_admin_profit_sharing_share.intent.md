# 功能意图：超管查看微信分账单号并带审计缘由发起分账

## 意图

系统管理员在推荐绩效「微信分账」Tab（及待分账队列）看到微信支付订单号、微信分账单号，并可对 pending/failed 行填写缘由后直接向微信发起分账。

## 角色

平台员工（super_admin / staff）。

## 行为

1. GET `/api/system-admin/profit-sharing/` 每行含 `wechat_transaction_id`、`wechat_profit_sharing_id`（可空）；不含 openid。
2. POST `/api/system-admin/profit-sharing/{id}/share/`，JSON `{reason}` 8–500 字，`Idempotency-Key`；绕过推荐人 15 天冻结。出站金额为 `min(台账佣金, floor((订单净额−已退款)×分成比例/100), 微信剩余待分金额)`；净额过小不足 1 分则 409，不调微信。
3. 前端分账按钮同步门闩 + 同意图同一幂等键；失败 toast/内联错误 `data-traceId`。出站失败时把当次 `trace_id` 写入 `billing_profit_sharing.fail_trace_id`，列表失败原因列复用该值。微信业务拒单（INVALID_REQUEST / RULE_LIMIT / NOT_ENOUGH）返回 **409** 及微信 `message`，不是 502。
4. 审计表记录 actor / impersonation / reason。

## 非目标

- 不向推荐人页回传微信单号
- 不改 25 天兜底

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 例外理由 |
|---------|--------|----|--------|---------|
| 超管发起分账 | — | 出站微信 CreateOrder | `handleSystemAdminShareProfitSharing` → `executeProfitSharing` | 与扫描/推荐人手动分账同一聚合写路径 |

## 变更记录

- 2026-08-25：超管单号列 + 带缘由分账
- 2026-08-26：失败落库 `fail_trace_id`
- 2026-08-26：出站金额按净额×比例向下取整；微信业务 400 映射 409
