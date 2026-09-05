# 实施计划 — 阿里云 GitLab 区域人工建节点

- **日期**: 2026-09-02
- **设计**: `docs/superpowers/specs/2026-09-02-aliyun-gitlab-region-manual-node-design.md`

## 任务

- [x] **T0 迁移** `dataMigrate/taskBill/075_gitlab_region_infra_status.sql`：列 `infra_status` 默认 `ready`；seed 10 条阿里云 `pending_node`。
- [x] **T1 领域门闩** `gitlabRegionAllowsAdminAPI`；pending_node 时 `ensureTenantGitlabGroupForRegion` 与 `regionResourceView` goroutine、admin provision 409。单测 T1/T2。
- [x] **T2 列表 JSON** 租户 GET 含 `infra_status`、`cloud_provider`（仍剥 token）。
- [x] **T3 事件** `billingEventTopics` 注册两事件；`markOrderPaid` 后对 pending_node 发布 Queued（key=order_id）；PUT ready 发布 MarkedReady（key=slug）。单测 T3。
- [x] **T4 Admin PUT** 接受 `infra_status`；非法值 400。单测 T4。
- [x] **T5 FE util** `groupGitlabRegionsByProvider` + 单测。
- [x] **T6 OrderCreate** optgroup、pending_node 提示 `data-testid="order-gitlab-pending-node-hint"`、留言条件。合同测 T5。
- [x] **T7 SystemAdmin** 徽章「待创建节点」；编辑可改 infra_status。
- [x] **T8 意图 INDEX** B-049j；Swagger：PUT 字段 + 列表字段（无新 path 则更新既有 gitlab-regions schema）。
- [x] **T9 事件契约** 更新意图对照表（已在 intent 中）。

每步 Red→Green。不自动 ECS。不 seed OIDC。
