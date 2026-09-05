# 实施计划: runAll 启动竞态 EADDRINUSE 修复

> Derived from:
> - Design: `.claude/plans/01-brainstorming-设计文档.md`
> - Value Stream: `docs/superpowers/plans/2026-06-22-runall-startup-race-eaddrinuse-fix-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-22-runall-startup-race-eaddrinuse-fix-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-22-runall-startup-race-eaddrinuse-fix-ddd.md`

## 修复概览

4 个增量，按依赖排序。增量 1 是核心修复（CAS 守卫），增量 2-4 是防御层。

---

## Increment 1: startAndCheck CAS 并发守卫 🔴 Core

### 1.1: 编写单元测试 — transitionServiceToStarting 并发安全

- [ ] 文件: `runAll/src/runner_startup_race_test.go`
- [ ] 测试: `TestTransitionServiceToStarting_ConcurrentCalls_OnlyOneSucceeds` — 两个 goroutine 同时 CAS `pending→starting`，断言只有一个成功
- [ ] 测试: `TestTransitionServiceToStarting_FromVariousStates` — 验证从 stopped/failed/skipped/pending 均可 CAS 成功
- [ ] 测试: `TestTransitionServiceToStarting_FromStarting_Fails` — 验证从 starting 状态 CAS 失败

### 1.2: 实现 — startAndCheck 入口 CAS 守卫

- [ ] 文件: `runAll/src/runner.go` — `startAndCheck()` 函数
- [ ] 在函数入口（line 449 之前）插入 `transitionServiceToStarting` CAS 调用
- [ ] CAS 成功 → 继续原有逻辑
- [ ] CAS 失败 → 调用新增的 `waitForServiceStart()` 等待另一个 goroutine 完成
- [ ] 移除 line 449 的无条件 `r.store.Update(svc.Name, StatusStarting, "")`（已被 CAS 替代）

### 1.3: 实现 — waitForServiceStart 辅助方法

- [ ] 文件: `runAll/src/runner.go`
- [ ] 新增 `func (r *Runner) waitForServiceStart(ctx context.Context, svc Service) error`
- [ ] 轮询 `r.store.Get(svc.Name)`，检测状态变为 healthy/failed/stopped
- [ ] 超时 30 秒后返回错误
- [ ] 轮询间隔 500ms

### 1.4: 编写集成测试 — DAG + API 并发场景

- [ ] 文件: `runAll/src/runner_startup_race_integration_test.go`
- [ ] 测试: 模拟两个 goroutine 同时调用 `startService` → 断言 `cmd.Start()` 只执行一次
- [ ] 测试: 验证 CAS 失败方收到明确的 "another goroutine is starting" 错误

### 1.5: 运行测试并验证

- [ ] 命令: `cd runAll && go test ./... -race -count=1 -run "TransitionServiceToStarting|StartupRace"`
- [ ] 确认 `-race` 检测器通过

---

## Increment 2: killPreviousRunAllProcess 等待托管服务释放 🟡 Defense

### 2.1: 编写单元测试 — 孤儿进程检测

- [ ] 文件: `runAll/src/main_startup_cleanup_test.go`
- [ ] 测试: `TestKillPreviousRunAllProcess_WaitsForManagedPorts` — mock lsof 返回端口占用 PID，验证等待逻辑
- [ ] 测试: `TestKillPreviousRunAllProcess_Timeout_SendsSIGKILL` — 超时后强制执行清理

### 2.2: 实现 — 增强 killPreviousRunAllProcess

- [ ] 文件: `runAll/src/main.go` — `killPreviousRunAllProcess()` 函数
- [ ] 在杀旧 runAll 之前，先通过旧 runAll 的 `/api/shutdown-self` 发起优雅关闭
- [ ] 之后遍历已知托管服务端口（从配置读取），用 `lsof` 检测是否仍被占用
- [ ] 等待端口释放（最长 15 秒，每 500ms 检查一次）
- [ ] 超时未释放 → `SIGKILL` 清理孤儿进程
- [ ] 日志输出清理过程（清理了哪些 PID/端口）

### 2.3: 运行测试

- [ ] 命令: `cd runAll && go test ./... -race -count=1 -run "KillPreviousRunAll"`

---

## Increment 3: gitOauth SO_REUSEADDR 修复 🟡 Defense

### 3.1: 修复 NoReverseDNSWSGIServer

