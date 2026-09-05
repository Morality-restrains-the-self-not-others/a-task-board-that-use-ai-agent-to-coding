# 设计：管理员订单展开展示分账对象与金额

- **日期**: 2026-08-22
- **入口**: `https://www.daydaymoney.com/system-admin/order-records` 订单记录 Tab 展开行（`OrderExpandDetail`）
- **架构版本**: v96 target
- **ADR**: [ADR-0030](../../adr/0030-admin-only-order-profit-sharing.md)

## 背景

管理员后台展开订单后可见行项与任务帖消耗，但看不到该笔订单**分账给谁、分了多少**。底层已有微信分账落库：`taskBill` 在支付完成时 `markOrderForProfitSharing` 写入 `billing_profit_sharing`（接收方 `referrer_user_id`、金额 `commission_yuan_cents`、状态）。缺口是**管理端只读展示**，且租户端与管理员共用 `OrderExpandDetail`、管理员展开目前打的是租户 GET，若把分账挂到租户详情会泄漏资金分配。

用户输入无 `data-traceId`。CRG：`code-review-graph update --brief` 增量 9 文件、0 节点、风险 0.00。`codegraph query handleGetOrder` 指向 `taskBill/src/handlers_orders.go`。

## 根据当前架构的理解

- 企业景观 current：v95 推荐分账资格取消与审计（已 shipped）
- 应用层：taskFE、taskBill、taskReferral；分账执行已在 taskBill + 微信 profitsharing
- 数据：`billing_profit_sharing` 属 taskBill / `task_bill` utf8mb4
- 本增量不改分账发起时机，只补**管理员可见的订单级分账快照读取**

## 方案（已采纳）

1. **SSOT**：`billing_profit_sharing`。不新建表、不把计提表 `billing_referral_commission_accrual` 当作分账展示源。
2. **新管理员只读详情**：`GET /api/system-admin/orders/{order_id}/`（underscore 别名同步）。鉴权与列表相同：`X-Gateway-Auth-Verified=1` + `authz.IsPlatformStaff`。用已有 `loadOrderByID`（ADR-0018：管理端允许仅主键加载）。响应 = 既有 `orderJSON`（items / buyer_note / resource_consumption）+ **始终存在**的 `profit_sharing` 数组。
3. **租户 GET 禁止该键**：`GET /api/tenant/{tenant_id}/billing/orders/{order_id}/` 的 `orderJSON` **不得**出现 `profit_sharing`（连空数组也不返回），避免租户探测「是否有推荐分账」。
4. **管理员展开改打管理员详情**：`useAdminOrderRowExpand` 改为请求 `/api/system-admin/orders/{id}/`，不再用租户 GET 拉展开。租户 `BillingOrders.vue` 继续租户 GET，且不向 `OrderExpandDetail` 传入 `profitSharing`。
5. **UI**：`OrderExpandDetail` 增加可选 prop `profitSharing`（`null` = 不渲染整块；`[]` = 「本订单无分账记录」；有行则表：接收方用户 ID、金额元、状态）。不展示 `referrer_openid` / 微信分账单号（支付标识最小化）。
6. **APISIX**：已有 `/api/system-admin/orders/*`，无需新路由。
7. **无新 MQ 事件**：记录仍在支付成功路径；本增量纯查询。

### 管理员 `profit_sharing[]` 契约

```json
{
  "receiver_user_id": "string",
  "amount_yuan": "0.55",
  "amount_yuan_cents": 55,
  "status": "pending|processing|finished|failed",
  "settle_after": "RFC3339 or empty",
  "settled_at": "RFC3339 or empty",
  "fail_reason": ""
}
```

金额来自 `commission_yuan_cents`；接收方为 `referrer_user_id`。多行按 `id` 升序（一单多接收方预留）。

## 备选（否决）

| 方案 | 否决原因 |
|------|----------|
| 在租户 GET 加字段、前端隐藏 | 响应体即泄漏；租户可直接调 API |
| `?include=profit_sharing` 挂在租户 GET | 只要租户带参即可读；鉴权边界不清 |
| 仅前端写死「不显示」 | 管理员仍走租户 GET，无法安全下发分账 |

## 非目标

- 不改微信分账比例、延迟天数、timer
- 不回填历史缺记录订单（记 OPT）
- 不在管理端展示 openid
- 不新增 Python/Django 接口
- 不把分账展示给租户「推荐收益」页（已有独立绩效接口）

## 权限

| 路径 | 角色 | 范围 |
|------|------|------|
| GET `/api/system-admin/orders/{order_id}/` | 平台员工 | 跨租户按订单主键 |
| GET `/api/tenant/{tid}/billing/orders/{id}/` | 该租户成员 | 仅本租户；无分账字段 |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 管理员查看订单分账 | — | — | — | 纯查询；写入已由支付完成 `markOrderForProfitSharing` 完成 |
| 租户查看订单详情 | — | — | — | 纯查询；不得附加分账 |

## 领域概念（供 DDD）

- Bounded context：Billing（taskBill）
- Entity：`ProfitSharingRecord`（已有），按 `order_id` 查询
- 新端口：管理员订单详情读模型，含 `ProfitSharingSnapshot`
- 不新增聚合写边界

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：9 files, 0 nodes, risk 0.00
- `codegraph query handleGetOrder`：`taskBill/src/handlers_orders.go:182`
- 影响面：`orderJSON` 被租户 GET 与（拟）管理员详情共用；分账必须在管理员包装层附加，禁止改 `orderJSON`

## 架构交付物

- `docs/architecture/v96-enterprise-landscape-20260822-1405-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
- `docs/architecture/v96-application-integration-20260822-1405-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
