# 订单评论测试意图

## 对应功能意图

`docs/intents/billing-order-comments.intent.md`

## 测试点

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 租户 manage POST 合法 content | 201 + DB 行 + author_side=tenant |
| T2 | 租户 view-only POST | 403 |
| T3 | 租户 GET 本租户订单评论 | 200 升序列表 |
| T4 | 租户访问他租户 orderId | 404 |
| T5 | 超管 POST | 201 + author_side=system_admin |
| T6 | 非 staff 超管路径 | 403 |
| T7 | content 空 / 超长 | 400 |
| T8 | OrderCommentThread 发送调用正确 URL | 组件单测 |
| T9 | 事件映射含 BILLING_ORDER_COMMENT_CREATED | 单测 |

## 实现位置

- `taskBill/src/order_comments_test.go`
- `taskFE/app/src/components/OrderCommentThread.test.js`
