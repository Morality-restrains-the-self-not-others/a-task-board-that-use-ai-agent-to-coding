# 订单评论：租户与系统管理员交流

## 意图

在租户订单列表 `/tenant/{tid}/billing/orders/` 与系统管理「订单查看」`/system-admin/order-records/` 的订单展开区，提供同一订单上的多轮文字评论线程，使租户侧（`billing:manage`）与平台员工（`IsPlatformStaff`）可互相留言。

## 验收

1. 租户有 `billing:manage` 可对所属订单 POST 评论；`billing:view` 可 GET
2. 系统管理员可对任意租户订单 GET/POST 评论；非 staff 403
3. 评论持久化于 `billing_order_comment`，按时间升序展示，标注 `author_side`
4. content 空或 >2000 字 → 400；跨租户 order → 404
5. 创建评论发布 `BILLING_ORDER_COMMENT_CREATED`（payload 不含正文）
6. OpenAPI 已登记；架构 v76 制品齐全
7. 前端错误带 `data-traceId`

## 设计文档

- `docs/superpowers/specs/2026-08-11-billing-order-comments-design.md`

## 变更日期

2026-08-11

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 订单评论已创建 | BillingOrderCommentCreated | BILLING_ORDER_COMMENT_CREATED | taskBill publishBillingEvent | 一期无消费者（审计/后续通知） | — |
