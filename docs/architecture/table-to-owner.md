# 表 → 拥有服务（owner）对照表

> 状态: current | 更新: 2026-07-10  
> 机器可读 SSOT：[`db/table_ownership.yaml`](../../db/table_ownership.yaml)  
> 库路径 SSOT：[`db/registry.yaml`](../../db/registry.yaml)  
> 规范：[`.ai/01_project_constraints/19_single_service_data_ownership.md`](../../.ai/01_project_constraints/19_single_service_data_ownership.md)

## 原则

- **一库 / 一表 ↔ 唯一 owner 服务**（读与写均适用）。
- 他服务需要数据时：经 owner **服务转发**（API/RPC/领域事件），或 **迁表** 到正确 owner。
- 禁止共享连接串、跨服务复用 Model/DAO、旁路 SQL。

## 库级 owner（摘要）

> 存储已全部 MySQL 化（`db/registry.yaml` 全部 `driver: mysql`；`path` 字段为 legacy SQLite 路径，仅用于清理）。下表 database_key / MySQL 库名 / owner 为运行事实。

| database_key | MySQL 库名 | owner 服务 |
|---|---|---|
| ~~`saas`~~ | ~~`db/saas/saas.sqlite3`~~（Django v57 退役，已删除） | ~~`saas-backend`~~ |
| `task-auth` | `task_auth` | `task-auth` |
| `task-bill` | `task_bill` | `task-bill` |
| `git-oauth` | `git_oauth` | `git-oauth` |
| `ai-provider` | `ai_provider` | `ai-provider` |
| `container` | `container` | `task-credential-service` |
| `task-tenant` | `task_tenant` | `task-tenant-service` |
| `task-project` | `task_project` | `task-project-service` |
| `task-task` | `task_task` | `task-task-service` |
| `task-cloud` | `task_cloud` | `task-cloud-service` |
| `task-ai-comment` | `task_ai_comment` | `task-ai-comment` |
| `task-budget` | `task_budget` | `task-cloud-service` |

表级清单以 `db/table_ownership.yaml` 的 `databases.*.tables` 为准；新增表时须同步该文件并写明 owner。

## 维护流程

1. 新表 / 新库：在 `table_ownership.yaml` 登记 `database_key`、`path`、`owner`、`tables`；若路径走 registry，同步 `db/registry.yaml`。
2. 变更 owner：先完成迁表或切断旧直连，再改 YAML 与本文。
3. 发现跨服务直连：优先整改；短期无法改完的写入 `known_cross_service_access`（须有 `reason` + `tracking`），不得静默忽略。

## CI

```bash
python3 db/scripts/ci/check_single_service_db_ownership.py
```

- 同一生产库路径被 **多个非 owner 服务** 引用且未在白名单 → **失败**。
- 白名单内存量债务 → **警告**（stderr），不阻断；整改后删除白名单项。
- 接入：`runAll/scripts/ci/check_ddd_bdd_compliance.py` 根包装器；独立 GHA：[`.github/workflows/db-ownership.yml`](../../.github/workflows/db-ownership.yml)（扫描 + smoke）。
- 全栈 live smoke（loopback）：[`scripts/smoke/internal-apis-live.sh`](../../scripts/smoke/internal-apis-live.sh)
  - runAll 一键：UI「Internal API smoke」→ `POST /api/smoke/internal-apis/verify`；`start-all` 成功后自动软跑
  - Nightly：[`.github/workflows/internal-apis-live-smoke.yml`](../../.github/workflows/internal-apis-live-smoke.yml)（`optional` 默认；`INTERNAL_API_SMOKE_REQUIRE=true` 时硬校验远端）

相关：接口 / 路由前缀 → 服务见 [`api-route-to-owner.md`](./api-route-to-owner.md)。

## Known debt {#known-debt}

当前 **无** 登记的跨服务直连债务（`known_cross_service_access: []`）。新发现的违规须先整改或登记白名单并附 tracking。

**cloud 事件 / IAM 迁表（已完成）**：`cloud_server_events`、`access_key_iam_associations` 已迁入 `task_cloud.db`（owner：`task-cloud-service`）。详见 [`cloud_domain_tables_task_cloud_migration.intent.md`](../intents/backend/cloud_domain_tables_task_cloud_migration.intent.md)、[`cloud_domain_tables_task_cloud_DEPLOY.md`](../intents/backend/cloud_domain_tables_task_cloud_DEPLOY.md)、架构 [`v21-application-integration-20260714-0130-claude`](./v21-application-integration-20260714-0130-claude.puml)。

## 变更记录

| 日期 | 说明 |
|---|---|
| 2026-08-24 | 库级表 SQLite 路径列 → MySQL 库名（`db/registry.yaml` 已全部 mysql；docs-cleanup） |
| 2026-08-19 | task-auth 补登记 `auth_kyc_profile` / `auth_kyc_audit_log` / `auth_aml_screening_record` / `auth_kyc_limit_policy`（禁止再用 accounts_kyc_*） |
| 2026-07-14 | django_client persist trace 清理；部署清单 DEPLOY.md；ArchiMate v21 三类伴生 |
| 2026-07-14 | `cloud_server_events` / `access_key_iam_associations` 迁入 task_cloud；taskEvents 改打 Cloud internal |
| 2026-07-14 | live smoke 挂入 runAll UI/`start-all` 后软检 + nightly workflow；Slice A/B 门禁复核仍 deferred |
| 2026-07-14 | 评估暂缓 `cloud_cloudserverevent` 迁入 taskCloud；smoke 断言显式要求 `0 known-debt warnings`；新增 internal API live smoke |
| 2026-07-14 | Go SSOT 库（task-project/task-task/task-cloud/task-ai-comment）写入 `db/registry.yaml`；smoke 独立 GHA `.github/workflows/db-ownership.yml`；known-debt 清零 |
| 2026-07-14 | 整改：`task-cloud-service`→`saas` 改为仅 HTTP（tenant-member / git-identities / feature-params store / budget-permissions）；移出 known-debt |
| 2026-07-14 | 整改：`task-events`→`saas` 改为 saas-backend internal intent HTTP；移出 known-debt |
| 2026-07-14 | 整改：`task-auth`→`saas`（django_session）改为仅 HTTP（`POST /api/internal/session/resolve/`）；移出 known-debt |
| 2026-07-13 | 整改：`saas-backend`→`task-budget` 改为仅 HTTP（workspace/task budget config + usage list）；移出 known-debt |
| 2026-07-13 | 整改：`task-credential-service` 业务只读改为 HTTP（task-task container-snapshot + saas git-identities lookup）；移出 known-debt |
| 2026-07-13 | 整改：`saas-backend`→`task-bill` 改为仅 HTTP（pricing/referral internal API）；移出 known-debt |
| 2026-07-10 | 初版：对照表 + CI 扫描器；登记已知跨服务直连债务 |
