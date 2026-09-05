# 功能意图：可插拔多区域 gitService

## 意图

平台管理员登记多个独立 GitLab CE 区域；租户必须显式选购至少一个区域，平台不提供静默默认区域。每区域数据隔离。**用户购买 GitLab 磁盘不自动建组**：支付后 `pending_admin`，由系统管理员「开通实施」调用区域 Admin API；失败则保持 `pending_admin` 可重试。GET 配额不得 ensure 组。

现网 `gitlab.${baseDomain}` 登记为可售区域 `tencent-shanghai-5`；上海新实例 `gitlab-tencent-sh-1.${baseDomain}` 登记为 `tencent-sh-1`。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|----------|----------------|------------|--------|--------------|---------|
| 租户选购某区域 GitLab 配额成功 | BillingTransactionCreated | BILLING_TRANSACTION_CREATED | taskBill outbox | 既有账单事件消费 | — |
| 区域配额覆盖 | TenantGitlabResourcePurchased | TENANT_GITLAB_RESOURCE_PURCHASED | taskBill POST | 审计/配额同步 | 证据豁免：首期以 BILLING_TRANSACTION_CREATED outbox + HTTP 配额写库为主 |
| 管理员开通实施 | GitlabTenantResourceProvisioned | gitlab-tenant-resource-provisioned | POST provision | 审计；无自动消费者 | 见 gitlab_disk_manual_admin_fulfillment |
| 管理员登记区域 | — | — | SystemAdmin CRUD | `billing_gitlab_region` | 配置类写库，无跨服务副作用 |
| GET 区域列表 / 配额视图 | — | — | — | — | 纯查询 |

## 验收

- GET/POST GitLab 资源接口缺少 `region` 返回 400，禁止回落到 `tencent-shanghai-5`
- `ensureTenantGitlabGroupForRegion` 使用该行 `gitlab_api_base` + `admin_private_token`
- 前端购买必须选择区域；空区域不发购买请求；磁盘/流量按行选区（见 [tenant_purchase_gitlab_region](tenant_purchase_gitlab_region.intent.md)）
- 管理端赠送 `gitlab_disk` / `gitlab_traffic` 必须选择区域（见 [admin_grant_gitlab_region](admin_grant_gitlab_region.intent.md)）；禁止静默默认区
- seed 含 `tencent-sh-1`（api/web `https://gitlab-tencent-sh-1.${baseDomain}`；token 空待 SystemAdmin 填）
- taskAuth OIDC 第二 client：`gitlab-git-service-tencent-sh-1` **必须出现在** `auth_oidc_client`（不能只写 YAML）。`010_oidc_bootstrap_clients` 每次 `taskAuth migrate` / 9999 必跑幂等 seed，禁止因 `data_migrate_log` 已有 step_key 而跳过新增 client

## 变更记录

- 2026-09-03：购买路径废除 GET 配额 hybrid 自动建组，改为超管手动开通（`gitlab_disk_manual_admin_fulfillment`）。
- 2026-08-18：GitLab SH-1 SSO `unauthorized_client`/`client not found`（trace `661ed44b12cea27653bfa34773e50087`，先前 `7bc6fededff5e2b9777e10343d6d018d`）。conf 已有第二 client，DB 无行——Go seed 被一次性 migrate key 跳过。规格：`docs/superpowers/specs/2026-08-18-oidc-bootstrap-client-seed-skip-design.md`。闸门：`goMigrateStep.Always`。
