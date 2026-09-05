# 价值流：微信关联账号查单

```
客服拿到微信昵称/绑定手机/openid
  → 打开订单记录页
  → 切到「微信关联账号」Tab
  → 输入并查询
  → taskBill GET wechat_account
  → taskAuth 解析 user_ids
  → taskBill 过滤订单（user_id ∪ pay_openid ∪ pay_unionid）
  → 列表展示（可分享 URL）
```

## 测试点

| ID | 步骤 | 用例 |
|----|------|------|
| VS-WA-1 | 切 Tab | AdminOrderLookupBox 显示微信输入 |
| VS-WA-2 | 空输入 | 不发请求 |
| VS-WA-3 | 昵称/openid/绑定手机 | Auth 解析 + Bill 列单 |
| VS-WA-4 | 仅支付 openid | 仍命中订单 |
| VS-WA-5 | 未命中 | 200 空列表 / 前端空态 |
| VS-WA-6 | 非员工 | 403 |
| VS-WA-7 | 交易单号回归 | order_number 行为不变 |
