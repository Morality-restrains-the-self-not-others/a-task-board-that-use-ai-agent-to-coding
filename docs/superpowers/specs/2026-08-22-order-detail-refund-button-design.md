# 订单详情支付成功态展示退款入口

- **Status:** accepted（/goal 自动采用）
- **Date:** 2026-08-22
- **Scope:** 租户订单详情页 `/tenant/{tid}/billing/orders/{orderId}/` 支付成功横幅

## Context

支付成功横幅仅提示「资源已发放」，退款入口只在订单列表操作列。用户从详情页无法申请退款，也不知道已消耗配额不会因退款恢复。

既有能力：`POST /api/tenant/{tid}/billing/refund-applications/`（`billing:manage`）、`isBillingOrderRefundable`、`useBillingRefund`、超管审批与原路退。

## Decision

1. **不新增 API / 不改退款金额计算。** 实付金额仍按订单全额进入审批（与现网一致）。「已消耗无法退回」指**资源配额**：获批后收回本单剩余未使用配额；已消耗批次与已创建任务帖不回滚、不删除。
2. 详情页已支付横幅增加「申请退款」按钮（资格与列表相同：微信/PayPal 实付、`refund_enabled`、无 pending/approved）。pending 时显示「退款中」。
3. 横幅与确认弹层均注明已消耗资源无法退回；弹层复用订单 `resource_consumption`。
4. 列表页与详情页共用确认弹层组件。确认按钮 `createClickGuard` + `Idempotency-Key`。服务端：带该头且本单已有活跃申请时返回已有申请（重放），不新增第二条。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 按剩余配额比例退款 | 改变资金语义，需法务/渠道部分退款产品决策；本次需求是展示按钮+资源说明 |
| 详情页链到列表 `?order_id=` | 多一跳，支付成功页仍无入口 |
| 新退款 API | 与现申请/审批状态机重复 |

## Consequences

- 正面：支付成功页可闭环申请；用户在提交前看到消耗不可退回。
- 负面：全额退款 + 保留已消耗任务对平台偏慷慨；比例退款记 OPT。
- 合规：不改 KYC/渠道；日志仍禁卡号；金额仅订单 ID + frozen_points。
