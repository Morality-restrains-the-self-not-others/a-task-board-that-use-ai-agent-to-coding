# v76 — Billing Order Comments (tenant ↔ system admin)

```mermaid
flowchart LR
  BO[BillingOrders MOD] --> OCT[OrderCommentThread NEW]
  SA[SystemAdminOrderRecords MOD] --> OCT
  OCT -->|GET/POST tenant comments| TB[taskBill MOD]
  OCT -->|GET/POST admin comments| TB
  TB --> BOC[(billing_order_comment NEW)]
  TB --> BRO[(billing_resource_order)]
  TB -->|BILLING_ORDER_COMMENT_CREATED| K[Kafka]
```

- **Version:** 76 target
- **Iteration:** billing-order-comments-v76
- **Based on:** v75
- **Decisions:** 扁平时间线；tenant write=`billing:manage`；admin=`IsPlatformStaff`；事件不含正文
