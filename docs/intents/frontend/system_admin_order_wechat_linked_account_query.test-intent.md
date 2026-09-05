# 测试意图：订单记录页按微信关联账号查询

## 覆盖

1. Tab 切换：默认交易单号；点「微信关联账号」出现微信输入框。
2. 空输入不发请求并提示。
3. 非空查询请求管理端列表带 `wechat_account`，不带 `order_number`。
4. 命中展示订单；未命中空态「未找到该微信关联账号」。
5. 成功后 `router.replace` 写入 `?wechat_account=`。
6. `?wechat_account=` 深链自动查询并选中微信 Tab。
7. 交易单号 Tab 回归：仍按 `order_number` 查询。

## 可执行测试

- `taskFE/app/src/components/system-admin/WechatLinkedAccountQuery.test.js`
- `taskFE/app/src/components/system-admin/AdminOrderLookupBox.test.js`
- `taskFE/app/src/components/system-admin/SystemAdminOrderListPanel.wechatAccount.test.js`
- `taskFE/app/src/composables/useSystemAdminOrderListDeepLink.test.js`
