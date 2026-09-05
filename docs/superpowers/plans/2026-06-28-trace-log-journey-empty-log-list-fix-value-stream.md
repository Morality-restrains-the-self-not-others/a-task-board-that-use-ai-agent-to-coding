# Value Stream: Trace Log Journey 日志目录对齐修复

> Derived from design: `docs/superpowers/specs/2026-06-28-trace-log-journey-empty-log-list-fix.md`

## Value Summary

修复 Promtail 监控目录与 runAll 日志写入目录不匹配的问题，使 Grafana Trace Log Journey 仪表盘能正常显示跨服务日志。

## Related Value Streams

- **platform-centralized-logging**: **modification** — 修复 Increment 1 (runall-log-tee) 中 `file_root` 解析不一致导致 Promtail 采集不到日志的问题。原流中 runAll tee 到 `/tmp/logs/`，Promtail 监控 `/tmp/ram-work/logs/`，两者不匹配。本修复将两处统一到同一目录。
- **per-service-log-config**: 并存 — 本修复不改变每服务独立日志路径的配置机制。

## End-to-End Flow

[runAll 启动] → [tee 各服务 stdout 到 file_root] → [Promtail tail file_root] → [Loki 存储] → [Grafana 查询] → [开发者看到日志]

**修复前断裂点：** runAll → `/tmp/logs/` | Promtail ← `/tmp/ram-work/logs/` ❌  
**修复后：** runAll → `/tmp/ram-work/logs/` ← Promtail ✅

## Value Increments

### Increment 1: 统一日志目录路径 (Fix)
**Value to user:** Grafana Trace Log Journey / Trace Log Explore 仪表盘日志列表正常显示。  
**Scope:**
- `conf/runAll.yaml`: `logging.file_root` 从 `../logs` 改为 `logs`
- `runAll/scripts/runall-local-promtail.sh`: 加固 `resolve_runall_log_root()` 的相对路径解析
- 重启 runAll + Promtail 使配置生效
**Depends on:** platform-centralized-logging Increment 1 (runall-log-tee)
