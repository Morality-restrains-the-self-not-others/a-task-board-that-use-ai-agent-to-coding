# 订单评论 — 实施计划

## Slice 1 — DDL + domain + 租户 API（TDD）
- [ ] `038_order_comments.sql`
- [ ] `order_comments.go` list/create
- [ ] `order_comments_test.go` Red→Green
- [ ] 路由 `/comments/` + OpenAPI

## Slice 2 — 超管 API + 事件
- [ ] admin GET/POST handlers
- [ ] `BILLING_ORDER_COMMENT_CREATED` 映射与发布
- [ ] 单测

## Slice 3 — FE 组件
- [ ] `OrderCommentThread.vue` + test
- [ ] 嵌入 BillingOrders / SystemAdminOrderRecords（控行数）

## Slice 4 — 意图/价值流/登记精准重启
- [ ] intents INDEX、value-stream 测试点
- [ ] register precise restart taskBill taskFE
