# Implementation Plan — tenant-gitlab-settings-resource-purchase

**设计：** `docs/superpowers/specs/2026-07-18-tenant-gitlab-settings-resource-purchase-design.md`

## Tasks

- [x] Migration `007_tenant_gitlab_resource.sql`
- [x] Domain/应用：`purchaseGitlabResources` + GET/POST handlers
- [x] 路由注册 + OpenAPI
- [x] 红绿单测：Get / Purchase / InsufficientBalance
- [x] 前端 Sidebar 文案「GitLab」
- [x] 前端双区块 + `useGitlabResourcePurchase.js` + 单测
- [x] intents + publish_evidence_exempt（配额事件）
- [x] 架构 v36 三件套 + VERSION_HISTORY
- [x] `runall-lifecycle.sh build`（公网 SPA）
- [x] Review + PR

## 事件契约

- 成功扣费 → 既有 `BILLING_TRANSACTION_CREATED` outbox
- `TenantGitlabResourcePurchased` → 首期豁免，见 `docs/intents/_meta/publish_evidence_exempt.yaml`
