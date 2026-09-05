# DDD — 账单页 GitLab 按区用量

- **日期**: 2026-08-25
- **NFR**: `docs/superpowers/plans/2026-08-25-billing-gitlab-quota-region-usage-nfr-clarification.md`

## 限界上下文

`Billing`（taskBill）已拥有 `TenantGitlabResource` 按区账本。本增量 **不改领域层**，只在 taskFE 增加展示适配。

## 已有实体（只读引用）

- `BillingTenantGitlabResource`：`(tenant_id, region)` → `disk_gb`, `disk_used_bytes`, `traffic_prepaid_gb`, `traffic_used_gb`, `disk_expires_at`
- `GitlabRegion`：`slug`, `name`, `gitlab_web_url`

## 展示值对象（前端）

`GitlabRegionQuotaRow`：从 quotas JSON 映射，不含行为。

## 端口

无新 Repository。查询端口：既有 HTTP GET quotas。

## 领域事件

| 业务意图 | 事件 | 例外 |
|----------|------|------|
| 查看按区配额与已用 | — | 纯查询，无业务事实变更，无跨边界副作用 |

## 架构变更影响

无。不更新 `docs/architecture/` 版本。
