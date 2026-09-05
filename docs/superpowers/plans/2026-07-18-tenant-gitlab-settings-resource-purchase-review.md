# Review — tenant-gitlab-settings-resource-purchase

**对照计划：** `2026-07-18-tenant-gitlab-settings-resource-purchase-plan.md`

## 结论：通过（可 Ship）

| 检查项 | 结果 |
|--------|------|
| 侧栏「GitLab」 | ✅ Sidebar.vue |
| 页面双区块 | ✅ builtin + self-hosted |
| GET/POST API + 迁移 | ✅ taskBill |
| 整单新购扣费 | ✅ 单测 10*5+5*4=70 |
| 余额不足不改配额 | ✅ |
| OpenAPI | ✅ |
| data-traceId | ✅ 资源/连接错误节点 |
| Intent→Event | ✅ BILLING_TRANSACTION_CREATED；TenantGitlabResourcePurchased 豁免 |
| 日志 | ✅ LogForwardStage + slog 小写 level |
| 公网 SPA build+collectstatic | ✅ |
| 无关 TaskDetail 改动 | ⚠️ 未纳入本 PR |

## Log Audit

- 购买成功：`gitlab_resource_purchased` forward_stage
- 失败：error/warn 结构化日志，无密钥

## Intent→Event Audit

- 扣费成功 → outbox `BILLING_TRANSACTION_CREATED`
- `TenantGitlabResourcePurchased` → publish_evidence_exempt（首期）
