# 功能意图：微信分账动账通知

## 意图

提供 HTTPS POST 接口供微信支付商户后台「分账动账通知」配置。验签并解密后按 `out_order_no` 幂等更新本地分账台账。

## 角色

微信支付服务器（无用户会话）。

## 行为

1. URL：`https://www.daydaymoney.com/api/billing/profitsharing/change-notify/`（须 HTTPS、无 query）。
2. 验签 + AEAD：`wechatNotifyH.ParseNotifyRequest`。
3. 以 `id` 去重；已处理返回 `{code:SUCCESS}`。
4. 更新 `wechat_profit_sharing_id`、`wechat_transaction_id`（订单列）、成功则 `status=finished`。
5. 存量 `/api/billing/profitsharing/notify/` 走同一处理函数。

## 业务意图 → 事件对照

| 业务意图 | 事件 | 例外 |
|---------|------|------|
| 接收微信动账结果 | 无内部 MQ | 微信入站对账，更新同一分账聚合 |

## 变更记录

- 2026-08-25：动账通知正式化（core/notify）
