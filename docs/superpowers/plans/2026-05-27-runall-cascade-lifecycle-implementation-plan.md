# runAll 链式启停（Cascade Lifecycle）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `http://localhost:9999/` 实现依赖链一键启停：默认链式启动/关闭、启动本组，失败可诊断，并保留 `cascade=false` 旧语义。

**Architecture:** 领域层 `ServiceCascadeOrchestrationService` 生成 `ServiceLifecyclePlan`（已实现）；Runner 串行执行既有 `StartServiceWithActor` / `StopServiceWithActor`；`configTopologyRepository` 适配 `Config.Flatten()`；UI 默认 `cascade: true`。

**Tech Stack:** Go, embed HTML/JS, `net/http`, 既有 `StatusStore` / `Runner`

**输入文档:**
- 设计: `docs/superpowers/specs/2026-05-27-runall-cascade-lifecycle-design.md`
- 价值流: `docs/superpowers/plans/2026-05-27-runall-cascade-lifecycle-value-stream.md`
- NFR: `docs/superpowers/plans/2026-05-27-runall-cascade-lifecycle-nfr-clarification.md`
- DDD: `docs/superpowers/plans/2026-05-27-runall-cascade-lifecycle-ddd-model.md`

---

## File Structure

| 文件 | 责任 |
|------|------|
| `runAll/src/domain/service_cascade_orchestration_service.go` | 计划生成（**已存在**） |
| `runAll/src/infrastructure/config_service_topology_repository.go` | **新建** — `ServiceTopologyRepository` |
| `runAll/src/runner.go` | `StartServiceCascade*` / `StopServiceCascade*` / `StartGroup*` |
| `runAll/src/ui.go` | `cascade` 解析、错误体、`/api/start-group` |
| `runAll/src/status.html` | 默认 cascade、启动本组、失败 alert |
| `runAll/src/runner_test.go` | 链式集成测试 |
| `runAll/src/ui_test.go` | API 默认 cascade、错误 JSON |

---

### Task 0: 确认领域层基线（已完成）

**Files:** `runAll/src/domain/service_cascade_*.go`（已提交于 DDD 步骤）

- [x] **Step 1: 运行领域测试**

Run: `cd runAll && go test ./src/domain/... -run Cascade -v`  
Expected: PASS（5 个用例）

---

### Task 1: 拓扑仓储 + Runner 启动链式（Increment 1 — Thin Slice）

**Files:**
- Create: `runAll/src/infrastructure/config_service_topology_repository.go`
- Create: `runAll/src/infrastructure/config_service_topology_repository_test.go`
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试 — 拓扑仓储**

```go
func TestConfigServiceTopologyRepository_ListAll(t *testing.T) {
    // 两服务 b depends_on a，ListAll 返回 2 节点
}
```

Run: `cd runAll && go test ./src/infrastructure/... -run TestConfigServiceTopologyRepository -v`  
Expected: FAIL

- [ ] **Step 2: 实现 `configServiceTopologyRepository`**

从 `*Config` 的 `Flatten()` 构建 `[]domain.ServiceTopologyNode`。

- [ ] **Step 3: 写失败测试 — StartServiceCascade 顺序**

```go
func TestRunner_StartServiceCascade_StartsUpstreamFirst(t *testing.T) {
    // a->b->c 全 stopped，StartServiceCascade(c) 后调用顺序或最终状态为 healthy
    // 可用记录 start 调用顺序的 spy / 假健康检查
}
```

Run: `cd runAll && go test ./src -run TestRunner_StartServiceCascade -v`  
Expected: FAIL

- [ ] **Step 4: 实现 Runner 方法**

```go
func (r *Runner) StartServiceCascadeWithActor(ctx, name, actor string) error
func (r *Runner) executeLifecyclePlan(ctx, plan domain.ServiceLifecyclePlan, actor string, start bool) error
```

- 构造 `NewServiceCascadeOrchestrationService(topology, runtimeRepo)`
- `PlanStartCascade` → 对 `plan.OrderedNames` 串行 `StartServiceWithActor`
- 失败时返回 `fmt.Errorf("start cascade failed on %q: %w", step, err)`（Increment 4 再扩展 JSON）

- [ ] **Step 5: 运行测试通过**

Run: `cd runAll && go test ./src -run 'TestRunner_StartServiceCascade|TestConfigServiceTopology' -v`

---

### Task 2: UI/API 默认启动链式（Increment 1 闭环）

**Files:**
- Modify: `runAll/src/ui.go`
- Modify: `runAll/src/status.html`
- Modify: `runAll/src/ui_test.go`

- [ ] **Step 1: 写失败测试 — API 默认 cascade**

```go
func TestAPIStart_DefaultCascadeTrue(t *testing.T) {
    // body 仅 name + session_id，应走 cascade 路径（可通过 mock 或日志断言）
}
```

- [ ] **Step 2: `ui.go` 解析 `cascade`**

