# Value Stream: runAll 分组批量重新编译

> Derived from design: `docs/superpowers/specs/2026-06-21-runall-group-rebuild-all-design.md`

## Value Summary

开发者在 runAll Web UI 上点击分组的「全部重新编译」按钮，该组内所有可编译服务按依赖顺序依次重新编译，一次操作替代逐个点击。

## Related Value Streams

- **runall-cascade-lifecycle**：扩展 — 在其已有的分组「启动本组」「关闭本组」按钮旁增加第三个分组级按钮「全部重新编译」，复用相同的 group-actions 渲染路径和 build 执行路径。
- **runall-go-build-command-inference**：复用 — 分组编译的 buildable 判定直接复用 `resolveBuildCommand()` 推断逻辑，无新增判定规则。

## End-to-End Flow

[开发者打开 :9999] → [定位到目标分组 header] → [点击「全部重新编译」] → [前端 POST /api/build-group] → [Runner 按依赖深度升序遍历组内 buildable 服务] → [对每个 eligible 服务调用 BuildService] → [汇总结果返回] → [UI alert 展示 built/failed/skipped/no_build] → [状态自动刷新]

## Value Increments

### Increment 1: 分组编译按钮 + API + 结果反馈 (Thin Slice)
**Value to user:** 一键触发组内所有可编译服务的重新编译，并看到汇总结果  
**Scope:**
- 后端 `POST /api/build-group` 端点 + `Runner.BuildGroup`
- 前端分组 header「全部重新编译」按钮 + `buildGroup()` JS 函数
- 编译结果 alert 汇总（built / failed / skipped / no_build）
- 按钮 disabled 态（组内无 buildable 服务时）
**Depends on:** nothing (独立增量，复用已有 BuildService)

## Fields Impact

| Field | Type | Description |
|-------|------|-------------|
| `runall.runtime.build_group_result_status` | string | 分组编译结果状态 (ok / partial / none) |
| `runall.runtime.build_group_built_count` | int | 成功编译服务数 |
| `runall.runtime.build_group_failed_names` | string[] | 编译失败的服务名列表 |
