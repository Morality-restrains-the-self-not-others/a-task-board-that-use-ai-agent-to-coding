# Value Stream: 开发环境一键重置全部数据库

> Derived from design: `docs/superpowers/specs/2026-05-31-dev-database-reset-design.md`

## Value Summary

本地开发者通过 runAll 一键停服、清空六库与 Redis/Kafka，并按 `db/<app>/` 脚本重新 migrate 与种子初始化，回到可登录、可跑测试的干净基线（不自动启服）。

## Related Value Streams

- **2026-05-31-centralized-db-directory-value-stream**（extension）：在集中路径之上增加 dev reset 编排
- **2026-05-31-runall-grafana-clear-all-observability-value-stream**（独立）：可观测清空与 DB 重置分开按钮
- **2026-05-31-runall-explicit-lifecycle-commands-value-stream**（依赖）：停服必须走 `stop_command`

Greenfield — 无既有 `platform-dev-database-reset` 流。

## End-to-End Flow

[开发者点击「重置全部数据库（开发）」] → [confirm RESET] → [runAll 停 DB owner 服务] → [删 6×SQLite+wals] → [Redis FLUSHALL + Kafka topics 重建] → [按 order migrate → init.sh] → [JSON 分步结果] → [开发者手动启 platform 组]

## Value Increments

### Increment 1: Registry + 删库 + saas migrate/init（Thin Slice）
**Value to user:** API 可清空 saas 库并恢复 admin/交付物种子  
**Scope:** `registry.yaml` 扩展、`db/saas/*`、domain 服务骨架、`POST /api/dev/reset-databases`（仅 saas 或全库删+migrate saas）  
**Depends on:** nothing

### Increment 2: 六库 migrate/init + Go 服务 migrate 子命令
**Value to user:** 全部 SQLite 按 order 重建 schema + 各 init.sh  
**Scope:** `db/{task-auth,task-bill,git-oauth,email,ai-provider}/*`、`taskAuth|taskBill run.sh migrate`  
**Depends on:** Increment 1

### Increment 3: Redis + Kafka + 停服编排
**Value to user:** 一次操作含 FLUSHALL 与 topic 重建；停服避免 WAL 锁  
**Scope:** `db/_infra/*`、owner→runAll 服务 stop  
**Depends on:** Increment 2

### Increment 4: UI + 安全闸 + 测试 + value-stream.yaml
**Value to user:** 状态页危险按钮、localhost/env 403、单测与 YAML 登记  
**Scope:** `status.html`、`ui_test`、`platform-dev-database-reset` step  
**Depends on:** Increment 3
