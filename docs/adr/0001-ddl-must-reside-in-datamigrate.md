# ADR-0001: 所有 DDL 必须放置在 dataMigrate 目录中

- **Status:** accepted
- **Date:** 2026-07-25 (回溯补写 2026-08-02)
- **Author:** Trae AI 团队
- **Deciders:** Trae AI 团队

---

## Context

monorepo 内各服务的数据定义语言（DDL：CREATE TABLE、ALTER TABLE、CREATE INDEX）和初始数据（seed/bootstrap data）散布在多处：

| 现状 (2026-07) | 问题 |
|---|---|
| Go 服务 `db.go` 内嵌 `CREATE TABLE IF NOT EXISTS` 字符串列表 | 与业务代码混杂，审查 DDL 变更需要 grep 全仓库 |
| Go 服务 `migrations/*.sql` 目录各异 | taskAuth/taskBill/taskReferral 各有 `migrations/` 但路径硬编码，无统一入口 |
| Go 服务 `src/*_bootstrap.go` | bootstrap 逻辑与业务 handler 混在同一 package |
| Django `scripts/init/*.py` | 无统一执行追踪，幂等性靠 ORM `get_or_create` 自行保证 |
| `ensureFeatureParamsSchema()` 等"安全网"函数 | 与 `dataMigrate/` 中的 SQL 文件双写，维护负担倍增 |

**根因**: 缺乏「DDL 和初始数据必须放在哪里、如何防止重复执行」的硬约束，各服务自行发明机制，导致：
- 新人不知道该在哪写 DDL
- 数据库初始化与服务启动耦合（必须先启动服务才能建表）
- `runAll`「初始化全部数据库」功能只覆盖部分服务（依赖启动时自动迁移的无法覆盖）

## Decision

**We will** 将所有数据库 DDL 和初始数据（seed/bootstrap）统一放置在 `dataMigrate/<service>/` 目录中，通过以下机制强制执行：

### 1. 目录规范
- 每个服务的 DDL/seed 文件放在 `dataMigrate/<service>/` 下
- 命名：`NNN_description.ext`（三位数字编号 + 描述 + 扩展名）
- 编号按服务独立，从 `001` 开始
- 扩展名：`.sql`（SQL）、`.go`（Go bootstrap）、`.py`（Python 脚本）

### 2. 执行机制

> **2026-08-03 起由 [ADR-0002](./0002-app-process-independent-of-db-migrate.md) 收紧：**  
> 业务进程启动 **不再** 调用 `runDataMigrate()`。执行权归 `http://<host>:9999/` 初始化数据库 / `migrate.sh` / 显式 migrate CLI。

- SQL 文件仍按文件名排序，经 `apply_datamigrate.sh` 或 migrate CLI 幂等执行
- 使用 `data_migrate_log` 追踪表记录已执行步骤

### 3. 三层防重复保障
| 层级 | 机制 | 说明 |
|---|---|---|
| L1 | `data_migrate_log` 追踪表 | 记录已执行的 step_key |
| L2 | 稳定存在性检查 | 用不可变业务条件（`is_default=1`），禁止用可变字段（`name='xxx'`） |
| L3 | SQL 幂等 | `CREATE TABLE IF NOT EXISTS` / `INSERT OR IGNORE` |

### 4. CI 强制执行
- 静态检查脚本 `dataMigrate/check_inline_ddl.sh` 扫描 Go 文件中内嵌的 `CREATE TABLE` 语句
- 集成在 `.github/workflows/repo-quality-gates.yml`，CI 失败时阻断合并
- 唯一豁免：`data_migrate_log` 追踪表本身（它是迁移机制本身）
- 已知豁免（已注册的 Go bootstrap 步骤、表重建临时表、测试夹具）

### 5. 明确禁止

