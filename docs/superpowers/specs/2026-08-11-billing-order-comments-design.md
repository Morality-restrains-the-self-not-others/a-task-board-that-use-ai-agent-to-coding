# 订单评论：租户 ↔ 系统管理员交流

- **Status:** accepted（goal-mode 自动采纳）
- **Date:** 2026-08-11
- **Iteration:** billing-order-comments-v76
- **Based on:** v75 application-integration
- **Architecture impact:** **是** — ArchiMate application-integration v76
- **ADR:** No-ADR: trivial tech choice, no architectural impact（在既有 taskBill 订单域内扩展留言表与 API，无新服务/新栈）
- **python_api_approval:** not_applicable（全 Go）

---

## 0. 问题

租户在 `/tenant/{tid}/billing/orders/` 查看订单，系统管理员在 `/system-admin/order-records/` 查看订单，双方缺少就某一订单的多轮文字沟通渠道。退款申请的 `reason`/`review_note` 仅为单轮审批备注，不可复用为客服线程。

## 1. 决策（锁定）

1. 在 **taskBill** 新建表 `billing_order_comment`（扁平时间线，无嵌套回复）。
2. 租户 API：`GET|POST /api/tenant/{tid}/billing/orders/{orderId}/comments/`。
3. 超管 API：`GET|POST /api/system_admin/orders/{orderId}/comments/`（body/query 含 `tenant_id` 校验归属）。
4. UI：订单展开区嵌入共享组件 `OrderCommentThread.vue`（租户页 + 超管页），不新开路由。
5. 发评论发布 Kafka 事件 `BILLING_ORDER_COMMENT_CREATED`（一期无消费者，审计/后续通知）。
6. 不做：附件、@提及、编辑/删除、已读回执、SSE 实时铃铛（记 OPT）。

## 2. 方案对比

| 方案 | 优点 | 缺点 | 结论 |
|------|------|------|------|
| A. 复用 task_comments | 已有线程 | 域错误、执行语义污染 | 拒绝 |
| B. 扩展退款 reason/review_note | 少表 | 单轮、与退款耦合 | 拒绝 |
| C. 新建 billing_order_comment + 双边 API | 边界清晰、最小 | 新表/新事件 | **采用** |

## 3. API / 数据

### 3.1 DDL

`dataMigrate/taskBill/038_order_comments.sql`：

```sql
CREATE TABLE IF NOT EXISTS billing_order_comment (
  id BIGINT NOT NULL PRIMARY KEY,
  tenant_id BIGINT NOT NULL,
  order_id BIGINT NOT NULL,
  author_user_id VARCHAR(64) NOT NULL,
  author_side VARCHAR(32) NOT NULL,  -- tenant | system_admin
  content TEXT NOT NULL,
  created_at DATETIME(6) NOT NULL,
  INDEX idx_billing_order_comment_order_created (order_id, created_at),
  INDEX idx_billing_order_comment_tenant_created (tenant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

伸缩评估：实体爆炸型（订单×留言），年增量预计 <10 万行 → 单表 + 索引即可，暂不分区。

### 3.2 租户

- `GET`：鉴权 `billing:view`；校验 order∈tenant；返回 `{ comments: [...], total }`（按 created_at ASC，limit≤100）。
- `POST` body `{ "content": "..." }`：鉴权 `billing:manage`（与退款同级写权限）；`author_side=tenant`；content 1～2000 字。

### 3.3 系统管理

- `GET|POST /api/system_admin/orders/{orderId}/comments/`（dash 别名同步）
- 鉴权：`X-Gateway-Auth-Verified=1` + `IsPlatformStaff`
- POST body：`{ "tenant_id": "...", "content": "..." }`；校验 order.tenant_id
- `author_side=system_admin`

### 3.4 响应字段

```json
{
  "id": "...",
  "tenant_id": "...",
  "order_id": "...",
  "author_user_id": "...",
  "author_side": "tenant|system_admin",
  "content": "...",
  "created_at": "..."
}
```

## 4. 前端

- `OrderCommentThread.vue`：列表 + textarea + 发送；props：`mode`（tenant|admin）、`tenantId`、`orderId`。
- `BillingOrders.vue` / `SystemAdminOrderRecords.vue`：展开明细下方嵌入（控制行数 ≤500）。
- 错误展示带 `data-traceId`。

## 5. 事件

| 业务意图 | 事件 | Topic | 发布点 |
|---------|------|-------|--------|
| 订单评论已创建 | BILLING_ORDER_COMMENT_CREATED | billing-order-comment-created | taskBill publishBillingEvent |

Payload：`comment_id, order_id, tenant_id, author_side, author_user_id, created_at, trace_id`（不含正文全文，防 PII 入 MQ）。

## 6. 测试意图（摘要）

1. 租户 manage 可 POST；view-only 可 GET、POST 403。
2. 超管可 POST；非 staff 403。
3. 跨租户 order_id → 404。
4. content 空/超长 → 400。
5. FE：展开后可见线程并可发送（组件单测）。

## 7. 架构交付物

- `docs/architecture/v76-application-integration-20260811-2215-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
