# 测试意图：微信分账动账通知

## 对应功能意图

`billing_profit_sharing_change_notify.intent.md`

## 用例

1. mock 模式 POST 明文 `out_order_no` + `transaction_id` + `order_id` → 200 SUCCESS，台账 finished 且写入微信单号。
2. 同一 `id` 再 POST → 200 SUCCESS，updated_at 不因二次业务写而乱序（inbox 命中跳过）。
3. live 且 `wechatNotifyH==nil` → 非 SUCCESS。
4. 未知 `out_order_no` → 仍 200 SUCCESS（避免微信积压；记 warn）。