| 禁止 | 替代方案 |
|---|---|
| 在 `db.go` 内嵌 `CREATE TABLE` 字符串列表 | 提取为 `dataMigrate/<app>/001_schema.sql` |
| 在 `db.go` 内嵌 `seedDefaults()` | 提取为独立脚本 |
| 在 `runMigrations()` 内嵌 ALTER TABLE | 新增 `dataMigrate/<app>/NNN_add_column.sql` |
| 用可变业务字段作存在性检查唯一条件 | 用 `is_default=1`、`is_system=1` 等语义稳定字段 |
| 多服务共用子目录 | 每个服务独立子目录 |
| 跳过 L1 追踪表仅靠 `IF NOT EXISTS` | 三层保障缺一不可 |
| 新增 DDL 首日不放入 dataMigrate（"先内嵌，后续提取"） | 首日即放入 dataMigrate |

## Alternatives Considered

### Alternative 1: 保持现状（DDL 分散在各服务 db.go 中）

- **Pros:** 无需改造，各服务已有的 `ensureXxxSchema()` 工作正常
- **Cons:** 
  - 数据库初始化依赖服务启动（无法离线验证 schema 完整性）
  - DDL 与业务代码混杂，审查困难
  - 新人认知负担高（"该放哪？"）
  - `runAll` 初始化功能无法覆盖
- **Why rejected:** 虽然短期内无需改动，但长期维护成本和一致性风险太高。

### Alternative 2: 使用 ORM 自动迁移（如 Django migrations / GORM AutoMigrate）

- **Pros:** 声明式，自动生成 DDL
- **Cons:** 
  - Go 服务未统一使用 ORM
  - Django 逐步退役，新服务均为 Go
  - ORM 迁移不适合复杂的 seed 数据场景
  - 无法做跨服务的一致性检查
- **Why rejected:** 项目已决定 Go 优先（见 ADR 待补），统一使用纯 SQL + Go bootstrap 模式更符合当前架构方向。

### Alternative 3: 每个服务独立 `migrations/` 目录 + 独立追踪表

- **Pros:** 各服务完全自治
- **Cons:** 
  - 无统一入口，runAll 初始化逻辑需逐个服务适配
  - 追踪表名称不统一（`taskauth_schema_migrations` vs `taskbill_schema_migrations`）
  - 认知负担高
- **Why rejected:** 统一目录 + 统一追踪表名称的收益远超各服务自治的灵活性。

## Consequences

### Positive

- 所有 DDL 变更集中在 `dataMigrate/<service>/` 一目了然
- `runAll`「初始化全部数据库」功能可以覆盖所有服务（不依赖服务启动）
- 数据库 Schema 可离线验证（`dataMigrate/*.sql` 独立于业务代码）
- 新人只需记住一个规则："DDL 放 dataMigrate"
- CI 自动阻断违规，不依赖人工 Code Review 记忆

### Negative / Trade-offs

- 每个服务增加 `dataMigrate/` 子目录维护负担
- Go bootstrap 脚本（`.go` 文件）需要独立 package，与业务代码分离
- `ensureFeatureParamsSchema()` 等"安全网"函数需要维护与 `dataMigrate/` SQL 的双写一致性（目前作为 `repoRoot()` 解析失败的兜底保留）
- 存量 Go 服务内嵌 DDL 迁移需要逐步完成（已跟踪在 `.cursor/rules/database-schema-centralized-migration.mdc` 存量迁移状态表中）

### Mitigations

- `check_inline_ddl.sh` 的豁免白名单允许已知的"安全网"函数存在，防止 CI 误报
- 存量迁移状态表跟踪每个服务的迁移进度，不要求一次性完成
- `db/registry.yaml` + `db/<service>/migrate.sh` 提供统一的脚本入口

## References

- [`.ai/01_project_constraints/34_data_migrate_directory_standard.md`](../../.ai/01_project_constraints/34_data_migrate_directory_standard.md) — 设计文档 v3.0.0
- [`.ai/01_project_constraints/38_database_init_registry_mandate.md`](../../.ai/01_project_constraints/38_database_init_registry_mandate.md) — 数据库注册表规范
- [`.cursor/rules/database-schema-centralized-migration.mdc`](../../.cursor/rules/database-schema-centralized-migration.mdc) — IDE 规则 + 存量迁移状态
- [`dataMigrate/check_inline_ddl.sh`](../../dataMigrate/check_inline_ddl.sh) — CI 静态检查脚本
- [`.github/workflows/repo-quality-gates.yml`](../../.github/workflows/repo-quality-gates.yml) — CI Pipeline 集成
