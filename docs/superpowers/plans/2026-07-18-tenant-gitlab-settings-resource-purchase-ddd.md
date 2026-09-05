# DDD — tenant-gitlab-settings-resource-purchase

## 限界上下文

- **Billing（taskBill）**：配额购买、锁价扣费、交易账本。
- **Git OAuth（taskGitOauth）**：自建 GitLab OAuth 连接（既有，本迭代不改模型）。

## 聚合

| 聚合 | 根 | 不变量 |
|------|-----|--------|
| TenantGitlabResource | tenant_id | disk_gb≥0, traffic_prepaid_gb≥0；购买成功才覆盖 |
| BillingAccount | account_id | balance≥0；锁价字段只读于购买路径 |

## 领域事件

| 事件 | 触发 | MQ |
|------|------|-----|
| BillingTransactionCreated | 扣费成功 | 既有 outbox |
| TenantGitlabResourcePurchased | 配额覆盖成功 | 首期豁免 |

## 架构变更影响

- 新增 DataObject `billing_tenant_gitlab_resource`（taskBill DB）
- 新增 Application Interface：gitlab-resources GET/POST
- 前端 View 重构为双区块；无新进程
