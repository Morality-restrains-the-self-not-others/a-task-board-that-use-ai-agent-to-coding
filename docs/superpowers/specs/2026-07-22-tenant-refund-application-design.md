# 设计：租户退款申请与管理员审批原路退回

- 日期：2026-07-22
- 状态：已采纳（goal-mode 自动决策，跳过确认门）
- 架构版本：v45 🎯 target
- `python_api_approval`: scoped（**仅** system-admin 超管闸门薄代理；业务落 Go taskBill）
- 相关：`docs/intents/tenant-refund-application.intent.md`、微信支付/PayPal 充值

## 1. 问题

租户账单页无法申请退还剩余积分对应金额；无余额冻结与审批流；支付侧无退款 API 与持久化支付台账，无法原路退回。

## 2. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 租户账单页可发起退款申请（租户管理员） | UI + API；非管理员 403 |
| S2 | 申请后冻结全部当前可用积分，进入 `pending` | `balance↓ frozen_balance↑`；消费不可用冻结部分 |
| S3 | 同租户仅允许一笔 pending | 重复申请 409 |
| S4 | 系统管理员在 `/system-admin/refund-applications/` 审批通过/驳回 | is_superuser；列表+操作 |
| S5 | 通过：FIFO 对 `user_recharge_paypal/wechat` 台账原路退款；清空冻结；写 `refund` 流水 | mock/单测；事件 |
| S6 | 驳回：解冻积分恢复可用 | 单测 |
| S7 | 合规：不落卡号；日志脱敏；法务待确认项书面化 | 设计文档 §合规 |
| S8 | 超管可开关「开启退款申请」；关闭后租户页按钮不可见且 POST 403 | taskBill `billing_refund_policy`；balance.`refund_enabled` |

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | taskBill：`frozen_balance` + `billing_refund_application` + `billing_payment_ledger`；租户公网 API；超管经 Django 薄代理调 internal | 与充值同服务，台账与余额同库事务 |
| B | Django 写退款表 | 违反 Go-first / 单表所有权 |
| C | 仅人工线下退款、无冻结 | 不满足「申请即冻结」 |

### 已锁定产品决策

| 决策 | 取值 |
|------|------|
| 申请额度 | **全部当前可用余额**（MVP；不可部分申请） |
| 冻结语义 | `balance -= n; frozen_balance += n`（扣费仍只看 `balance`） |
| 可原路退资金 | 仅 `user_recharge_paypal` / `user_recharge_wechat` 台账 `remaining_points` FIFO |
| 非付费来源积分 | 审批通过时直接核销冻结，不调支付退款 |
| 驳回 | 全额解冻 |
| 租户权限 | `requireTenantAdmin` |
| 管理权限 | `is_superuser`（对齐价格管理） |
| 支付退款 | PayPal：`POST /v2/payments/captures/{id}/refund`（入账写 `provider_capture_id`，缺省时按 order 解析）；微信：`refunddomestic` 原路退；`TASKBILL_REFUND_PROVIDER=mock` / 微信 mock / PayPal 未配置时回退 mock |
| 消费扣台账 | `recordConsumption*` 同步 FIFO 扣减 `billing_payment_ledger.remaining_points` |

## 4. 领域概念

| 概念 | 说明 |
|------|------|
| **RefundApplication** | 租户退款申请：pending / approved / rejected |
| **FrozenBalance** | 账户冻结积分，不可消费 |
| **PaymentLedger** | 付费充值台账：渠道、provider_ref、points、remaining_points、currency |
| **OriginalPathRefund** | 按台账 FIFO 调 PayPal/微信退款 API |

## 5. 数据模型（taskBill SQLite）

```sql
ALTER TABLE billing_account ADD COLUMN frozen_balance INTEGER NOT NULL DEFAULT 0;

CREATE TABLE billing_payment_ledger (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL,
  account_id INTEGER NOT NULL,
  channel TEXT NOT NULL,              -- paypal | wechat
  provider_ref TEXT NOT NULL,         -- order_id / out_trade_no
  provider_capture_id TEXT NOT NULL DEFAULT '',
  points INTEGER NOT NULL,
  remaining_points INTEGER NOT NULL,
  amount_minor INTEGER NOT NULL,      -- 分或最小货币单位（与入账一致：yuan*100）
  currency TEXT NOT NULL,
  billing_transaction_id INTEGER,
  created_at TEXT NOT NULL
);

CREATE TABLE billing_refund_application (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL,
  account_id INTEGER NOT NULL,
  applicant_user_id TEXT NOT NULL,
  frozen_points INTEGER NOT NULL,
  status TEXT NOT NULL,               -- pending|approved|rejected
  reason TEXT NOT NULL DEFAULT '',
  reviewer_user_id TEXT NOT NULL DEFAULT '',
  review_note TEXT NOT NULL DEFAULT '',
  payment_refund_refs TEXT NOT NULL DEFAULT '[]', -- JSON
  reviewed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX uq_refund_pending_tenant
  ON billing_refund_application(tenant_id) WHERE status = 'pending';
```