- `readJSONPayload` 后读取 `cascade`，缺省为 `true`
- `cascade==true` → `StartServiceCascadeWithActor`；否则 `StartServiceWithActor`

- [ ] **Step 3: `status.html` 启动请求**

`postServiceAction` 对 start/stop 增加 `cascade: true`（或专用字段默认 true）。

- [ ] **Step 4: 运行 UI 测试**

Run: `cd runAll && go test ./src -run 'TestAPIStart' -v`

- [ ] **Step 5: 手工验收 QS-01**

打开 `http://localhost:9999/`，`taskFE` stopped 时单次点击「启动」，观察 `git-oauth` → `saas-backend` → `taskFE` 依次变绿。

---

### Task 3: Runner 关闭链式（Increment 2）

**Files:**
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试**

```go
func TestRunner_StopServiceCascade_StopsDownstreamFirst(t *testing.T) {
    // a<-b<-c 全 healthy，StopServiceCascade(a) 顺序 c,b,a
}
```

- [ ] **Step 2: 实现 `StopServiceCascadeWithActor`**

`PlanStopCascade` → 串行 `StopServiceWithActor`

- [ ] **Step 3: UI stop 默认 cascade**

`ui.go` + `status.html`（Task 2 若未完成则一并做）

- [ ] **Step 4: 回归 `cascade=false` 阻断**

```go
func TestRunner_StopService_BlocksWhenCascadeFalse(t *testing.T) {
    // 现有 TestRunner_StopService_BlocksWhenActiveDependentsExist 保持通过
}
```

Run: `cd runAll && go test ./src -run 'TestRunner_StopService' -v`

---

### Task 4: 启动本组（Increment 3）

**Files:**
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/ui.go`
- Modify: `runAll/src/status.html`
- Modify: `runAll/src/ui_test.go`
- Modify: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试 — StartGroup**

```go
func TestRunner_StartGroup_PlatformOrder(t *testing.T) {
    // 仿 platform 依赖，断言 PlanStartGroup 顺序
}
func TestAPIStartGroup(t *testing.T) {
    // POST /api/start-group { group, session_id }
}
```

- [ ] **Step 2: 实现 `StartGroupWithActor`**

复用 `executeLifecyclePlan` + `PlanStartGroup`。

- [ ] **Step 3: UI「启动本组」**

组头按钮 `data-action="start-group"`，调用 `postGroupAction('/api/start-group', ...)`。

- [ ] **Step 4: 回归 StopGroup**

Run: `cd runAll && go test ./src -run 'TestRunner_StopGroup|TestAPIStopGroup' -v`  
Expected: PASS（无行为变更）

---

### Task 5: 级联失败可诊断 + cascade 错误体（Increment 4）

**Files:**
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/ui.go`
- Modify: `runAll/src/status.html`
- Modify: `runAll/src/ui_test.go`

- [ ] **Step 1: 定义执行错误类型**

```go
type cascadeError struct {
    err error
    report domain.CascadeExecutionReport
}
```

`executeLifecyclePlan` 在失败时填充 `Completed` / `FailedAt`。

- [ ] **Step 2: `writeCascadeError` in `ui.go`**

HTTP 400 + JSON:

```json
{"error":"...", "cascade":{"completed":["a"],"failed_at":"b"}}
```

- [ ] **Step 3: UI alert 展示**

解析 `result.cascade` 追加到 `alert` 文案。

- [ ] **Step 4: 日志**

`log.Printf("[cascade] %s plan: %s", op, plan.String())`

- [ ] **Step 5: 测试**

```go
func TestAPIStartCascadeFailureReturnsCascadeBody(t *testing.T) {}
```

Run: `cd runAll && go test ./src -run 'TestAPI.*Cascade' -v`

---

### Task 6: 全量回归与价值流登记

**Files:**
- Modify: `value-stream.yaml`（实现后将相关 step `status: planned` → `active`，可选）
- Create: `task2app/Saas_project/view_test/runall-start-cascade-thin-slice.md`（验收清单，可选）

- [ ] **Step 1: 全量 runAll 测试**

Run: `cd runAll && go test ./... -race`

- [ ] **Step 2: valueStream 配置校验**

Run: `cd valueStream && go test ./...`

- [ ] **Step 3: 验收对照**

| QS | 验证 |
|----|------|
| QS-01 | 一键启动 taskFE |
| QS-02 | 关闭 git-oauth 连带下游 |
| QS-03 | 失败含 failed_at |
| QS-05 | cascade=false 阻断 |

---

## 实施顺序摘要

```text
Task 0 (domain 已就绪)
  → Task 1–2 (启动链式 E2E)
  → Task 3 (关闭链式)
  → Task 4 (启动本组)
  → Task 5 (错误诊断)
  → Task 6 (回归)
```

## 明确不做（本计划）

- `StartAll` / `StopAll`（Increment 5 / 二期）
- 链式 `RestartService` / `BuildService`
- 链内并行 level 执行
