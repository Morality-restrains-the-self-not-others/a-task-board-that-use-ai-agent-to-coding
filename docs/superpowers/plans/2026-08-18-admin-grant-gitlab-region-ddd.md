# DDD — 管理端赠送 GitLab 须选区域

- **日期**: 2026-08-18
- **NFR**: `docs/superpowers/plans/2026-08-18-admin-grant-gitlab-region-nfr-clarification.md`

taskBill 现为 `main` 包过程式服务。本次**不**新建 `taskBill/domain/` 包（避免为单字段抽取六边形骨架），模型如下，实现落在既有 `ResourceGrantInput` / `adminGrantResources`。

## 限界上下文

计费 / 租户 GitLab 配额（owner：taskBill）。

## 聚合

**ResourceGrantBatch**（一次管理员提交）

- 实体：`ResourceGrant`（id, tenant_id, resource_type, quantity, expires_at, reason）
- 值对象：`GitlabRegionSlug` — 仅 `gitlab_disk` / `gitlab_traffic` 必填；须解析为启用中的 `GitlabRegion`
- 不变量：GitLab 类型无 slug 则拒绝；配额只更新 `(tenant_id, slug)` 行

**TenantGitlabResource**（已有）

- PK：`(tenant_id, region)`
- 磁盘/流量按区隔离

## 领域服务

`adminGrantResources`：校验 → 加配额 → 记 grant → 记流水 → 记 0 元订单。GitLab 分支必须带 slug。

## 端口

不新增；继续用现有 `*sql.DB`。区域解析复用 `getGitlabRegionBySlug`。

## 领域事件

无新增。既有赠送无 MQ；见意图例外表。

## 幂等

`billing_idempotency_key`，键由调用方提供；与「这一次赠送」同粒度。
