# 应用进程独立于数据库迁移（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-03
- 维护者：Trae AI 团队
- 优先级：高（禁止忽略）
- Cursor：`.cursor/rules/app-process-independent-db-migrate.mdc`（alwaysApply）
- ADR：`docs/adr/0002-app-process-independent-of-db-migrate.md`
- 关联：`34_data_migrate_directory_standard.md`、`38_database_init_registry_mandate.md`、`.cursor/rules/database-schema-centralized-migration.mdc`

## 背景

存量实践中，业务进程在 `openDB()` / `main` 启动路径上调用 `runDataMigrate` / `EnsureSchema`，导致：

1. **Schema 变更与进程生命周期耦合** — 未起服务就无法完整初始化；起服务则隐式跑 DDL，难审计、难回滚编排
2. **与统一入口双轨** — `http://10.2.150.68:9999/`（及任意 host 的 9999）「初始化全部数据库」已通过 `db/registry.yaml` → `migrate.sh` → `apply_datamigrate.sh` 覆盖 `dataMigrate/`，启动再跑一遍是冗余且危险的第二路径
3. **故障面扩大** — 迁移失败会阻断 HTTP 服务启动；DELIMITER/双路径等 SQL 问题直接打成 crash loop

## 核心原则

1. **DDL / seed 唯一存放处**：`dataMigrate/<service>/`
2. **迁移唯一编排入口**：runAll Dev UI `http://<host>:9999/` →「初始化全部数据库」→ `POST /api/dev/init-databases` → `InitAllDatabases()` → 各库 `migrate.sh` / `init.sh`（推荐实现为 `db/scripts/apply_datamigrate.sh`）
3. **业务应用进程禁止执行迁移**：长期运行的 server / worker **不得**在启动或运行路径调用 `runDataMigrate` / `runDataMigrateFromDir` / `EnsureSchema` / 等价逻辑

## 强制要求

### 1. 业务进程

- `openDB` / `OpenDB` / 连接池初始化：**只**建立连接与连接池参数，**禁止**执行 `dataMigrate` SQL
- `main` 在监听 HTTP/RPC 之前：**禁止**调用迁移函数（含 `RunGoDataMigrate` 等 Go seed 步骤）
- **禁止**以「安全网」名义在业务路径 `CREATE TABLE` / `ALTER TABLE`（`data_migrate_log` 本身除外，且仅允许出现在迁移执行器内）

### 2. 允许的迁移执行路径

| 路径 | 允许 | 说明 |
|------|------|------|
| `http://<host>:9999/` 初始化全部数据库 | ✅ | 权威入口（含 `10.2.150.68:9999`） |
| `db/<db>/migrate.sh` → `apply_datamigrate.sh` | ✅ | 9999 与 CI/运维脚本调用 |
| `go run ./src migrate`（或等价 CLI） | ✅ | 仅显式 migrate 子命令；可由 migrate.sh 串联（如 Go seed） |
| 单元/集成测试夹具 | ✅ | 测试库可调用迁移辅助函数 |
| `bootstrap-*` 等一次性运维 CLI | ✅ | 非长期业务进程；应先依赖 migrate 或自行调用迁移 |
| 业务 server 默认 `main` / `openDB` | ❌ | 禁止 |

### 3. 新增 SQL 生效方式

1. 将文件放入 `dataMigrate/<service>/NNN_*.sql`
2. 在 **9999** 执行「初始化全部数据库」（或对该库跑 `migrate.sh`）
3. **再**启动或重启业务进程

**禁止**假设「重启服务即自动迁移」。

### 4. 与既有规范的关系

- **补充并收紧** `34`：目录仍在 `dataMigrate/`；执行权从「启动可跑」改为「仅统一入口 / 显式 migrate CLI」
- **落实** `38`：「禁止仅依赖服务启动时的自动迁移」升级为「禁止服务启动执行迁移」
- **修正** `database-schema-centralized-migration.mdc` 中「启动时必须调用 runDataMigrate」的过时表述

## 验收

```bash
# 业务 openDB / OpenDB 不得内嵌迁移调用
python3 dataMigrate/check_no_startup_migrate.py

# 9999 路径仍注册完整
test -f db/registry.yaml
rg -n 'migrate_script:' db/registry.yaml

# 手工：打开 http://10.2.150.68:9999/ → 初始化全部数据库 → 再启业务进程
```

## 变更日志

- 2026-08-03：1.0.0 — 首版；与 ADR-0002 同步。