- [ ] 文件: `gitOauth/run.sh` — `NoReverseDNSWSGIServer` 类
- [ ] 在 `server_bind()` 中，`self.socket.bind()` 之前增加:
  ```python
  self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
  ```
- [ ] 还需要 `import socket` (检查现有 import)

### 3.2: 排查其他 Python WSGI 服务

- [ ] 搜索 `grep -rn "def server_bind\|class.*WSGIServer" gitOauth/ task2app/ taskAuth/ taskBill/ --include="*.sh" --include="*.py"`
- [ ] 对发现的自定义 WSGI 服务，同样增加 `SO_REUSEADDR`

### 3.3: 手动验证

- [ ] 命令: 连续 5 次重启 gitOauth → 无 EADDRINUSE
- [ ] 验证方法: `cd gitOauth && for i in 1 2 3 4 5; do bash scripts/runall-stop.sh; sleep 1; bash run.sh & sleep 2; curl -s http://localhost:8002/api/health/; done`

---

## Increment 4: listenerPIDs 错误不静默 🟢 Defense

### 4.1: 修复错误处理

- [ ] 文件: `runAll/src/runner.go` — `startAndCheck()` 函数中的端口检测 (lines 454-458)
- [ ] 将 `continue` 改为: 标记服务 Failed，返回错误
  ```go
  if err != nil {
      log.Printf("[%s] port check error on %s: %v", svc.Name, port, err)
      r.store.Update(svc.Name, StatusFailed, 
          fmt.Sprintf("port check failed on %s: %v", port, err))
      r.store.UpdateDependencyStatus(svc.Name, StatusFailed)
      return fmt.Errorf("[%s] port check error on %s: %w", svc.Name, port, err)
  }
  ```

### 4.2: 编写测试

- [ ] 文件: `runAll/src/runner_port_check_test.go`
- [ ] 测试: `TestStartAndCheck_PortCheckError_FailsImmediately` — mock `listenerPIDs` 返回错误，断言服务标记 Failed
- [ ] 测试: `TestStartAndCheck_PortCheckError_DoesNotLaunch` — 断言 `cmd.Start()` 未被调用

### 4.3: 运行测试

- [ ] 命令: `cd runAll && go test ./... -race -count=1 -run "PortCheck"`

---

## 构建与回归验证

### 5.1: 完整构建

- [ ] 命令: `cd runAll && ./build.sh`
- [ ] 确认编译通过，无 warning

### 5.2: 全量回归测试

- [ ] 命令: `cd runAll && go test ./... -race -count=1`
- [ ] 确认所有已有测试通过

### 5.3: 端到端验证

- [ ] 场景 A: 正常 DAG 引导 — 所有服务 healthy
- [ ] 场景 B: DAG 引导中点击「全部启动」— 无 EADDRINUSE
- [ ] 场景 C: 重启 runAll — 无孤儿进程冲突
- [ ] 场景 D: 连续 5 次重启 gitOauth — 全部成功

---

## 文件变更清单

| 文件 | 变更类型 | 增量 |
|------|---------|------|
| `runAll/src/runner.go` | 修改 | Inc 1, 4 — CAS 守卫 + 错误处理 |
| `runAll/src/main.go` | 修改 | Inc 2 — 等待托管服务释放 |
| `gitOauth/run.sh` | 修改 | Inc 3 — SO_REUSEADDR |
| `runAll/src/runner_startup_race_test.go` | 新增 | Inc 1 — CAS 单元测试 |
| `runAll/src/runner_startup_race_integration_test.go` | 新增 | Inc 1 — 并发集成测试 |
| `runAll/src/main_startup_cleanup_test.go` | 新增 | Inc 2 — 孤儿进程测试 |
| `runAll/src/runner_port_check_test.go` | 新增 | Inc 4 — 端口检测错误测试 |

## 风险点

| 风险 | 缓解 |
|------|------|
| `waitForServiceStart` 轮询引入延迟 | 500ms 间隔 + 30s 超时，实际启动 < 5s |
| `killPreviousRunAllProcess` 改动影响启动速度 | 15s 超时仅在旧 runAll 异常时触发，正常场景无需等待 |
| SO_REUSEADDR 可能掩盖真正的端口冲突 | SO_REUSEADDR 仅允许 TIME_WAIT 复用，不影响活跃 LISTEN 端口 |
