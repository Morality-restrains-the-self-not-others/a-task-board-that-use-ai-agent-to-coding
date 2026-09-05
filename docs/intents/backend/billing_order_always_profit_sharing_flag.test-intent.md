# 测试意图：租户微信支付对所有订单打分账标识

对应功能意图：`billing_order_always_profit_sharing_flag.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 付款人有推荐边且支付时资格 active | Native `SettleInfo.ProfitSharing==true` |
| T2 | 无推荐边 | **仍** `ProfitSharing==true`（不再 omit） |
| T3 | 边 `commission_eligible=0`，支付时资格 active | `ProfitSharing==true` |
| T4 | 边存在但支付时资格 inactive | **仍** `ProfitSharing==true` |
| T5 | 资格 HTTP 失败 | **仍**打标（预下单不再 fail-closed） |
| T6 | T3 条件下支付成功 `markOrderForProfitSharing` | 落 1 条 `billing_profit_sharing` |
| T7 | 无推荐人或支付时资格 inactive | 不落佣金行 |
| T8 | 微信支付成功且无佣金行 | 调用 Unfreeze，`out_order_no=UF{order_id}` |
| T9 | 已有佣金行 | 支付回调不调用 Unfreeze |
| T10 | mock 预下单 | 不打微信；不解冻 |

映射：

- `taskBill/src/wechat_pay_profit_sharing_flag_test.go`
- `taskBill/src/wechat_profit_sharing_record_test.go`
- 解冻：`taskBill/src/wechat_pay_profit_sharing_unfreeze_test.go`

价值流：`docs/flows/value-stream-test-integration.wsd` VS-PS-4 / VS-PS-5 / VS-PS-7
