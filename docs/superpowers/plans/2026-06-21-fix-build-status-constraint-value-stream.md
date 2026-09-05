# Value Stream: 修复编译(build)受运行状态限制

> Derived from design: `docs/design/fix-build-status-constraint.md`

## Value Summary

开发者点击 runAll Web UI 任意服务的「编译」按钮时，编译操作不受服务当前运行时状态（pending/starting/retrying/skipped/restarting）限制，仅防止同一服务并发编译。

## Related Value Streams

- **runall-group-build-all** (`value-stream.yaml`): **修改** — 此 fix 放宽 `BuildService` 和 `BuildGroup` 的状态门，使 `build_group_skipped_names` 字段语义从"非 terminal 状态跳过"变为"仅 building 并发冲突时跳过"。
- **runall-go-build-command-inference** (2026-05-31): 无关 — 该流关注 build 命令推断，不涉及状态门。

## End-to-End Flow

[开发者点击服务行「编译」或分组「全部重新编译」] → [前端 POST /api/build 或 /api/build-group] → [Runner.BuildService：CAS 状态检查（排除 building）] → [执行构建脚本] → [日志写入 buffer] → [状态恢复为编译前状态]

## Value Increments

### Increment 1: 移除编译状态限制 (Thin Slice)
**Value to user:** 任何状态的服务均可编译（仅 building 状态拒绝并发编译）  
**Scope:**
- `runner.go:BuildService` — CAS 扩展为全部 8 种非 building 状态
- `domain/build_group_result_value_object.go:IsTerminalBuildStatus` — `status != ServiceStatusBuilding`
- 更新对应测试用例
**Depends on:** nothing（直接修改现有逻辑）

## Fields Impact

| Field | Change | Description |
|-------|--------|-------------|
| `runall.runtime.build_group_skipped_names` | 语义变更 | 旧: "因状态不允许而跳过" → 新: "仅在 building 并发冲突时跳过" |
