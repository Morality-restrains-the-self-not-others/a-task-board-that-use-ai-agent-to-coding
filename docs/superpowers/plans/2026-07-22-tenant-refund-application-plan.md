# 实施计划：租户退款申请

## 爆炸半径与测试缺口（CRG）

- 触点：taskBill 充值/扣费/支付；billing_bridge proxy；BillingDashboard；SystemAdmin Sidebar/router；api_route_ownership
- 缺口：无既有 refund 测例 → 新建 `refund_*_test.go` + Playwright mock

## 任务

### 后端 taskBill

- [ ] 迁移 `010_refund_freeze_ledger.sql`（frozen_balance、ledger、application、pending 唯一索引、回填）
- [ ] `creditRecharge` 写入 payment_ledger（paypal/wechat）
- [ ] `refund.go`：apply / reject / approve（FIFO + mock 退款）
- [ ] handlers + mountRoutes + OpenAPI public/internal
- [ ] balance 响应含 frozen/available
- [ ] outbox 事件三类 + BILLING_TRANSACTION_CREATED
- [ ] 单测 UT-REFUND-01..04

### Django

- [ ] `refund_views.py` 超管列表/approve/reject → forward_to_taskbill
- [ ] urls + `api_route_ownership.yaml` 登记
- [ ] emit 允许新 event_type（若有白名单）

### 前端

- [ ] BillingDashboardOverview：冻结展示 + 申请按钮/确认
- [ ] composable `useBillingRefund.js`
- [ ] SystemAdminRefundApplications.vue + Sidebar + router
- [ ] Playwright：租户申请 + 超管审批（mock）

### 文档/门禁

- [ ] intents 已写；value-stream 测试点；collectstatic 若改 SPA
- [ ] 行数门禁；go OpenAPI 路由检查
