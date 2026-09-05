# Value Stream: 集中式 SQLite 目录 `db/`

> Derived from design: `docs/superpowers/specs/2026-05-31-centralized-db-directory-design.md`

## Value Summary

开发者与运维可在 monorepo 根 `db/` 一处查看、备份、迁移全部本地 SQLite，无需在各服务目录间查找。

## Related Value Streams

Greenfield — 纯基础设施路径归口，不修改 `value-stream.yaml` 业务字段。与 `task-auth`/`task-bill` 拆库设计互补（物理位置变更，逻辑库名不变）。

## End-to-End Flow

[开发者 clone] → [registry.yaml 定义路径] → [各服务 loader 解析] → [SQLite 读写正常] → [备份 tar db/]

## Value Increments

### Increment 1: Registry + Loader + SaaS 主库（Thin Slice）
**Value to user:** Django 主站与新 loader 可用，验证 registry 契约。
**Scope:** `db/registry.yaml`、`db/load/loader.py`、`paths_loader.db_sqlite_path()`、测试。
**Depends on:** nothing

### Increment 2: Go 微服务 + Django 多库路由
**Value to user:** taskAuth/taskBill 与 Django router 指向 `db/`。
**Scope:** `db/load/registry.go`、taskAuth/taskBill config、settings `_taskauth_sqlite_path` / `_taskbill_sqlite_path`。
**Depends on:** Increment 1

### Increment 3: 子项目 + 迁移脚本
**Value to user:** gitOauth、email、ai-provider 归口；旧库一键搬迁。
**Scope:** 各 settings.py、`migrate-existing.sh`、README、gitignore。
**Depends on:** Increment 2

### Increment 4: 配置清理
**Value to user:** 移除 `databasePath` / `DB_SQLITE_PATH` 双源。
**Scope:** `port_config.json`、`paths.conf`、脚本默认路径、文档。
**Depends on:** Increment 3
