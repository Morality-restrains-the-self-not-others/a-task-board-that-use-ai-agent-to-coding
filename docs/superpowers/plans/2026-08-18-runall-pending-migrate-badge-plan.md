# 实施计划 — 9999 未 migrate 标注

- **日期**: 2026-08-18
- **设计**: `docs/superpowers/specs/2026-08-18-runall-pending-migrate-badge-design.md`

## Tasks

- [x] domain `DiffStepKeys` + `BuildMigratePendingReport` + 单测（Red→Green）
- [x] `RegisteredDatabase.Database` 从 registry 带上
- [x] infrastructure：解析 dataMigrate 目录、列 *.sql、mysql CLI 读 step_key
- [x] `GET /api/dev/migrate-status` + 15s 缓存 + 结构化日志
- [x] UI：徽章、琥珀 class、首屏/init/clear 刷新；不进 2s `refresh()`
- [x] ui_test / status_page 锚点
- [x] 重建 bin/runAll 并热替换 9999

## Intent → Event

对照 `docs/intents/platform/runall_pending_migrate_badge.intent.md`：纯查询例外，无 publish 任务。
