# DDD — 阿里云 GitLab 区域人工建节点

- **日期**: 2026-09-02
- **NFR**: `docs/superpowers/plans/2026-09-02-aliyun-gitlab-region-manual-node-nfr-clarification.md`

不新建 `taskBill/domain/` 包。落在既有 `GitlabRegion` / `markOrderPaid` / SystemAdmin PUT。

## 限界上下文

计费 GitLab 区域目录与租户配额（owner：taskBill）。部署配方仍属 infra/gitService（节点就绪后）。

## 聚合

**GitlabRegion**（目录根）

- 属性：slug、cloud_provider、infra_status、api_base、token、is_active、access_mode
- 不变量：`pending_node` ⇒ 禁止 Admin API；`ready` ⇒ 开通允许（仍需 token）
- 命令：`MarkInfraReady`（仅 system-admin）

**TenantGitlabResource** PK `(tenant_id, region)`

- 支付磁盘 → `pending_admin`（既有）
- 开通成功 → `active`

**ResourceOrder** — 不变；GitLab 行 region 可为 pending_node slug。

## 值对象

- `InfraStatus`: `ready` | `pending_node`
- `CloudProvider`: `tencent` | `aliyun` | …

## 领域服务

`ShouldCallGitlabAdminAPI(region) bool` — infra ready 且 api_base、token 非空。

## 事件

| 事件 | 触发 | Key |
|------|------|-----|
| GitlabManualNodeFulfillmentQueued | 支付后存在 pending_node 区 GitLab 行 | order_id |
| GitlabRegionInfraMarkedReady | infra_status 变为 ready | slug |

消费者：审计；无自动建机消费者（禁止为 DLT 注册自动建机）。

## 幂等

与 NFR 表一致。MarkInfraReady 为状态转移。
