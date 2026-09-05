# 订单评论 — DDD

## 限界上下文

**Billing（taskBill）** — 订单是聚合根；评论是订单下的实体集合。

## 聚合

- `ResourceOrder`（既有）
- `OrderComment`（新）：属于 Order，不跨订单移动

## 领域事件

- `BillingOrderCommentCreated` → `BILLING_ORDER_COMMENT_CREATED`

## 不变量

1. comment.tenant_id == order.tenant_id
2. author_side ∈ {tenant, system_admin}
3. content 非空且 ≤2000 rune
4. 评论不可变（一期无 update/delete）
