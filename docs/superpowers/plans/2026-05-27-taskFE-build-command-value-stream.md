# Value Stream: taskFE 编译 build_command 配置

> Derived from design: `docs/superpowers/specs/2026-05-27-taskFE-build-command-design.md`

## Value Summary

开发者在 runAll 控制台对 taskFE 点击「编译」时，能成功执行 `npm run build`，不再因缺少 build_command 报错。

## End-to-End Flow

[开发者点击编译] → [runAll BuildService] → [读取 build_command] → [npm run build] → [状态恢复 / 构建成功]

## Value Increments

### Increment 1: build_command 配置补齐（Thin Slice）

**Value to user:** 编译按钮可用  
**Scope:** `runAll/config.yaml` + `runAll.yaml` 增加 `build_command: "npm run build"`；回归测试  
**Depends on:** nothing
