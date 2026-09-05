# 测试意图：管理端按微信关联账号查询资源订单

## 覆盖

1. 解析到的 user_id 命中其订单。
2. 仅 `pay_openid` / `pay_unionid` 等值命中（user_id=0）。
3. 无命中空列表 200。
4. 与 `order_number` 同时传 400；超长 400。
5. taskAuth 非 200 → 管理端 502。
6. 未鉴权 401；非 staff 403。
7. OpenAPI 列出 `wechat_account`。

## 可执行测试

- `taskBill/src/orders_list_wechat_account_test.go`