回填：从 `billing_transaction` 中 `user_recharge_paypal/wechat` 解析 `transaction_id` 前缀写入 ledger（`remaining_points=points`，历史消费不做精确归因，审批时以 `min(frozen, sum(remaining))` 为资金退款上限）。

## 6. API

### 租户（经 billing_bridge proxy → taskBill）

- `GET /api/tenant/{id}/billing/refund-applications/` — 列表（本租户）
- `POST /api/tenant/{id}/billing/refund-applications/` — 申请（body 可选 `reason`）
- `GET /api/tenant/{id}/billing/accounts/balance/` — 响应增补 `frozen_balance`、`available_balance`

### 内部（taskBill）

- `GET /api/internal/taskbill/refund-applications/?status=`
- `POST /api/internal/taskbill/refund-applications/{id}/approve/`
- `POST /api/internal/taskbill/refund-applications/{id}/reject/` body `{note?}`
- `GET|PUT|POST /api/internal/taskbill/refund-policy/` — 平台退款开关 `{enabled}`（默认开启）

### Django 薄代理（超管闸门，登记 ownership）

- `GET /api/system-admin/refund-applications/`
- `POST /api/system-admin/refund-applications/{id}/approve/`
- `POST /api/system-admin/refund-applications/{id}/reject/`
- `GET|PUT /api/system-admin/refund-policy/` — 超管开关（薄代理）

**Python 例外理由**：与 `pricing-packages` 同模式——仅 `is_superuser` 校验 + `forward_to_taskbill`；余额/退款业务全部在 Go。

## 7. 业务意图 → 事件

| 业务意图 | 事件名 | MQ / 契约 | 发布点 |
|---------|--------|-----------|--------|
| 退款申请已提交（已冻结） | BillingRefundApplicationSubmitted | BILLING_REFUND_APPLICATION_SUBMITTED | taskBill outbox → Django emit |
| 退款申请已驳回（已解冻） | BillingRefundApplicationRejected | BILLING_REFUND_APPLICATION_REJECTED | 同上 |
| 退款已批准并完成核销/原路退 | BillingRefundCompleted | BILLING_REFUND_COMPLETED | 同上 |
| 退款流水入账（兼容） | BillingTransactionCreated | BILLING_TRANSACTION_CREATED | 既有路径 |

## 8. 合规假设与待法务确认

- **假设**：B2B SaaS 预付积分；退款为未消费预付的原路退；适用运营地支付机构规则由法务确认。
- **待确认**：退款时限、手续费承担、跨境 PayPal 币种、KYC 是否在退款前强制。
- **实现约束**：不落卡号/证件；日志仅 tenant_id / application_id / 脱敏 provider_ref；测试仅沙箱/mock。

## 9. 架构变更

- 视图：application-integration **v45**（基于 v41 current）
- 交付：`.puml` + `.archimate` + `.mermaid.md` + VERSION_HISTORY

## 🕸️ Code Review Graph 分析

- **graph_status**: unavailable/stale（MCP `code-review-graph` 未挂载；CLI `search creditRecharge` 返回 0 nodes）
- **探测路径**: MCP unavailable → CLI search/status → 文件级调研（explore agent）
- **触点社区 / 关键节点**: `taskBill/src/{credit,charge,paypal_pay,wechat_pay,handlers,tenant_member}.go`；`billing_bridge/{proxy_views,client,pricing_store}.py`；`BillingDashboardOverview.vue`；`SystemAdminGrantPoints.vue` / Sidebar / router
- **爆炸半径（文件/函数）**: `creditRecharge`、`recordConsumptionUsage`、`handleBalance`、PayPal/微信入账、BillingProxy catch-all、system-admin 路由与 ownership YAML
- **与 Archimate 对照**: 扩展计费充值边为「申请→冻结→审批→原路退」
- **风险与约束**: 支付订单历史缺 capture_id；FIFO 与混合来源积分；并发申请/扣费
- **对本次设计的影响**: 必须引入 payment_ledger；冻结从 balance 挪出避免改全量扣费路径

## 10. 非目标

- 部分金额退款申请
- Alipay/Stripe
- 自动无审批退款
- 完整历史消费归因引擎
