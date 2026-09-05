# Review — 部署机 9999 源码编译重启

- **Date:** 2026-09-02
- **CRG:** unavailable（无 `.code-review-graph/graph.db`）
- **Codegraph:** 未用（索引未作为本切片入口）

## 对照计划

Task 1–4 已落地：`SOURCE_ROOT` 白名单、`preciseRestartFile` 读源码仓、`prepareDeploySourceArtifacts`、BuildAll 部署路径不重启、增量 `install_over`、文档。

## 阻断项

无。未保留旧 `update.sh` 日常路径（脚本仍在 seed 作应急）。无 Logic-Rollback。

## 日志审计

`[deploy-source]` compile/rsync/install start+ok/fail；不打印 `conf-local` 内容。失败走既有 SSE Error 字段。

## 举一反三

`SyncMonorepoConf` 在 BuildAll / BuildGroup / BuildService 三处 `DEPLOY_MODE` 跳过。

## 意图 → 事件

`docs/intents/platform/deploy_9999_source_compile_restart.intent.md` 声明 no-event（ops 控制面）。
