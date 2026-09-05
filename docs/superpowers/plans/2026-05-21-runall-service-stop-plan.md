# runAll Service Stop/Start Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `runAll` 增加“单服务关闭/启动 + 分组关闭”能力，关闭后进入 `stopped` 状态，并阻止关闭仍有活跃下游依赖的服务。

**Architecture:** 采用“领域先行 + TDD”路径：先在领域层定义服务生命周期事件与停止策略，再在运行器实现状态迁移与依赖保护，最后接入 API 与前端按钮。保持现有 `StatusStore -> /api/status -> status.html` 主链路不变，仅扩展状态与动作。

**Tech Stack:** Go (`net/http`, `sync`, `syscall`), 原生 HTML/CSS/JS, `go test`.

---

## File Structure (Planned Changes)

- Modify: `runAll/src/domain/service_operation_events.go`  
  责任：新增 stop/start/group-stop 领域事件。
- Modify: `runAll/src/status.go`  
  责任：新增 `stopped` 状态及相关状态更新行为。
- Modify: `runAll/src/runner.go`  
  责任：新增 `StopService` / `StartService` / `StopGroup` 与依赖保护逻辑。
- Modify: `runAll/src/ui.go`  
  责任：新增 `POST /api/stop`、`POST /api/start`、`POST /api/stop-group`。
- Modify: `runAll/src/status.html`  
  责任：新增“关闭/启动/关闭本组”按钮与前端请求逻辑。
- Modify: `runAll/src/runner_test.go`  
  责任：覆盖依赖阻断、状态迁移、分组关闭顺序。
- Modify: `runAll/src/ui_test.go`  
  责任：覆盖新增 API 路由与参数/错误处理。
- Modify: `runAll/src/status_test.go`  
  责任：覆盖 `stopped` 状态存储与序列化。

---

### Task 1: 领域契约（DDD）- 生命周期事件补全

**Files:**
- Modify: `runAll/src/domain/service_operation_events.go`
- Test: `runAll/src/runner_test.go`

- [ ] **Step 1: 先写失败测试（断言 stop/start/group-stop 行为可观测）**

Run: `cd runAll && go test ./src -run 'TestRunner_StopService|TestRunner_StartService|TestRunner_StopGroup' -v`  
Expected: FAIL（方法或状态不存在）。

- [ ] **Step 2: 在领域事件文件新增事件定义**

```go
type ServiceStopRequested struct {
	ServiceName string
	OccurredAt  time.Time
}

type ServiceStopped struct {
	ServiceName string
	OccurredAt  time.Time
}

type ServiceStartRequested struct {
	ServiceName string
	OccurredAt  time.Time
}

type ServiceStarted struct {
	ServiceName string
	OccurredAt  time.Time
}

type ServiceGroupStopRequested struct {
	GroupName  string
	OccurredAt time.Time
}
```

- [ ] **Step 3: 运行相关测试确认仍失败但可编译到下一层**

Run: `cd runAll && go test ./src/domain -v`  
Expected: PASS（领域文件编译通过）。

- [ ] **Step 4: 提交本任务**

Run:
```bash
git add runAll/src/domain/service_operation_events.go
git commit -m "feat(runall-domain): add service stop/start/group lifecycle events"
```

---

### Task 2: 状态机扩展 - 新增 stopped 状态

**Files:**
- Modify: `runAll/src/status.go`
- Modify: `runAll/src/status_test.go`
- Test: `runAll/src/status_test.go`

- [ ] **Step 1: 写失败测试（stopped 状态可存取）**

```go
func TestStatusStore_UpdateStopped(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	store.Update("svc", StatusStopped, "")
	got := store.Get("svc")
	if got == nil || got.Status != StatusStopped {
		t.Fatalf("status=%v want=%v", got, StatusStopped)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run TestStatusStore_UpdateStopped -v`  
Expected: FAIL（`StatusStopped` 未定义）。

- [ ] **Step 3: 在状态常量中增加 stopped，并保持 JSON 输出兼容**

```go
const (
	StatusPending    Status = "pending"
	StatusStarting   Status = "starting"
	StatusRetrying   Status = "retrying"
	StatusHealthy    Status = "healthy"
	StatusFailed     Status = "failed"
	StatusSkipped    Status = "skipped"
	StatusRestarting Status = "restarting"
	StatusBuilding   Status = "building"
	StatusStopped    Status = "stopped"
)
```

- [ ] **Step 4: 运行状态测试确认通过**

Run: `cd runAll && go test ./src -run 'TestStatusStore' -v`  
Expected: PASS。

- [ ] **Step 5: 提交本任务**

Run:
```bash
git add runAll/src/status.go runAll/src/status_test.go
git commit -m "feat(runall): add stopped status to service state machine"
```

---

### Task 3: Runner 核心能力 - Stop/Start/StopGroup + 依赖阻断

