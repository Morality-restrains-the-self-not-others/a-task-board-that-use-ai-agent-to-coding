# 设计文档：修复编译(build)受运行状态限制的问题

## 问题描述

页面 `http://183.250.1.132:9999/` 点击「编译」按钮时触发错误：

```
Build failed: service "task-events-billing-transaction-created-1-process-billing-transaction"
is skipped, can only build healthy, failed, or stopped services
```

原因：`BuildService` 和 `BuildGroup` 函数限制了只有 `healthy`、`failed`、`stopped` 三种状态的服务才能编译。用户认为**编译是纯磁盘操作（如 `go build`、`./build.sh`），不应受服务运行状态限制**。

## 受影响代码

| 文件 | 函数/位置 | 问题 |
|------|----------|------|
| `runAll/src/runner.go:1594-1628` | `BuildService` | CAS 只允许 healthy/failed/stopped → building |
| `runAll/src/runner.go:1512` | `BuildGroup` | 调用 `IsTerminalBuildStatus` 过滤 |
| `runAll/src/domain/build_group_result_value_object.go:76-86` | `IsTerminalBuildStatus` | 只允许 3 种状态 |
| `runAll/src/runner_test.go:373-379` | `TestBuildService_StatusConflict` | 测试用例需更新 |
| `runAll/src/runner_test.go:410-416` | `TestBuildService_ConcurrentBuildRejected` | 测试用例需更新 |
| `runAll/src/domain/build_group_result_value_object_test.go:109-130` | `TestIsTerminalBuildStatus` | 测试用例需更新 |

## 根因分析

```
BuildService (runner.go:1594)
  └─ CompareAndSwapStatus(name, StatusHealthy, StatusBuilding)     ← 只允许 healthy
  └─ CompareAndSwapStatus(name, StatusFailed, StatusBuilding)      ← 只允许 failed  
  └─ CompareAndSwapStatus(name, StatusStopped, StatusBuilding)     ← 只允许 stopped
  └─ else → error("can only build healthy, failed, or stopped")    ← 其他状态被拒绝

BuildGroup (runner.go:1487)
  └─ IsTerminalBuildStatus(current.Status)                         ← 同样只允许 3 种状态
  └─ 不满足 → 跳过该服务
```

**设计原则**：编译(build)本质是执行构建脚本（`go build`、`./build.sh` 等），只写磁盘产出物（二进制等），不依赖也不应受服务运行时状态限制。唯一的约束是**防止并发编译**（同一服务同时两个 build）。

## 修复方案

### 修改 1：`BuildService` — 允许全部非 `building` 状态

**文件**: `runAll/src/runner.go:1594-1628`

将 3 个 CAS 分支扩展为覆盖全部状态（共 8 种允许状态），仅 `StatusBuilding` 不可编译：

```go
func (r *Runner) BuildService(ctx context.Context, name string) error {
    svc := r.findService(name)
    if svc == nil {
        return fmt.Errorf("service %q not found", name)
    }
    buildCmd := resolveBuildCommand(*svc)
    if buildCmd == "" {
        return fmt.Errorf("service %q has no build command configured", name)
    }

    // Allow build from any status except "building" (prevents concurrent builds).
    buildableStatuses := []Status{
        StatusHealthy, StatusFailed, StatusStopped,
        StatusPending, StatusStarting, StatusRetrying,
        StatusSkipped, StatusRestarting,
    }
    previousStatus := StatusFailed
    swapped := false
    for _, from := range buildableStatuses {
        if r.store.CompareAndSwapStatus(name, from, StatusBuilding) {
            previousStatus = from
            swapped = true
            break
        }
    }
    if !swapped {
        current := r.store.Get(name)
        if current == nil {
            return fmt.Errorf("service %q not found", name)
        }
        if current.Status == StatusBuilding {
            return fmt.Errorf("service %q is already building", name)
        }
        return fmt.Errorf("service %q is %s, cannot build right now", name, current.Status)
    }
    // ... rest remains the same
}
```

### 修改 2：`IsTerminalBuildStatus` — 扩展允许状态

