# runAll Stability-First Orchestration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 runAll 成为唯一、可诊断、可恢复的服务编排入口，彻底消除“端口冲突导致 pending”与“启动成功但未就绪不可区分”问题。

**Architecture:** 以 `Runner` 为应用编排入口，按 `Preflight -> Launch -> Readiness` 三阶段执行；领域层通过 `ServiceOwnership`、`StartupSession`、`ServiceStartupAttempt` 维护一致性与审计语义；状态与 UI 只消费结构化失败（`phase + failure_code + hint + session_id`），不再依赖散乱日志文本。

**Tech Stack:** Go, YAML config, in-memory repositories (phase 1), stdlib process/syscall/http, Go test

---

## File Structure (先锁定边界)

- Modify: `runAll/src/config.go`（单一配置源校验、配置指纹）
- Modify: `runAll/src/main.go`（新增命令入口：`doctor` / `takeover`）
- Modify: `runAll/src/runner.go`（三阶段状态机、ownership 校验、失败分类）
- Modify: `runAll/src/status.go`（状态结构增加 phase/failure/session/hint）
- Modify: `runAll/src/ui.go`（状态页展示结构化失败）
- Create: `runAll/src/domain/service_failure_hint_value_object.go`
- Create: `runAll/src/domain/service_runtime_prereq_probe_repository.go`
- Create: `runAll/src/infrastructure/inmemory_service_ownership_repository.go`
- Create: `runAll/src/infrastructure/inmemory_startup_session_repository.go`
- Create: `runAll/src/infrastructure/inmemory_startup_attempt_repository.go`
- Create: `runAll/src/preflight.go`
- Create: `runAll/src/doctor.go`
- Test: `runAll/src/runner_test.go`（扩展：冲突拦截、前置失败、readiness 分类）
- Test: `runAll/src/config_test.go`（单一配置源与配置分叉 fail-fast）
- Test: `runAll/src/status_test.go`、`runAll/src/ui_test.go`（结构化字段透传）

---

### Task 1: 配置源唯一化与 ConfigFingerprint 落库

**Files:**
- Modify: `runAll/src/config.go`
- Modify: `runAll/src/config_test.go`
- Test: `runAll/src/config_test.go`

- [ ] **Step 1: 写失败测试（双配置分叉必须 fail-fast）**

```go
func TestLoadConfig_FailsWhenDualConfigMismatch(t *testing.T) {
    rootCfg := t.TempDir() + "/runAll.yaml"
    innerCfg := t.TempDir() + "/runAll/config.yaml"
    _ = os.WriteFile(rootCfg, []byte("version: '1'\ngroups: []\n"), 0o644)
    _ = os.WriteFile(innerCfg, []byte("version: '2'\ngroups: []\n"), 0o644)

    _, err := LoadConfigWithSourceGuard(rootCfg, innerCfg)
    require.Error(t, err)
    require.Contains(t, err.Error(), "config source mismatch")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run TestLoadConfig_FailsWhenDualConfigMismatch -v`  
Expected: FAIL（`LoadConfigWithSourceGuard` 未实现）

- [ ] **Step 3: 最小实现（统一入口 + 指纹）**

```go
func LoadConfigWithSourceGuard(primaryPath, secondaryPath string) (*Config, domain.ConfigFingerprint, error) {
    cfg, err := LoadConfig(primaryPath)
    if err != nil { return nil, domain.ConfigFingerprint{}, err }
    primaryHash, _ := fileSHA256(primaryPath)
    if secondaryPath != "" {
        if _, err := os.Stat(secondaryPath); err == nil {
            secondaryHash, _ := fileSHA256(secondaryPath)
            if primaryHash != secondaryHash {
                return nil, domain.ConfigFingerprint{}, fmt.Errorf("config source mismatch: %s != %s", primaryPath, secondaryPath)
            }
        }
    }
    fp, err := domain.NewConfigFingerprint(primaryPath, primaryHash)
    return cfg, fp, err
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll && go test ./src -run TestLoadConfig_FailsWhenDualConfigMismatch -v`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `git add runAll/src/config.go runAll/src/config_test.go && git commit -m "feat(runall): enforce single config source with fingerprint"`

---

### Task 2: Ownership 仓储实现与 owner-only 操作约束

**Files:**
- Create: `runAll/src/infrastructure/inmemory_service_ownership_repository.go`
- Modify: `runAll/src/runner.go`
- Test: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试（非 owner stop/restart 必须拒绝）**

