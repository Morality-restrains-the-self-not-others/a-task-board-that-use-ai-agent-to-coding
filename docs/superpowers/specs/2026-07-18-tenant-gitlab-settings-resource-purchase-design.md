# 租户 GitLab 设置页：内建资源购买 + 自建连接 — 设计文档

- 日期：2026-07-18
- 状态：已采纳（goal-mode 自动确认）
- 迭代名：`tenant-gitlab-settings-resource-purchase`
- 页面入口：`/tenant/:tenant/settings/gitlab-connection/`（侧栏「设置」→「GitLab」）

## 1. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 左侧菜单文案为「GitLab」（原「GitLab 连接」） | Sidebar 文案 |
| S2 | 页面上半区：购买系统内建 GitLab 磁盘 GB 与流量预购 GB | UI 两区块 |
| S3 | 每次确认调整按**当前账户锁价**对提交的目标量做**整单新购扣费**并更新配额 | taskBill POST + 单测 |
| S4 | 页面下半区：保留自建 GitLab OAuth 连接参数（既有 CRUD） | 既有 API 不变 |
| S5 | Swagger（taskBill OpenAPI）可见新接口 | openapi.yaml |
| S6 | 请求失败错误节点带 `data-traceId` | 前端 |

**本迭代不做**：向 GitLab CE 下发 namespace 存储硬限额（gitService 执法另案）；仅记账与租户侧配额展示。

## 2. 方案对比（已选 A）

| 方案 | 说明 | 结论 |
|------|------|------|
| **A. taskBill 新购配额 API + 页面双区块** | 表存 disk/traffic 配额；调整即按锁价整单扣费 | **采纳** — 价目已在 taskBill；连接已在 taskGitOauth；职责清晰 |
| B. 仅前端展示价目、无扣费 | 不符「直接扣费」 | 否 |
| C. 新建独立 Go 服务 | 过度设计 | 否 |

## 3. 计费语义（关键决策）

- 输入：`disk_gb`（≥0 整数）、`traffic_prepaid_gb`（≥0 整数）。
- 费用：`cost = disk_gb * locked_gitlab_disk_points_per_gb_per_month + traffic_prepaid_gb * locked_gitlab_traffic_points_per_gb`。
- **新购**：每次成功 POST 按上述公式对**本次提交的全量**扣费（非仅增量）；成功后配额覆盖为提交值。
- 降配：不退款；若 `cost>0` 仍按全量新购扣费（与「相当于新购」一致）。若两量均为 0 且 `cost=0`，仅更新配额。
- 余额不足：`402`/`400` + 明确文案（沿用 `InsufficientBalanceError`）。
- 幂等：可选 `idempotency_key`；缺省用雪花 `transaction_id`。

## 4. 数据模型（taskBill 独占）

```sql
CREATE TABLE billing_tenant_gitlab_resource (
  tenant_id INTEGER PRIMARY KEY,
  disk_gb INTEGER NOT NULL DEFAULT 0,
  traffic_prepaid_gb INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);
```

扣费沿用 `billing_transaction` / `billing_usage` / `billing_unit`（`gitlab_disk`、`gitlab_traffic`）。

## 5. API（Go / taskBill，经 billing_bridge 代理）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/tenant/{tid}/billing/gitlab-resources/` | 已登录租户成员 | 当前配额、已用量（disk_used_gb / traffic_used_gb）、锁价、余额 |
| POST | `/api/internal/taskbill/report-gitlab-disk-usage/` | 内部密钥 | 上报 disk_used_bytes（计量侧） |
| POST | `/api/tenant/{tid}/billing/gitlab-resources/purchase/` | 已登录（与换套餐一致；首期不强制 admin header） | body: disk_gb, traffic_prepaid_gb, idempotency_key? |

自建连接 API 不变：`/api/tenant/{tid}/gitlab-oauth-connection/`（taskGitOauth）。

## 6. 前端

- `Sidebar.vue`：菜单「GitLab 连接」→「GitLab」。
- `WorkspaceSettingsGitlabConnection.vue`：
  1. **系统内建 GitLab**：磁盘 GB、流量预购 GB、单价展示、预估积分、确认购买。
  2. **自建 GitLab 连接**：既有表单。
- 拆出 composable `useGitlabResourcePurchase.js`（控行数）。

## 7. 事件

| 意图 | 事件 | 投递 | 说明 |
|------|------|------|------|
| 购买成功 | `BillingTransactionCreated`（既有 outbox） | 已有 `BILLING_TRANSACTION_CREATED` | 复用消费链 |
| 配额更新 | `TenantGitlabResourcePurchased` | 首期证据豁免（HTTP+审计日志）；后续 Kafka | 见 intents |

## 8. 架构影响

- 应用层：taskBill 新增 DataObject `billing_tenant_gitlab_resource` + API。
- 无新进程；见 `docs/architecture/v36-*`。

## 9. 合规

- 积分消费，非第三方收单；不引入卡数据/KYC 变更。
