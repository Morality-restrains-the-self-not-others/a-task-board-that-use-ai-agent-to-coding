# Code Review: runAll 链式关闭混合所有权

**结论:** 通过（无 critical 问题）

## 变更摘要

- 新增 `CascadeStepActorResolver`：级联每步解析有效 session
- `executeLifecyclePlan` / `StopGroupWithActor` 按服务 owner 委托，非 takeover
- 单点 `StopServiceWithActor` 行为不变

## 审查项

| 项 | 结果 |
|----|------|
| 根因覆盖 | PASS — 混合 bootstrap/UI ownership 场景 |
| 安全边界 | PASS — cascade=false 仍 strict |
| 测试 | PASS — 领域 + Runner + UI stop-group |
| 回归 | PASS — `TestStopService_RejectsNonOwnerSession` |

## 备注

- `StopGroupWithActor` 非 owner 现可成功（委托 owner），与链式关闭语义一致；已更新 `ui_test.go`
