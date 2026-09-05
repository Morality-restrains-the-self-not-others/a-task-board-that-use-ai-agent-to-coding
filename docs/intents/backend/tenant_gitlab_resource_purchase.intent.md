# 功能意图：租户购买系统内建 GitLab 磁盘与流量预购

## 意图

租户通过**资源订单**支付购买系统内建 GitLab 的磁盘 GB 与流量预购 GB。
支付成功后：流量配额立即到账；**磁盘只入账额度并保持 `pending_admin`，GitLab 组须系统管理员开通实施**（见 `gitlab_disk_manual_admin_fulfillment`）。**没有**账户留存现金，
`POST .../gitlab-resources/.../purchase/` 不得扣 `billing_account.balance`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 下单支付成功入账 | BillingTransactionCreated | BILLING_TRANSACTION_CREATED | taskBill `markOrderPaid` outbox/流水 | 既有账单事件消费 | — |
| 支付磁盘待开通 | GitlabDiskFulfillmentQueued | gitlab-disk-fulfillment-queued | markOrderPaid | 审计；超管列表为 SSOT | — |
| 确认购买配额覆盖 | TenantGitlabResourcePurchased | TENANT_GITLAB_RESOURCE_PURCHASED | 资源订单已支付 | 审计/配额同步 | 证据豁免：额度写库以 `markOrderPaid` 为主；建组以开通事件为准 |
| GET 配额视图 | — | — | — | — | 纯查询 |
| 拒绝钱包扣费购买 | — | — | POST `/purchase/` 返回 409 `USE_RESOURCE_ORDER` | 引导 `/billing/orders/create/` | 无状态变更 |

## 验收

- GET 返回当前配额须带 `?region=`；租户视图不含 `balance_points`
- 账单首页 quotas 不含 `balance_points` / `frozen_balance`
- POST `/purchase/` 即使账户有历史 `balance` 也 409，不改余额、不发配额
- 购买入口为 `/billing/orders/create/`；支付成功后流量立即发放，磁盘入账为待开通额度
- 内部上报磁盘：`POST /api/internal/taskbill/report-gitlab-disk-usage/`
- 用户下单 GitLab 磁盘起购 10 GB、目录默认 4.00 元/GB/月（见 `gitlab_disk_catalog_price_and_min_gb`）

## 变更记录

- 2026-09-03：磁盘购买改为超管手动开通，支付不再自动建 GitLab 组。
- 2026-09-03：用户购买磁盘改为起购 10 GB、目录价 4.00 元/GB/月；取消每区域累计 1 GB 封顶。
- 2026-08-22：切断钱包扣费购买，支付只转资源配额。
