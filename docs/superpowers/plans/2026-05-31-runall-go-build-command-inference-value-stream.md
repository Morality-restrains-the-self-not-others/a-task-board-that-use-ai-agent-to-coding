# Value Stream: runAll Go 服务编译按钮修复

> Derived from design: `docs/superpowers/specs/2026-05-31-runall-go-build-command-inference-design.md`

## Value Summary

运维在 runAll Web UI 点击 Go 服务的「编译」时，应能真正执行 build，与按钮 `buildable` 状态一致。

## Related Value Streams

- **runall-cascade-lifecycle**（`value-stream.yaml`）：间接 — Go 服务编译后再启停行为一致
- **taskFE-build-command**（2026-05-27）：同类 UI 编译问题，已用 YAML 补齐；本次用 Runner 推断统一 Go 服务

## End-to-End Flow

[用户点击编译] → [UI POST /api/build] → [Runner 解析 build 命令] → [执行 build] → [日志写入 buffer] → [状态恢复]

## Value Increments

### Increment 1: Runner 推断 build（Thin Slice）
**Value to user:** task-auth / task-bill 等无显式 `build_command` 的 Go 服务可编译  
**Scope:** `resolveBuildCommand` 接入 `BuildService` / `restartService` / `runBuild`  
**Depends on:** nothing

### Increment 2: task-events 显式 YAML + 推断收紧
**Value to user:** task-events-* 编译可执行，无误推断  
**Scope:** `runAll.yaml` 补 `build_command`；`extractGoBuildFromRunScript` 跳过含 `$` 行  
**Depends on:** Increment 1

### Increment 3: config.yaml 对齐（可选）
**Value to user:** 默认 `--config` 与根目录 `runAll.yaml` 行为一致  
**Scope:** go-run-container / go-relay 编排同步  
**Depends on:** Increment 1
