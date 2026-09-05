# DDD — 租户购买 GitLab 须按行选区

- **日期**: 2026-08-18
- **NFR**: `docs/superpowers/plans/2026-08-18-tenant-purchase-gitlab-region-nfr-clarification.md`

不新建 `taskBill/domain/` 包。模型落在既有 `createOrder` / `orderItemInput`。

## 限界上下文

计费 / 租户 GitLab 配额（owner：taskBill）。

## 聚合

**ResourceOrder**（一次租户下单）

- 实体：`ResourceOrderItem`（resource_type, quantity, region）
- 值对象：`GitlabRegionSlug` — GitLab 行必填，须解析为启用区
- 不变量：GitLab 行无有效 slug 则拒绝创建

**TenantGitlabResource**（已有，PK `(tenant_id, region)`）— 支付后按行项 region 累加。

## 领域服务

`createOrder`：会员校验 → 计价 → 写 pending 订单。GitLab 分支必须 `getGitlabRegionBySlug`。

## 事件

无新增 MQ。下单成功：`tracelog` `resource_order_created`（含 region）。支付发放：既有 `BILLING_TRANSACTION_CREATED` outbox。例外见意图文档。

## 幂等

创建订单不新增幂等键（保持可建多笔 pending）。支付 `markOrderPaid` 按订单已 paid 短路。
