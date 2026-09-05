# Review：租户退款申请

- 日期：2026-07-22
- 对照计划：`docs/superpowers/plans/2026-07-22-tenant-refund-application-plan.md`

## 结论

**可交付（无 blocking critical）**。自动修复：`refund.go` 行数 595→拆为 `refund.go` + `refund_approve.go`（≤500）。

## Checklist

| 项 | 结果 |
|----|------|
| 租户申请冻结 + pending 唯一 | ✅ 单测 |
| 驳回解冻 | ✅ 单测 |
| 批准 FIFO + refund 流水 | ✅ 单测（mock provider） |
| 超管 Django 薄代理 | ✅ 代码审查 |
| Vue 账单申请 + 管理审批页 | ✅ 代码审查 |
| OpenAPI / ownership | ✅ OpenAPI 单测；refund 路由已登记 exception |
| Intent→Event | ✅ Kafka topics + djangoEmitEvent |
| Log Audit | ✅ slog level=info/warn |
| 行数门禁 | ✅ 已削文件 |
| CRG | unavailable（MCP 未挂载；CLI 图稀疏） |

## 风险（非阻塞）

1. 支付原路退 MVP 为 mock 成功；live PayPal/微信退款 API 待接真实 SDK（OPT）
2. 历史台账 `remaining_points` 未按消费扣减，资金退款上限可能高于真实未消费付费积分
3. Django route check 仓库内另有无关 baseline 漂移（billing/urls、repoOauth），与本功能无关

## CRG 风险

- graph_status: unavailable
- 人工爆炸半径：taskBill credit/charge/handlers、billing_bridge proxy、BillingDashboard、SystemAdmin*
