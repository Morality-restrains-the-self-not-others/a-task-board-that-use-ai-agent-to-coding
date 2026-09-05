# Implementation Plan: Fix Code Review Findings (runAll)

> 输入: 设计文档 `docs/superpowers/specs/2026-06-23-fix-review-findings-design.md`

## Tasks

### F1: 异步化 handleBuildAllAction

- [ ] **ui.go** — 将 `handleBuildAllAction` 改为 `runLifecycleActionAsync` 异步模式，参照 `handleStopAllAction`；删除 `r.Context()` 依赖，使用 `context.Background()`
- [ ] **ui.go** — 移除 handler 中直接调用 `runner.BuildAll(r.Context())` 的同步逻辑

### F2: PlanStopAll 空列表保护

- [ ] **service_cascade_orchestration_service.go** — 在 `PlanStopAll` 中，`filterStoppableAll` 返回空切片时直接返回空 `ServiceLifecyclePlan`，不调用 `NewServiceLifecyclePlan`

### F3: collapseAllGroups 空数据保护

- [ ] **status.html** — 在 `collapseAllGroups` 开头增加 `lastStatusData` 为空/空数组的提前返回

### 验证

- [ ] `go build ./src/...` 编译通过
- [ ] `go test ./src/...` 测试通过
- [ ] `curl -X POST /api/build-all` 返回 202 Accepted（异步模式）
- [ ] 全部服务已停止时 `curl -X POST /api/stop-all` 不报错
- [ ] 页面「折叠全部」按钮在空服务列表时不异常
