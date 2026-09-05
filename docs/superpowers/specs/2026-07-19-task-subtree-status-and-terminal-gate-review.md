# Review：任务子树状态与终态门禁

- 日期：2026-07-19
- 对照计划：`2026-07-19-task-subtree-status-and-terminal-gate-plan.md`

## 结论

**通过（可 ship）** — 无 critical / important 未决项。

## 核对

| 项 | 状态 |
|----|------|
| GET subtree + summary | ✅ Go 测绿 |
| PATCH/switch 门禁 409 | ✅ Go 测绿 |
| 全部 settled 可关闭 | ✅ |
| Vue 面板 + 409/traceId | ✅ Vitest 绿 |
| 意图 → 事件 | ✅ 成功仍 TASK_STATUS_CHANGED；拒绝无事件 |
| Log：gate_blocked | ✅ `event=descendant_gate_blocked` |
| 架构 v41 三类伴生 | ✅ puml/archimate/mermaid |
| Python 新接口 | ✅ 无 |

## Log Audit

- 门禁拒绝：`event=descendant_gate_blocked task_id=… open_count=…`
- 列名解析失败：`event=progress_column_name_resolve_fail`（降级，不 5xx）

## Intent→Event Audit

- 查看子树：纯查询，无事件 ✅
- 终态成功：既有 `TASK_STATUS_CHANGED` ✅
- 门禁拒绝：无状态变更，无事件 ✅