**文件**: `runAll/src/domain/build_group_result_value_object.go:76-86`

```go
// IsTerminalBuildStatus reports whether a service status allows building.
// Only services actively building cannot start another build (concurrent build
// prevention); all other statuses are buildable because compilation is a
// disk-only operation independent of runtime state.
func IsTerminalBuildStatus(status string) bool {
    return status != ServiceStatusBuilding
}
```

或者更明确地列出：
```go
func IsTerminalBuildStatus(status string) bool {
    switch status {
    case ServiceStatusHealthy, ServiceStatusFailed, ServiceStatusStopped,
         ServiceStatusPending, ServiceStatusStarting, ServiceStatusRetrying,
         ServiceStatusSkipped, ServiceStatusRestarting:
        return true
    default:
        return false
    }
}
```

推荐使用第一种（排除法），因为未来新增状态时默认允许编译是安全的行为。

### 修改 3：`BuildGroup` — 已自动修复

**文件**: `runAll/src/runner.go:1512`

`BuildGroup` 使用 `IsTerminalBuildStatus` 过滤，修改 2 完成后自动生效，无需额外改动。

### 修改 4：测试更新

#### `runner_test.go:372-379` — `TestBuildService_StatusConflict`

原测试：服务状态为 `restarting`，期望编译报错。
新测试：服务状态为 `restarting`，期望编译成功（因为 restarting 现在允许编译）。

```go
// 改为: 只有 building 状态才拒绝编译
store.Update("build-conflict", StatusBuilding, "")
err = runner.BuildService(context.Background(), "build-conflict")
if err == nil {
    t.Fatal("expected already-building error")
}
if !strings.Contains(err.Error(), "already building") {
    t.Fatalf("unexpected error: %v", err)
}
```

#### `runner_test.go:410-416` — `TestBuildService_ConcurrentBuildRejected`

保持不变（并发 build 仍然被拒绝，因为第一个 build 已将状态置为 `building`）。

#### `build_group_result_value_object_test.go:109-130` — `TestIsTerminalBuildStatus`

将 `pending`、`starting`、`retrying`、`restarting`、`skipped` 的 expected 改为 `true`：

```go
{ServiceStatusHealthy, true},
{ServiceStatusFailed, true},
{ServiceStatusStopped, true},
{ServiceStatusBuilding, false},     // 唯一 false
{ServiceStatusStarting, true},      // 改为 true
{ServiceStatusRetrying, true},      // 改为 true
{ServiceStatusRestarting, true},    // 改为 true
{ServiceStatusPending, true},       // 改为 true
{ServiceStatusSkipped, true},       // 改为 true
```

### 不修改的部分

- **`restartService`** (runner.go:844)：restart 有自己的 CAS 状态门（healthy/failed/stopped/pending → restarting），其内部 `runBuild` 不经过 `BuildService`，行为不变。
- **`startService`** (runner.go:1306)：与 build 无关。

## 价值流影响

| 项目 | 影响 |
|------|------|
| **受影响流** | `runall-group-build-all` (云平台与资源域) — `runall-build-group-thin-slice` 步骤 |
| **字段变更** | `runall.runtime.build_group_skipped_names` — 含义从"因状态不允许而跳过"变为"仅在 building 状态冲突时跳过" |
| **测试文件** | `../../runAll/src/runner_test.go`、`../../runAll/src/domain/build_group_result_value_object_test.go` |
| **状态变更** | `runall-build-group-thin-slice` planned → 建议提升为 active |

## 域概念清单

| 类别 | 概念 |
|------|------|
| **Bounded Context** | 云平台与资源 (runAll 服务编排) |
| **Key Entity** | `Service` — 被管理的服务<br>`ServiceStatus` — 服务运行时状态 (pending/starting/retrying/healthy/failed/skipped/restarting/building/stopped) |
| **Domain Service** | `BuildService` — 单服务编译<br>`BuildGroup` — 分组批量编译 |
| **Business Rule** | 编译是磁盘操作，除了防止并发编译(building)外，不受任何运行时状态限制 |
