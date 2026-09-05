# 实施计划：Fork 确认弹窗副本数量

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-fork-copy-count-design.md`

## 任务

- [x] T1 纯函数 `clampForkCopyCount` + 单测（默认/边界 0、1、99、100、NaN）
- [x] T2 模态数量选择：默认 1、max 99、confirm 带 `copyCount`；更新 `ForkAutoRunConfirmModal.test.js`
- [x] T3 `forkTask`：N 次 POST、`${batch}:${i}`、只 open 第一份、部分失败；新文件 `taskDetailEditing.forkCopyCount.test.js`
- [x] T4 Header + `useTaskDetail` 透传；`createClickGuard`；更新 `TaskDetailPageHeader.test.js`
- [x] T5 Playwright：数量 3 → 3 POST + 同前缀不同键
- [x] T6 意图 / 测试意图 / value-stream wsd

每步 Red → Green。不改 taskTaskService API。
