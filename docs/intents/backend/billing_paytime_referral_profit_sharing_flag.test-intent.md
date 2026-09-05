# 测试意图：租户微信支付按支付时刻推荐资格设置分账标识

对应功能意图：`billing_paytime_referral_profit_sharing_flag.intent.md`

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 付款人有推荐边且支付时资格 active | Native `PrepayRequest.SettleInfo.ProfitSharing==true` |
| T2 | 无推荐边 | 预下单打标已改 ADR-0040：见 `billing_order_always_profit_sharing_flag.test-intent.md` T2 |
| T3 | 边 `commission_eligible=0`，支付时资格 active | 仍设置 `ProfitSharing==true`（不以绑边快照为准） |
| T4 | 边 `commission_eligible=1`，支付时资格 inactive | 预下单仍打标（ADR-0040）；本文件只断言不落佣金行 |
| T5 | 资格 HTTP 非 200 / 超时 | 预下单仍打标；不落佣金行 |
| T6 | T3 条件下支付成功 `markOrderForProfitSharing` | 落 1 条 `billing_profit_sharing` |
| T7 | 支付时资格 inactive | `markOrderForProfitSharing` 不落行 |
| T8 | taskReferral GET 有活跃 approved | `{"active":true}` |
| T9 | taskReferral GET 无码 / revoked / expired | `{"active":false}` |
| T10 | taskReferral GET 缺 `user_id` | 400 |

映射：

- `taskBill/src/wechat_pay_profit_sharing_flag_test.go`
- `taskBill/src/wechat_profit_sharing_record_test.go`
- `taskReferral/src/referral_qualification_internal_test.go`

价值流：`docs/flows/value-stream-test-integration.wsd` VS-PS-4 / VS-PS-5 / VS-PS-6
