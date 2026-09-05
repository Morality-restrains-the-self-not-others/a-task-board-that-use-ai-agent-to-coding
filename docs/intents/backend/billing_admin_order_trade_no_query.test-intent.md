# 测试意图：管理端按微信支付单号 / 商户订单号查询资源订单

## 覆盖

1. `payment_ref=wechat:WX…` 时，粘贴无前缀商户订单号仍命中。
2. 订单列 `wechat_transaction_id` 命中微信支付单号。
3. 导出串带前导反引号时 normalize 后命中。
4. 本地无微信支付单号、管理端 stub 微信 API 返回 `out_trade_no` 后命中，并回写 `wechat_transaction_id`。
5. 入账 `persistWechatPayVouchers`：`payment_ref` 带 `wechat:` 前缀时台账 `provider_capture_id` 仍能按去前缀 `provider_ref` 写上。
6. `recordWechatTransactionIDForOrder` 回归：前缀 `payment_ref` 不再 0 行更新。

## 可执行测试

- `taskBill/src/orders_list_trade_no_test.go`
- `taskBill/src/wechat_pay_vouchers_test.go`
- `taskBill/src/wechat_profit_sharing_txnid_test.go`
