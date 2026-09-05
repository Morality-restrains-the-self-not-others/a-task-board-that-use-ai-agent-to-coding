# 数据迁移与初始化脚本目录规范与防重复机制

## 基本信息

- 版本：2.0.0
- 创建日期：2026-07-25
- 最后修改：2026-07-25
- 维护者：Trae AI 团队

## 背景（为何是元规则）

monorepo 内各服务的迁移脚本（schema migration）和初始数据（seed / bootstrap data）散布在多处：

| 现状 | 问题 |
|---|---|
| Go 服务 `db.go` 内嵌 `CREATE TABLE` + `seedDefaults()` | 与业务代码混杂；防重复检查条件脆弱（如 `name='系统默认进度体系'`，重命名后重启即重复创建 — 实际已触发 bug） |
| Go 服务 `migrations/*.sql` 目录各异 | taskAuth/taskBill/taskReferral 各有 `migrations/` 但路径硬编码在 `migrate.go` 中，无统一入口 |
| Go 服务 `src/*_bootstrap.go` | bootstrap 逻辑与业务 handler 混在同一 package，职责不清 |
| Django `scripts/init/*.py` | 无统一执行追踪，幂等性靠 ORM `get_or_create` 自行保证 |
| taskCredentialService `migrations/` 单文件 | 路径硬编码在 infrastructure 层 |

**根因**: 缺乏「迁移和初始化脚本该放哪里、如何防止重复执行」的元规则，导致各服务自行发明机制，质量参差不齐。

## 规则分类

### 核心规则

#### 1. 目录规范：`dataMigrate/<app>/`

- **描述**：每个服务的 **Schema 迁移脚本**（DDL：CREATE/ALTER TABLE）和 **数据初始化脚本**（DML：seed data / bootstrap data / 系统默认配置）**必须**统一放在 `dataMigrate/<app>/` 目录下。禁止将迁移/初始化逻辑内嵌在 `db.go` 或 `src/` 业务代码中。
- **优先级**：高
- **规则类型**：核心规则

##### 目录结构