```go
func TestStopService_RejectsNonOwnerSession(t *testing.T) {
    r := newRunnerForOwnershipTest(t)
    err := r.StopServiceWithActor(context.Background(), "saas-backend", "session-b")
    require.Error(t, err)
    require.Contains(t, err.Error(), "requires explicit takeover")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run TestStopService_RejectsNonOwnerSession -v`  
Expected: FAIL（`StopServiceWithActor` 未实现）

- [ ] **Step 3: 最小实现（接入 `ServiceOwnershipGuardService`）**

```go
func (r *Runner) StopServiceWithActor(ctx context.Context, name, actorSessionID string) error {
    if err := r.ownershipGuard.EnsureOperableBySession(name, actorSessionID); err != nil {
        return err
    }
    return r.StopService(ctx, name)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll && go test ./src -run TestStopService_RejectsNonOwnerSession -v`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `git add runAll/src/infrastructure/inmemory_service_ownership_repository.go runAll/src/runner.go runAll/src/runner_test.go && git commit -m "feat(runall): enforce owner-only service operations"`

---

### Task 3: Preflight 端口冲突拦截（3 秒内失败）

**Files:**
- Create: `runAll/src/preflight.go`
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/status.go`
- Test: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试（外来进程占用返回 `PRECHECK_PORT_CONFLICT`）**

```go
func TestRun_FailsFastOnForeignPortConflict(t *testing.T) {
    r, cancel := newRunnerWithOccupiedPort(t)
    defer cancel()
    err := r.Run(context.Background(), true)
    require.Error(t, err)
    require.Contains(t, err.Error(), "PRECHECK_PORT_CONFLICT")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run TestRun_FailsFastOnForeignPortConflict -v`  
Expected: FAIL（无 preflight 冲突判断）

- [ ] **Step 3: 最小实现（先 Preflight 再 Launch）**

```go
if err := r.preflightService(ctx, node.Service); err != nil {
    r.store.UpdateFailure(node.Service.Name, "preflight", "PRECHECK_PORT_CONFLICT", err.Error(), r.sessionID)
    return fmt.Errorf("[%s] PRECHECK_PORT_CONFLICT: %w", node.Service.Name, err)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll && go test ./src -run TestRun_FailsFastOnForeignPortConflict -v`  
Expected: PASS（且日志显示 preflight 阶段失败）

- [ ] **Step 5: 提交**

Run: `git add runAll/src/preflight.go runAll/src/runner.go runAll/src/status.go runAll/src/runner_test.go && git commit -m "feat(runall): add preflight port conflict fail-fast"`

---

### Task 4: 三阶段生命周期 + 失败分类（Launch/Readiness 分离）

**Files:**
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/status.go`
- Test: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试（readiness 超时必须是 `READINESS_TIMEOUT`）**

```go
func TestStartAndCheck_ClassifiesReadinessTimeout(t *testing.T) {
    r := newRunnerWithNeverHealthyService(t)
    err := r.startAndCheck(context.Background(), &ServiceNode{Service: r.cfg.Flatten()[0]})
    require.Error(t, err)
    status := r.store.Get("failing-service")
    require.Equal(t, "readiness", status.Phase)
    require.Equal(t, "READINESS_TIMEOUT", status.FailureCode)
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run TestStartAndCheck_ClassifiesReadinessTimeout -v`  
Expected: FAIL（尚未记录 phase/failure_code）

- [ ] **Step 3: 最小实现（状态切分到 phase + code）**

```go
r.store.UpdatePhase(svc.Name, "launch")
if err := cmd.Start(); err != nil {
    r.store.UpdateFailure(svc.Name, "launch", "LAUNCH_PROCESS_EXITED", err.Error(), r.sessionID)
    return err
}
r.store.UpdatePhase(svc.Name, "readiness")
if err := waitHealthy(ctx, svc.HealthCheck); err != nil {
    r.store.UpdateFailure(svc.Name, "readiness", "READINESS_TIMEOUT", err.Error(), r.sessionID)
    return err
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll && go test ./src -run TestStartAndCheck_ClassifiesReadinessTimeout -v`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `git add runAll/src/runner.go runAll/src/status.go runAll/src/runner_test.go && git commit -m "feat(runall): model launch and readiness phases with structured failures"`

---

### Task 5: Runtime Prereq Gate（migration 等关键前置）

**Files:**
- Create: `runAll/src/domain/service_runtime_prereq_probe_repository.go`
- Modify: `runAll/src/runner.go`
- Test: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试（prereq 失败返回 `PRECHECK_RUNTIME_PREREQ_FAILED`）**

```go
func TestRun_BlocksOnRuntimePrereqFailure(t *testing.T) {
    r := newRunnerWithFailingPrereqProbe(t)
    err := r.Run(context.Background(), true)
    require.Error(t, err)
    require.Contains(t, err.Error(), "PRECHECK_RUNTIME_PREREQ_FAILED")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run TestRun_BlocksOnRuntimePrereqFailure -v`  
Expected: FAIL（尚无 runtime prereq gate）

- [ ] **Step 3: 最小实现（Preflight 注入 probe）**

```go
if err := r.runtimePrereqProbeRepo.CheckServicePrerequisites(ctx, svc.Name); err != nil {
    r.store.UpdateFailure(svc.Name, "preflight", "PRECHECK_RUNTIME_PREREQ_FAILED", err.Error(), r.sessionID)
    return fmt.Errorf("PRECHECK_RUNTIME_PREREQ_FAILED: %w", err)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll && go test ./src -run TestRun_BlocksOnRuntimePrereqFailure -v`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `git add runAll/src/domain/service_runtime_prereq_probe_repository.go runAll/src/runner.go runAll/src/runner_test.go && git commit -m "feat(runall): add runtime prerequisite gate in preflight"`

---

### Task 6: `doctor` / `takeover` 命令与 UI 可诊断化

**Files:**
- Create: `runAll/src/doctor.go`
- Modify: `runAll/src/main.go`
- Modify: `runAll/src/ui.go`
- Modify: `runAll/src/ui_test.go`
- Modify: `runAll/src/status_test.go`

- [ ] **Step 1: 写失败测试（doctor 返回确定性退出码 + 结构化结果）**

```go
func TestDoctor_ReturnsNonZeroWhenPreflightFails(t *testing.T) {
    code, report := runDoctorForTest(t, failingDoctorRunner())
    require.Equal(t, 2, code)
    require.Contains(t, report, "PRECHECK_PORT_CONFLICT")
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll && go test ./src -run TestDoctor_ReturnsNonZeroWhenPreflightFails -v`  
Expected: FAIL（doctor 未实现）

- [ ] **Step 3: 最小实现（新增命令分发 + UI 字段）**

```go
// main.go
command := flag.String("command", "run", "run|doctor|takeover")
// ...
switch *command {
case "doctor":
    os.Exit(RunDoctor(ctx, runner))
case "takeover":
    err = runner.TakeoverService(*serviceName, *actorSessionID)
default:
    err = runner.Run(ctx, *daemon)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll && go test ./src -run "TestDoctor_ReturnsNonZeroWhenPreflightFails|TestUIIncludesFailureFields" -v`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `git add runAll/src/doctor.go runAll/src/main.go runAll/src/ui.go runAll/src/ui_test.go runAll/src/status_test.go && git commit -m "feat(runall): add doctor and structured failure visibility in UI"`

---

### Task 7: 全链路回归与验收脚本

**Files:**
- Modify: `runAll/src/runner_test.go`
- Modify: `runAll/src/config_test.go`
- Modify: `runAll/src/ui_test.go`

- [ ] **Step 1: 补齐回归测试矩阵**

```go
func TestRegressionMatrix(t *testing.T) {
    t.Run("port conflict fail-fast", testPortConflict)
    t.Run("runtime prereq blocked", testPrereqBlocked)
    t.Run("fix prereq then healthy", testPrereqRecoveredHealthy)
    t.Run("non-owner rejected", testNonOwnerRejected)
}
```

- [ ] **Step 2: 运行全部测试**

Run: `cd runAll && go test ./src/... -v`  
Expected: PASS（覆盖 M1/M2/M3 核心场景）

- [ ] **Step 3: 运行格式化与静态检查**

Run: `cd runAll && gofmt -w src/**/*.go && go test ./src/...`  
Expected: PASS

- [ ] **Step 4: 验证 NFR 关键指标（手工 smoke）**

Run: `cd runAll && go run ./src/main.go -command doctor -config ./config.yaml`  
Expected: 输出每服务 phase/code/hint；失败时退出码非 0

- [ ] **Step 5: 提交**

Run: `git add runAll/src && git commit -m "test(runall): add stability-first regression coverage and acceptance checks"`

---

## 计划自检

- 覆盖性：M1（配置源+ownership）、M2（三阶段+prereq gate）、M3（doctor+UI可诊断）均有对应任务。
- 无占位符：无 `TODO/TBD/implement later`。
- 依赖顺序：先领域接口与值对象，再 Runner 应用编排，再 UI/CLI，再回归测试。
- 与 `/5-ddd` 一致：所有新增行为都通过现有领域对象与仓储接口表达，没有直接引入基础设施耦合到 `domain/`。

