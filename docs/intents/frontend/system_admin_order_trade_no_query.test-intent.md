# 测试意图：系统管理订单页按微信账单单号查询

## 覆盖

1. 空输入不 emit search。
2. 普通展示号 trim 后 emit。
3. 带前导反引号的微信支付单号 emit 时已去掉反引号。
4. 展开详情在详情 payload 含 `out_trade_no` / `wechat_transaction_id` 时展示两列。

## 可执行测试

- `taskFE/app/src/components/system-admin/OrderNumberPasteJump.test.js`
- `taskFE/app/src/components/system-admin/OrderExpandDetail.profitSharing.test.js`
- `taskFE/app/src/components/system-admin/SystemAdminOrderListPanel.orderNumberPaste.test.js`