**Files:**
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/runner_test.go`
- Test: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试（有活跃下游时禁止关闭）**

```go
func TestRunner_StopService_BlocksWhenActiveDependentsExist(t *testing.T) {
	// 构造 a -> b，a healthy 且 b healthy，关闭 a 应报错
}
```

- [ ] **Step 2: 写失败测试（stopped 可 start 恢复）**

```go
func TestRunner_StartService_FromStopped(t *testing.T) {
	// 先 stop，再 start，最终应回到 healthy（或失败时为 failed）
}
```

- [ ] **Step 3: 写失败测试（StopGroup 按逆序关闭）**

```go
func TestRunner_StopGroup_ReverseDependencyOrder(t *testing.T) {
	// 断言先停下游，再停上游
}
```

- [ ] **Step 4: 运行 Runner 测试确认失败**

Run: `cd runAll && go test ./src -run 'TestRunner_StopService|TestRunner_StartService|TestRunner_StopGroup' -v`  
Expected: FAIL。

- [ ] **Step 5: 实现最小 Runner 逻辑**

```go
func (r *Runner) StopService(ctx context.Context, name string) error
func (r *Runner) StartService(ctx context.Context, name string) error
func (r *Runner) StopGroup(ctx context.Context, group string) error
```

实现要点：
- `StopService` 先检查活跃下游依赖（`healthy/retrying/starting/restarting/building` 视为活跃），有则返回错误。
- 通过后执行 `stopMonitoring(name)` + `stopProcess(name)`，状态更新为 `stopped`。
- `StartService` 仅允许当前 `stopped`，然后复用 `startAndCheck`，成功后恢复 monitor。
- `StopGroup` 仅在组内服务集合内按 DAG 逆序关闭，返回聚合错误信息（若有）。

- [ ] **Step 6: 运行 Runner 测试确认通过**

Run: `cd runAll && go test ./src -run 'TestRunner_StopService|TestRunner_StartService|TestRunner_StopGroup' -v`  
Expected: PASS。

- [ ] **Step 7: 提交本任务**

Run:
```bash
git add runAll/src/runner.go runAll/src/runner_test.go
git commit -m "feat(runall): support stop/start/group-stop with dependency guard"
```

---

### Task 4: API 接口扩展 - stop/start/stop-group

**Files:**
- Modify: `runAll/src/ui.go`
- Modify: `runAll/src/ui_test.go`
- Test: `runAll/src/ui_test.go`

- [ ] **Step 1: 写失败测试（新增三个路由）**

```go
func TestAPIStopService(t *testing.T) {}
func TestAPIStartService(t *testing.T) {}
func TestAPIStopGroup(t *testing.T) {}
```

校验点：
- method 必须是 `POST`
- `name/group` 必填
- Runner 返回错误时透传 JSON error

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run 'TestAPIStopService|TestAPIStartService|TestAPIStopGroup' -v`  
Expected: FAIL（404 或 handler 缺失）。

- [ ] **Step 3: 在 `registerUIHandlers` 增加路由和请求体处理**

Run: `cd runAll && go test ./src -run 'TestAPIStopService|TestAPIStartService|TestAPIStopGroup' -v`  
Expected: PASS。

- [ ] **Step 4: 提交本任务**

Run:
```bash
git add runAll/src/ui.go runAll/src/ui_test.go
git commit -m "feat(runall-api): add stop/start/stop-group endpoints"
```

---

### Task 5: UI 交互 - 关闭/启动/关闭本组按钮

**Files:**
- Modify: `runAll/src/status.html`
- Modify: `runAll/src/ui_test.go`
- Test: `runAll/src/ui_test.go`

- [ ] **Step 1: 写失败测试（首页包含新动作片段）**

Run: `cd runAll && go test ./src -run TestUIHomePage -v`  
Expected: FAIL（缺少新按钮/动作标识）。

- [ ] **Step 2: 修改前端状态映射与按钮渲染**

实现要点：
- `dotClass` 增加 `stopped: 'gray'`
- 服务行为 `stopped` 时显示“启动”按钮，否则显示“关闭”
- 每个组标题旁增加“关闭本组”按钮

- [ ] **Step 3: 新增前端请求函数**

```javascript
postServiceAction('/api/stop', name, 'Stop')
postServiceAction('/api/start', name, 'Start')
postGroupAction('/api/stop-group', groupName, 'Stop group')
```

- [ ] **Step 4: 运行前后端联合测试**

Run: `cd runAll && go test ./src -run 'TestUIHomePage|TestAPIStatus' -v`  
Expected: PASS。

- [ ] **Step 5: 提交本任务**

Run:
```bash
git add runAll/src/status.html runAll/src/ui_test.go
git commit -m "feat(runall-ui): add stop/start and group stop actions"
```

---

### Task 6: 全量回归与手工冒烟

**Files:**
- Test: `runAll/src/...` 全量
- Optional Modify: `docs/superpowers/specs/2026-05-21-runall-service-stop-design.md`（若实现与设计有偏差）

- [ ] **Step 1: 跑全量测试**

Run: `cd runAll && go test ./...`  
Expected: PASS。

- [ ] **Step 2: 手工验证 UI 行为**

Run:
```bash
cd runAll
go run ./src --config ../runAll.yaml --ui-port :9999
```

Expected:
- 单服务可“关闭”，状态变 `stopped`；
- `stopped` 服务可“启动”并恢复健康；
- 有活跃下游依赖时关闭被拒绝；
- 点击“关闭本组”按逆依赖顺序关闭。

- [ ] **Step 3: 最终提交（如尚有变更）**

Run:
```bash
git add runAll docs/superpowers/specs/2026-05-21-runall-service-stop-design.md
git commit -m "test(runall): validate service stop/start and group stop flow"
```

---

## DDD Structure Validation Checklist

- [ ] 领域事件先于 Runner/API/UI 落地（domain first）。
- [ ] 停止策略（依赖阻断）由领域规则驱动，基础设施仅执行进程控制。
- [ ] API 层不内嵌依赖图判断，仅调用 Runner 能力。
- [ ] 任务顺序保持 `domain -> state model -> runner -> api -> ui -> regression`。

## Self-Review

- Spec coverage: 已覆盖单服务关闭/启动、分组关闭、`stopped` 状态、依赖阻断、测试与回归。
- Placeholder scan: 无 `TODO/TBD/later` 占位。
- Type consistency: 统一使用 `stopped`、`StopService/StartService/StopGroup`、`/api/stop|start|stop-group`。
