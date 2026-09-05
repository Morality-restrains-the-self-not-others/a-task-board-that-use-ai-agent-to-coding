# 功能意图：管理端按微信关联账号查询资源订单

## 意图

平台员工在系统管理订单记录页输入微信关联账号后，服务端解析关联用户并返回其资源订单（含仅有支付 openid/unionid 的订单），无需先选租户。

## 角色

- 系统管理员 / 平台员工：跨租户查询

## 行为

1. `GET /api/system-admin/orders/` 接受可选 query `wechat_account`（1–128 字符）。
2. 与 `order_number` 同时出现：HTTP 400。
3. 调用 taskAuth `wechat-linked-account` 解析 `user_ids`；同时等值匹配本库 `pay_openid` / `pay_unionid`。
4. `user_id IN (...) OR pay_openid = q OR pay_unionid = q`，分页与既有列表一致。
5. 未命中：200，`orders=[]`，`total=0`。
6. 超长 400；taskAuth 失败 502（不把部分支付命中伪装成完整结果）。
7. 未登录 401；非平台员工 403。

## 非目标

- 租户侧列表本增量不增加 `wechat_account`。
- 模糊搜索。
- 向响应下发 openid。

## 业务意图 → 事件对照

**无对应新事件（书面例外）**：只读列表过滤。

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|----------|
| 按微信关联账号查询订单 | — | — | — | 纯查询；无新业务状态 |
