# db/ — 数据库集中目录

与 `dockerInfra/` 同级，统一管理 monorepo 内全部数据库配置和脚本。

## 配置真源

`registry.yaml` — **所有服务数据库的唯一注册表**。新增服务数据库必须在此注册（见 [.ai/01_project_constraints/38_database_init_registry_mandate.md](../.ai/01_project_constraints/38_database_init_registry_mandate.md)）。

`table_ownership.yaml` — **表 → owner 服务**对照（一库/一表仅一服务直连）；人读摘要见 [`docs/architecture/table-to-owner.md`](../docs/architecture/table-to-owner.md)。CI：`python3 db/scripts/ci/check_single_service_db_ownership.py`。

## 数据库现状（2026-07-27 MySQL 迁移完成）

`registry.yaml` 已注册 **13 个 MySQL 数据库**，全部具有 migrate/init 脚本：

| Key | MySQL 库名 | Owner | 残留 SQLite（清理用） |
|-----|-----------|-------|----------------------|
| `saas` | `saas` | saas-backend | `db/saas/saas.sqlite3` |
| `task-auth` | `task_auth` | task-auth | `db/task-auth/auth.sqlite3` |
| `task-bill` | `task_bill` | task-bill | `db/task-bill/billing.sqlite3` |
| `task-budget` | `task_budget` | task-cloud-service | `db/task_budget/task_budget.db` |
| `git-oauth` | `git_oauth` | git-oauth | `db/git-oauth/git-oauth.sqlite3` |
| `ai-provider` | `ai_provider` | ai-provider | `db/ai-provider/ai-provider.sqlite3` |
| `container` | `container` | task-credential-service | `db/container/tokens.sqlite3` |
| `task-project` | `task_project` | task-project-service | `data/task_project.db` |
| `task-task` | `task_task` | task-task-service | `data/task_task.db` |
| `task-cloud` | `task_cloud` | task-cloud-service | `data/task_cloud.db` |
| `task-ai-comment` | `task_ai_comment` | task-ai-comment | `db/task-ai-comment/task_ai_comment.db` |
| `task-tenant` | `task_tenant` | task-tenant-service | `data/task_tenant.db` |
| `task-referral` | `task_referral` | task-referral | `db/task-referral/referral.sqlite3` |

> 注意：`registry.yaml` 中 `driver: mysql` 是主要连接方式；`path:` 字段仅标记残留 SQLite 文件以便清除。

## 环境变量覆盖

| Key | MySQL DSN 环境变量 | SQLite 路径环境变量 |
|-----|--------------------|---------------------|
| `saas` | `SAAS_MYSQL_DSN` | `SAAS_DATABASE_PATH` |
| `task-auth` | `TASKAUTH_MYSQL_DSN` | `TASKAUTH_DATABASE_PATH` |
| `task-bill` | `TASKBILL_MYSQL_DSN` | `TASKBILL_DATABASE_PATH` |
| `task-budget` | `TASK_BUDGET_MYSQL_DSN` | `TASK_BUDGET_DATABASE_PATH` |
| `git-oauth` | `GITOAUTH_MYSQL_DSN` | `GITOAUTH_DATABASE_PATH` |
| `ai-provider` | `AI_PROVIDER_MYSQL_DSN` | `AI_PROVIDER_DATABASE_PATH` |
| `task-project` | `TASK_PROJECT_MYSQL_DSN` | `TASK_PROJECT_DATABASE_PATH` |
| `task-task` | `TASK_TASK_MYSQL_DSN` | `TASK_TASK_DATABASE_PATH` |
| `task-cloud` | `TASK_CLOUD_MYSQL_DSN` | `TASK_CLOUD_DATABASE_PATH` |
| `task-ai-comment` | `TASK_AI_COMMENT_MYSQL_DSN` | `TASK_AI_COMMENT_DATABASE_PATH` |
| `task-tenant` | `TASK_TENANT_MYSQL_DSN` | `TASK_TENANT_DATABASE_PATH` |
| `task-referral` | `TASK_REFERRAL_MYSQL_DSN` | `TASK_REFERRAL_DATABASE_PATH` |
| `container` | `CONTAINER_MYSQL_DSN` | `CONTAINER_DATABASE_PATH` |

## 开发环境一键重置

由 runAll Web UI 两个按钮触发（仅 localhost 或 `RUNALL_ALLOW_DEV_DB_RESET=1`）：

| 操作 | API |
|------|-----|
| **清空全部数据库（开发）** | `POST /api/dev/clear-databases?confirm=CLEAR_ALL` |
| **初始化全部数据库（开发）** | `POST /api/dev/init-databases?confirm=INIT_ALL` |

**清空流程：** 关闭全部应用（仅保留 docker-redis/kafka）→ 删除残留 SQLite+wals → DROP+CREATE MySQL 数据库 → Redis FLUSHALL → Kafka 重建 topics（不执行 migrate/种子）

**初始化流程：** 按 `registry.order` 执行各 `migrate_script` → `init_script`（不删库；有 owner 服务仍在运行时会拒绝）

典型顺序：先清空，再初始化，最后在 runAll 手动启组。不含 `init-tenant`。

## License

本仓库以 GNU Affero General Public License v3.0 授权，见 [LICENSE](./LICENSE)。不附带 AGPL 义务的专有许可见 [COMMERCIAL.md](./COMMERCIAL.md)。
