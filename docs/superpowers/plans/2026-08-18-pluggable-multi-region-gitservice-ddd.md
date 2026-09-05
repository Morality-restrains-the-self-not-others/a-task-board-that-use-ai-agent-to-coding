# DDD：可插拔多区域 GitLab

- **日期**: 2026-08-18
- **BC**: Billing（区域商品/配额）、GitHosting（实例）、Identity（OIDC per instance）

## Aggregates

### GitlabRegion（root: slug）
- VO: WebURL, APIBase, Capacity, CloudProvider
- 不变式: `is_active` 才可售；token 非空才可自动开通

### TenantGitlabQuota（root: tenant_id + region）
- 字段: disk/traffic、provisioning_status ∈ {not_purchased, provisioning, active, pending}
- 不变式: 无平台默认 region；购买必须指定 active region

## Domain Services
- `ProvisionTenantRegionGroup(tenant, region)` → 调 GitHosting 端口

## Ports
- `GitlabAdminPort.EnsureTenantGroup(ctx, apiBase, token, tenantID, limitBytes)`

## Domain Events

| 事件 | 载荷要点 |
|------|----------|
| GitlabRegionUpserted | slug, is_active |
| TenantGitlabResourcePurchased | tenant_id, region, disk_gb |
| TenantGitlabRegionProvisioned | tenant_id, region |
| TenantGitlabRegionProvisioningPending | tenant_id, region, error |

## 落地说明
本迭代在既有 `taskBill` 包内演进（非新建 domain 包），端口以函数 `ensureTenantGitlabGroupForRegion` 体现；事件首期结构化日志 + 可选 outbox，与既有 BILLING_TRANSACTION_CREATED 并存。
