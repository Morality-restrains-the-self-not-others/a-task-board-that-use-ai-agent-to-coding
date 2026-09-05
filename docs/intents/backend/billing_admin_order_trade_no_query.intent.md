# 功能意图：管理端按微信支付单号 / 商户订单号查询资源订单

## 意图

平台员工在系统管理订单记录页粘贴微信商户后台导出的「微信支付单号」或「商户订单号」后，无需先选租户即可精确查到对应资源订单。支付入账时必须把这两类凭证完整写入订单与支付台账，避免只能靠带 `wechat:` 前缀的 `payment_ref` 碰运气。

## 角色

- 系统管理员 / 平台员工：跨租户查询
- 支付回调 / 入账路径：写凭证（非查询角色）

## 行为

1. `GET /api/system-admin/orders/` 的 `order_number` 在原有展示订单号、主键、`payment_ref` 之外，等值匹配：
   - 商户订单号 `out_trade_no`（导出列「商户订单号」，如 `WX878981209491800064`）
   - 微信支付单号 `wechat_transaction_id`（导出列「微信支付单号」，如 `4500000359202608221274536815`）
   - 去掉 `wechat:` 前缀后的 `payment_ref`
2. 查询串先 `trim`，再去掉首尾 `` ` `` / `'` / `"`（Excel 导出防科学计数法）。
3. 本地 0 条、管理端跨租户、且输入像微信支付单号（18–32 位纯数字）时，调用微信 Native `QueryOrderById`，用返回的 `out_trade_no` 再查本地，并回写 `wechat_transaction_id`。
4. 支付成功入账（mock / live 回调、pending 入账）写入：
   - `billing_resource_order.out_trade_no`
   - `billing_resource_order.wechat_transaction_id`
   - `billing_payment_ledger.provider_capture_id`（按 `provider_ref` ∈ 商户单号 / `wechat:`+商户单号 / 原始 `payment_ref` 匹配）
5. 未命中：200，`orders=[]`，`total=0`。超长 400。未登录 401；非平台员工 403。
6. 详情 JSON 含 `out_trade_no`、`wechat_transaction_id`。

## 非目标

- 模糊搜索、按金额/时间扫微信账单。
- 租户侧列表本增量不增加微信单号兜底调微信 API。
- 日志打印完整单号或回调原文。

## 业务意图 → 事件对照

**无对应新事件（书面例外）**：查询只读；凭证回写是既有支付成功路径的字段补全，不新增业务状态。

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|----------|
| 按微信账单单号查单 | — | — | — | 纯查询 |
| 支付入账回写凭证 | — | — | — | 沿用既有支付成功事件；本增量只补列 |