```
<repo-root>/
  dataMigrate/                              # monorepo 级 dataMigrate 根目录
    taskProjectService/                     # 按服务名（app）分子目录
      001_schema.sql                        # Schema DDL — 建表
      002_default_progress_system.go        # Data seed — 默认进度体系
      003_default_deliverable_system.go     # Data seed — 默认交付物体系
    taskBill/
      001_schema.sql                        # Schema DDL
      002_seed_default_pricing.sql          # Data seed
      003_seed_recharge_billing_unit.sql
      ...
    taskAuth/
      001_schema.sql                        # Schema DDL
      002_super_admin.sql                   # Data seed
      003_oidc_bootstrap_clients.go         # Bootstrap
      ...
    taskCredentialService/
      001_schema.sql
    scripts/                                # 跨服务初始化脚本（纯 Python，无框架依赖）
      init_system.py                        # 系统初始化编排
      init_tenant.py                        # 租户注册
      repair_users_without_company.py       # 事件重放修复缺公司用户
      create_kafka_topics.py                # Kafka 主题创建
```
```
      

**命名规则**：
- `NNN_<描述>.<ext>` — 三位数字编号前缀确保有序执行
- 编号按服务独立，每个服务从 001 开始
- 扩展名 `.sql`（SQL 迁移/种子）、`.go`（Go bootstrap 逻辑）、`.py`（Python/Django 脚本）、`.sh`（Shell 编排）

**分类**：同一服务下的脚本可混合 schema 和 seed（编号保证顺序），也可在文件名中标记类型（`_schema` / `_seed` / `_bootstrap`）。

##### 存量迁移路径

| 现状位置 | 迁移目标 |
|---|---|
| Go: `taskAuth/migrations/*.sql` | → `dataMigrate/taskAuth/` |
| Go: `taskBill/migrations/*.sql` | → `dataMigrate/taskBill/` |
| Go: `taskReferral/migrations/*.sql` | → `dataMigrate/taskReferral/` |
| Go: `taskCredentialService/migrations/001_create_tables.sql` | → `dataMigrate/taskCredentialService/001_schema.sql` |
| Go: `taskProjectService/src/db.go` `seedDefaults()` | → `dataMigrate/taskProjectService/002_default_progress_system.go` |
| Go: `taskAuth/src/oidc_bootstrap.go` `seedOidcBootstrapClients()` | → `dataMigrate/taskAuth/` |
| Django: `Saas_project/scripts/init/*.py` | → `scripts/` (纯 Python 重写，已移除 Django 依赖；各服务 seed 脚本分发至对应 `dataMigrate/<app>/`) |
| Go: `db.go` 内嵌 `CREATE TABLE IF NOT EXISTS`（taskProjectService/taskTaskService/taskCloudService/taskTenantService/taskAIComment 等） | 提取为 `dataMigrate/<app>/001_schema.sql` |

##### Go 服务集成方式

**业务进程不得在启动时执行迁移**（见 [40_app_process_independent_of_db_migrate.md](./40_app_process_independent_of_db_migrate.md)）。

迁移由 `http://<host>:9999/` → `migrate.sh` → `apply_datamigrate.sh` 执行；可选显式 `go run ./src migrate`（仅 CLI）。

```go
// ❌ 禁止在 openDB / server main 调用
// ✅ 仅 migrate CLI：
if os.Args[1] == "migrate" {
    return runDataMigrateFromDir(dsn, repoRoot)
}
```

##### Go seed/bootstrap 脚本

Go 服务的 `.go` seed 脚本放在 `dataMigrate/<app>/` 下，属于独立 package（如 `package datamigrate_taskauth`），通过注册函数暴露给 main：

```go
// dataMigrate/taskProjectService/002_default_progress_system.go
package datamigrate_taskproject

import "database/sql"

func DefaultProgressSystem_Key() string { return "002_default_progress_system" }

func DefaultProgressSystem_Apply(db *sql.DB) error {
    var id string
    db.QueryRow(`SELECT id FROM progress_systems WHERE is_default=1 LIMIT 1`).Scan(&id)
    if id != "" { return nil }
    // INSERT ...
    return nil
}
```

#### 2. 防重复机制：`data_migrate_log` 追踪表

- **描述**：每个拥有持久化存储的服务 **必须** 维护一张 `data_migrate_log` 表，记录已执行的迁移/初始化步骤。每次脚本执行前先查表，已执行则跳过。
- **优先级**：高
- **规则类型**：核心规则

##### 追踪表 Schema

```sql
CREATE TABLE IF NOT EXISTS data_migrate_log (
    step_key TEXT PRIMARY KEY,              -- 脚本文件名（如 "001_schema.sql"）
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    checksum TEXT DEFAULT ''                -- 可选：脚本内容 hash，用于变更检测
);
```

##### 通用辅助函数

```go
func ensureMigrateStep(db *sql.DB, stepKey string, apply func(*sql.DB) error) error {
    var exists int
    db.QueryRow(`SELECT 1 FROM data_migrate_log WHERE step_key = ?`, stepKey).Scan(&exists)
    if exists == 1 { return nil }
    if err := apply(db); err != nil { return err }
    _, err := db.Exec(`INSERT INTO data_migrate_log (step_key, applied_at) VALUES (?, datetime('now'))`, stepKey)
    return err
}
```

##### 三层防重复保障

| 层级 | 机制 | 适用场景 |
|---|---|---|
| **L1: 追踪表** | `data_migrate_log` 记录已执行的 step_key | 所有脚本的**强制执行追踪** |
| **L2: 稳定存在性检查** | 用不可变的业务条件判断数据是否已存在（如 `WHERE is_default=1`） | 防止追踪表被意外清空后重复插入 |
| **L3: SQL 幂等** | `CREATE TABLE IF NOT EXISTS` / `INSERT OR IGNORE` / unique constraint | 数据库层面的最终防线 |

##### 稳定性检查条件设计原则

**禁止**将用户可修改的字段作为唯一性判断条件：
- ❌ `WHERE name = '系统默认进度体系'` — 名称可被管理员重命名
- ❌ `WHERE display_name = 'xxx'` — 同上
- ✅ `WHERE is_default = 1` — 语义稳定
- ✅ `WHERE is_system = 1` — 系统标记不可变

#### 3. 迁移与初始化统一管理

- **描述**：Schema 迁移（DDL）和数据初始化（DML）统一放入 `dataMigrate/<app>/`，按编号排序执行。不再区分 `migrations/` 和 `dataInit/` 两套目录。
- **优先级**：高
- **规则类型**：核心规则

**原因**：
- 单一目录降低认知负担（"该放哪里？→ `dataMigrate/<app>/`"）
- 编号天然保证 schema 先于 seed 执行（001_schema.sql 在 002_seed_*.sql 之前）
- 统一追踪表 `data_migrate_log` 同时覆盖 schema 和 seed

### 最佳实践

#### 脚本编写规范

1. **每个脚本只做一件事** — 一个文件只建一批相关表，或只初始化一类数据
2. **幂等是硬要求** — SQL 使用 `CREATE TABLE IF NOT EXISTS` / `INSERT OR IGNORE`；Go 使用存在性检查 + 追踪表
3. **编号有序** — schema 在前（如 001-009），seed/bootstrap 在后（如 010+）
4. **失败即停止** — 任一脚本失败，服务启动 fatal
5. **日志透明** — 每步 log `[<service>] dataMigrate: <step_key> (skipped|applied)`

#### 执行顺序

服务启动时：
1. `openDB(dbPath)` — 打开数据库连接
2. `CREATE TABLE IF NOT EXISTS data_migrate_log` — 确保追踪表存在
3. 加载 `dataMigrate/<app>/` 下所有脚本，按文件名排序
4. 逐个检查 `data_migrate_log` → 未执行则执行 → 记录
5. 业务 HTTP server 启动

### 明确禁止

| 禁止 | 说明 |
|---|---|
| 在 `db.go` 中内嵌 `CREATE TABLE` 字符串列表 | 提取为 `dataMigrate/<app>/001_schema.sql` |
| 在 `db.go` 中内嵌 `seedDefaults()` | 提取为 `dataMigrate/<app>/` 独立脚本 |
| 用可变的业务字段作为存在性检查的唯一条件 | 如 `name='xxx'` — 重命名后失效 |
| 多服务共用 `dataMigrate/` 子目录 | 每个服务独立子目录 |
| 跳过 L1 追踪表仅靠 L2/L3 | 三层保障缺一不可 |

### 与已有规则的关系

| 规则 | 关系 |
|---|---|
| [19_single_service_data_ownership.md](./19_single_service_data_ownership.md) | dataMigrate 脚本操作的表必须属于本服务 |
| [20_go_service_first_apis.md](./20_go_service_first_apis.md) | 新建 Go 服务同样遵循本规则 |
| [33_new_service_runall_registration.md](./33_new_service_runall_registration.md) | 新服务的 build.sh / runAll 需确保 dataMigrate 在 server 启动前执行 |

### 存量追踪表迁移

已有 migration 追踪表的服务（taskAuth/taskBill/taskReferral），旧表名与新表名对照：

| 服务 | 旧追踪表 | 新统一名称 |
|---|---|---|
| taskAuth | `taskauth_schema_migrations` | → `data_migrate_log`（新增，旧表保留作为迁移记录） |
| taskBill | `taskbill_schema_migrations` | → `data_migrate_log`（同上） |
| taskReferral | `taskreferral_schema_migrations` | → `data_migrate_log`（同上） |

已执行的旧 migration 不重复执行（新 `data_migrate_log` 初始化时将旧追踪表中已执行的步骤回填）。

## 变更日志

- 2026-08-03：2.1.0 — 执行权收紧：业务进程禁止启动迁移；统一由 9999 / migrate.sh 管理（ADR-0002 / 约束 40）。
- 2026-07-31：3.0.0 — 移除 `dataMigrate/saas/`（Django 退役后遗留）。脚本迁移：admin verify → `dataMigrate/taskAuth/010_verify_super_admin.py`；legal docs → `dataMigrate/taskBill/026_seed_default_legal_documents.py`；跨服务脚本 → `scripts/`（纯 Python，零 Django 依赖）。详见 OPT-20260731-XXX。
- 2026-07-25：2.0.0 — 重命名为 `dataMigrate`，合并 schema migration 与 data seed 到统一目录；追踪表 `data_migrate_log` 覆盖两类脚本。
- 2026-07-25：1.0.0 — 初版（`dataInit` 仅覆盖 data seed）。
