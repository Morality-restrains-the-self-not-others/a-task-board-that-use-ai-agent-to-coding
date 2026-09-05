# 功能意图：GitLab 磁盘购买须系统管理员手动发货开通

## 背景与目标

GitLab 磁盘下单支付后，系统把行标成 `pending_admin`，但租户 GET 配额会后台调用 `ensureTenantGitlabGroupForRegion`，等于自动发货。产品要求：**用户购买只入账额度，GitLab 组必须由系统管理员点「开通实施」才创建。**

## 范围与边界

- 范围内：`markOrderPaid` 磁盘分支；`regionResourceView`/`listGitlabResourceViews` 禁止对 `pending_admin` ensure；开通成功事件；待开通 GET 列表
- 范围外：任务帖/流量自动发放；管理端赠送仍可 ensure；`disk_expires_at` 改到开通日起算；自动建阿里云节点

## 约束与风险

- 已 `active` 租户加购不得被打回 `pending_admin`
- `pending_node` 仍禁止任何 Admin API
- 无 timer 扫表建组
- 新接口只落 taskBill Go

## 验收标准

1. 首次购买支付后 `disk_gb` 增加、`provisioning_status=pending_admin`，GitLab 无新租户组
2. 多次 GET 配额不调用 ensure（pending_admin）
3. 超管 POST provision 后组存在且 `active`，发出 `GitlabTenantResourceProvisioned`
4. 支付发出 `GitlabDiskFulfillmentQueued`（key=order_id）
5. GET pending-fulfillment 仅 system-admin，列出待开通行
6. 任务帖/流量支付仍立即发放

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 用户支付 GitLab 磁盘 | GitlabDiskFulfillmentQueued | gitlab-disk-fulfillment-queued | markOrderPaid | 审计；列表为 SSOT | — |
| 管理员开通实施成功 | GitlabTenantResourceProvisioned | gitlab-tenant-resource-provisioned | handleAdminProvisionGitlabResource | 审计；无自动消费者 | — |
| GET 配额/待开通列表 | — | — | — | — | 纯查询 |

## 变更记录

- 2026-09-03：相对上一版（hybrid：GET 配额即 ensure 组），购买路径改为只入账、超管手动开通。
