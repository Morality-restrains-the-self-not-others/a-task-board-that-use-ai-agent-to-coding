# runAll 日志复制与 git-oauth 端口冲突恢复 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `http://localhost:9999/` 增加日志复制按钮，并修复 UI 手动启动 `git-oauth` 时因跳过 preflight 导致的 `Address already in use` 问题。

**Architecture:** 领域层已提供 `ServicePreflightDomainService` 与端口冲突值对象；基础设施层用 `lsof`/`syscall` 实现三个 Repository 适配器；`preflight.go` 与 `startService()` 统一调用 `EnsurePortsReadyForLaunch`；`status.html` 纯前端实现剪贴板复制与轻量反馈，不新增后端 API。

**Tech Stack:** Go 1.x, embed HTML/JS, `lsof`, `syscall`, Go test, runAll UI at `:9999`

**Inputs:**
- Value Stream: `docs/superpowers/plans/2026-05-26-runall-log-copy-gitoauth-port-conflict-value-stream.md`
- NFR: `docs/superpowers/plans/2026-05-26-runall-log-copy-gitoauth-port-conflict-nfr-clarification.md`
- DDD: `docs/superpowers/plans/2026-05-26-runall-log-copy-gitoauth-port-conflict-ddd-model.md`

---

## File Structure

| 文件 | 职责 |
|------|------|
| `runAll/src/domain/*` | 已完成：端口冲突 VO、`ServicePreflightDomainService`、仓储接口 |
| Create: `runAll/src/infrastructure/lsof_port_listener_probe_repository.go` | `PortListenerProbeRepository` 实现 |
| Create: `runAll/src/infrastructure/syscall_foreign_process_termination_repository.go` | `ForeignProcessTerminationRepository` 实现 |
| Create: `runAll/src/infrastructure/runner_owned_process_registry_repository.go` | `OwnedProcessRegistryRepository` 实现 |
| Modify: `runAll/src/preflight.go` | 委托 `ServicePreflightDomainService`，删除重复 `filterForeignPIDs` |
| Modify: `runAll/src/runner.go` | `startService()` 在 Launch 前调用 `runPreflight()` |
| Modify: `runAll/src/status.html` | 复制按钮 + 反馈文案 |
| Modify: `runAll/src/ui_test.go` | HTML 片段测试 |
| Modify: `runAll/src/runner_test.go` | 手动启动 preflight 集成测试 |
| Modify: `runAll/src/preflight_test.go` | preflight 适配器测试（新建） |

---

### Task 0: 确认领域层契约已就绪（前置）

**Files:**
- Verify: `runAll/src/domain/service_preflight_domain_service.go`
- Verify: `runAll/src/domain/port_conflict_snapshot_value_object.go`
- Test: `runAll/src/domain/*_test.go`

- [ ] **Step 1: 运行领域层测试**

Run: `cd runAll/src && go test ./domain/... -count=1 -v`  
Expected: PASS（所有 domain 包测试通过）

- [ ] **Step 2: 确认无基础设施泄漏**

Run: `rg 'os/exec|syscall|lsof|net/http' runAll/src/domain/`  
Expected: 无匹配（领域层保持纯净）

---

### Task 1: 日志面板「复制日志」按钮（Increment 1）

**Files:**
- Modify: `runAll/src/status.html`
- Modify: `runAll/src/ui_test.go`

- [ ] **Step 1: 写失败测试（HTML 必须包含复制 hook）**

在 `runAll/src/ui_test.go` 的 `TestUIHomePage_ContainsRequiredSnippets` 的 `requiredSnippets` 中追加：

```go
`id="logs-panel-copy"`,
`async function copyLogsToClipboard()`,
`navigator.clipboard.writeText`,
```

并新增测试：

```go
func TestUIHomePage_LogCopyFailureFallbackPresent(t *testing.T) {
	store := NewStatusStore()
	store.Init([]string{"svc"})
	mux := http.NewServeMux()
	registerUIHandlers(mux, store, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, snippet := range []string{
		`function copyLogsToClipboard()`,
		`updateLogsMeta('copy failed`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("status.html missing log copy snippet %q", snippet)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll/src && go test -run 'TestUIHomePage_.*LogCopy' -v`  
Expected: FAIL（缺少 copy 按钮/handler）

- [ ] **Step 3: 实现复制按钮与 handler**

在 `status.html` 的 `.logs-panel-actions` 中，`Refresh` 前插入：

```html
<button id="logs-panel-copy" class="logs-panel-btn" type="button">Copy</button>
```

在 `<script>` 中追加：

```javascript
async function copyLogsToClipboard() {
  const contentEl = document.getElementById('logs-content');
  const text = contentEl ? contentEl.textContent : '';
  if (!text || text.trim() === '' || text.includes('Select a service')) {
    updateLogsMeta('copy failed: no logs');
    return;
  }
  try {
    if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
      await navigator.clipboard.writeText(text);
    } else {
      throw new Error('clipboard API unavailable');
    }
    updateLogsMeta('copied');
  } catch (err) {
    updateLogsMeta(`copy failed: ${err.message || 'unknown error'}`);
  }
}
```

绑定事件（紧接 `logs-panel-refresh` listener 之后）：

```javascript
document.getElementById('logs-panel-copy').addEventListener('click', (event) => {
  pulseClickFeedback(event.currentTarget);
  copyLogsToClipboard();
});
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll/src && go test -run 'TestUIHomePage_' -v`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `cd runAll && git add src/status.html src/ui_test.go && git commit -m "feat(runAll): add copy logs button to status panel"`

---

### Task 2: 基础设施 Repository 适配器

**Files:**
- Create: `runAll/src/infrastructure/lsof_port_listener_probe_repository.go`
- Create: `runAll/src/infrastructure/syscall_foreign_process_termination_repository.go`
- Create: `runAll/src/infrastructure/runner_owned_process_registry_repository.go`
- Create: `runAll/src/infrastructure/preflight_repository_adapters_test.go`

- [ ] **Step 1: 写失败测试**

```go
package infrastructure

import "testing"

func TestLsofPortListenerProbeRepository_ListListeningPIDs_InvalidPort(t *testing.T) {
	repo := NewLsofPortListenerProbeRepository(nil)
	if _, err := repo.ListListeningPIDs(""); err == nil {
		t.Fatal("expected empty port to fail")
	}
}

func TestSyscallForeignProcessTerminationRepository_Terminate_Empty(t *testing.T) {
	repo := NewSyscallForeignProcessTerminationRepository()
	if err := repo.Terminate(nil); err != nil {
		t.Fatalf("empty terminate should succeed: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll/src && go test ./infrastructure/... -run TestLsof -v`  
Expected: FAIL（类型未定义）

- [ ] **Step 3: 实现三个适配器**

`lsof_port_listener_probe_repository.go`:

```go
type LsofPortListenerProbeRepository struct {
	listenerFn func(port string) ([]int, error)
}

func NewLsofPortListenerProbeRepository(listenerFn func(string) ([]int, error)) LsofPortListenerProbeRepository {
	return LsofPortListenerProbeRepository{listenerFn: listenerFn}
}

func (r LsofPortListenerProbeRepository) ListListeningPIDs(port string) ([]int, error) {
	if strings.TrimSpace(port) == "" {
		return nil, fmt.Errorf("port is required")
	}
	if r.listenerFn == nil {
		return nil, fmt.Errorf("listener function is required")
	}
	return r.listenerFn(port)
}
```

`syscall_foreign_process_termination_repository.go` — 将现有 `terminatePIDs` 逻辑移入此文件（从 `preflight.go` 提取）。

`runner_owned_process_registry_repository.go`:

```go
type RunnerOwnedProcessRegistryRepository struct {
	ownedFn func() map[int]struct{}
}

func NewRunnerOwnedProcessRegistryRepository(ownedFn func() map[int]struct{}) RunnerOwnedProcessRegistryRepository {
	return RunnerOwnedProcessRegistryRepository{ownedFn: ownedFn}
}

func (r RunnerOwnedProcessRegistryRepository) OwnedPIDs() map[int]struct{} {
	if r.ownedFn == nil {
		return map[int]struct{}{}
	}
	return r.ownedFn()
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll/src && go test ./infrastructure/... -count=1`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `cd runAll && git add src/infrastructure/lsof_port_listener_probe_repository.go src/infrastructure/syscall_foreign_process_termination_repository.go src/infrastructure/runner_owned_process_registry_repository.go src/infrastructure/preflight_repository_adapters_test.go && git commit -m "feat(runAll): add preflight repository adapters"`

---

### Task 3: 重构 preflight.go 委托领域服务（Increment 2/3 核心）

**Files:**
- Modify: `runAll/src/preflight.go`
- Create: `runAll/src/preflight_test.go`

- [ ] **Step 1: 写失败测试（preflight 使用 domain 服务并记录 failure）**

```go
func TestPreflightService_UsesDomainServiceAndRecordsPortConflict(t *testing.T) {
	runner := newRunnerWithPreflightDeps(t)
	runner.listenerPIDsFn = func(port string) ([]int, error) {
		if port == "8002" {
			return []int{4242}, nil
		}
		return nil, nil
	}

	svc := Service{
		Name: "git-oauth",
		HealthCheck: HealthCheck{URL: "http://127.0.0.1:8002/api/health/"},
	}
	err := runner.preflightService(context.Background(), svc)
	if err == nil {
		t.Fatal("expected port conflict when cleanup disabled")
	}
	status := runner.store.Get("git-oauth")
	if status == nil || !strings.Contains(status.Error, domain.ServiceFailureCodePortConflict) {
		t.Fatalf("expected structured port conflict, got %+v", status)
	}
}
```

实现时可通过 `runner.preflightFn` 注入或增加 `autoCleanupPorts bool` 测试开关；生产路径 `autoCleanup=true`。

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll/src && go test -run TestPreflightService_UsesDomainService -v`  
Expected: FAIL

- [ ] **Step 3: 重构 preflightService**

```go
func (r *Runner) newPreflightDomainService() domain.ServicePreflightDomainService {
	listenerFn := r.listenerPIDsFn
	if listenerFn == nil {
		listenerFn = listenerPIDs
	}
	return domain.NewServicePreflightDomainService(
		infrastructure.NewLsofPortListenerProbeRepository(listenerFn),
		infrastructure.NewSyscallForeignProcessTerminationRepository(),
		infrastructure.NewRunnerOwnedProcessRegistryRepository(r.ownedProcessPIDs),
	)
}

func (r *Runner) preflightService(ctx context.Context, svc Service) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	ports := resolveServicePorts(&svc)
	if len(ports) == 0 {
		return nil
	}

	service := r.newPreflightDomainService()
	results, err := service.EnsurePortsReadyForLaunch(svc.Name, ports, true)
	if err != nil {
		r.store.RecordPreflightFailure(svc.Name, domain.ServiceFailureCodePortConflict, err.Error())
		return fmt.Errorf("[%s] %s", svc.Name, err.Error())
	}
	for _, result := range results {
		if len(result.TerminatedPIDs) > 0 {
			log.Printf("[%s] preflight cleaned foreign listeners on port %s (pid=%s)",
				svc.Name, result.Port, joinPIDs(result.TerminatedPIDs))
		}
	}
	return nil
}
```

删除 `preflight.go` 中重复的 `filterForeignPIDs`，改为使用 `domain.FilterForeignPIDs`（若 `joinPIDs` 仍被使用则保留）。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd runAll/src && go test -run TestPreflightService -v`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `cd runAll && git add src/preflight.go src/preflight_test.go && git commit -m "refactor(runAll): delegate port preflight to domain service"`

---

### Task 4: 手动启动路径接入 preflight（闭合 Errno 48 根因）

**Files:**
- Modify: `runAll/src/runner.go`
- Modify: `runAll/src/runner_test.go`

- [ ] **Step 1: 写失败测试**

```go
func TestRunner_StartServiceWithActor_RunsPreflightBeforeLaunch(t *testing.T) {
	runner, cleanup := newRunnerForManualStartPreflightTest(t)
	defer cleanup()

	preflightCalled := false
	origPreflight := runner.preflightFn
	runner.preflightFn = func(ctx context.Context, svc Service) error {
		preflightCalled = true
		if origPreflight != nil {
			return origPreflight(ctx, svc)
		}
		return runner.preflightService(ctx, svc)
	}

	runner.listenerPIDsFn = func(port string) ([]int, error) {
		return nil, nil
	}

	if err := runner.StartServiceWithActor(context.Background(), "git-oauth", "ui-session"); err != nil {
		t.Fatalf("StartServiceWithActor: %v", err)
	}
	if !preflightCalled {
		t.Fatal("expected preflight before manual launch")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd runAll/src && go test -run TestRunner_StartServiceWithActor_RunsPreflightBeforeLaunch -v`  
Expected: FAIL（preflightCalled=false）

- [ ] **Step 3: 在 startService 中插入 runPreflight**

在 `CompareAndSwapStatus` 成功后、`startAndCheck` 之前：

```go
	if err := r.runPreflight(ctx, *svc); err != nil {
		r.store.SetPID(name, 0)
		return err
	}
```

`runPreflight` 已存在时直接复用；preflight 失败时 `RecordPreflightFailure` 已将状态置为 `failed`。

- [ ] **Step 4: 写端口冲突恢复集成测试**

```go
func TestRunner_StartServiceWithActor_CleansForeignPortConflict(t *testing.T) {
	runner, foreignPID, cleanup := spawnForeignListenerOnPort(t, "8002")
	defer cleanup()

	if err := runner.StartServiceWithActor(context.Background(), "git-oauth", "ui-session"); err != nil {
		t.Fatalf("StartServiceWithActor after conflict cleanup: %v", err)
	}
	_ = foreignPID
	status := runner.store.Get("git-oauth")
	if status == nil || status.Status != StatusHealthy {
		t.Fatalf("expected healthy git-oauth, got %+v", status)
	}
}
```

若现有 test helper 无 `spawnForeignListenerOnPort`，可复用 `runner_test.go` 中端口冲突场景 helper 或 stub `listenerPIDsFn` + 模拟 terminate。

- [ ] **Step 5: 运行测试确认通过**

Run: `cd runAll/src && go test -run 'TestRunner_StartServiceWithActor_' -v`  
Expected: PASS

- [ ] **Step 6: 提交**

Run: `cd runAll && git add src/runner.go src/runner_test.go && git commit -m "fix(runAll): run preflight before UI manual service start"`

---

### Task 5: UI 交互反馈增强（Increment 4）

**Files:**
- Modify: `runAll/src/status.html`
- Modify: `runAll/src/ui_test.go`

- [ ] **Step 1: 写失败测试**

```go
func TestUIHomePage_PreflightFeedbackSnippetsPresent(t *testing.T) {
	// ... fetch home page ...
	for _, snippet := range []string{
		`updateLogsMeta('copied'`,
		`pulseClickFeedback(event.currentTarget)`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("missing feedback snippet %q", snippet)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败/通过基线**

Run: `cd runAll/src && go test -run TestUIHomePage_PreflightFeedback -v`

- [ ] **Step 3: 增强 copyLogsToClipboard 反馈**

复制成功时将 meta 文案设为 `` `${logsState.service} - copied at ${formatNowTime()} - last ${logLines} lines` ``；失败保留 `copy failed: ...`。

启动失败时，服务行已有 `error-msg` 与 `hint`（来自 `/api/status`）；确认 `deriveFailureHint` 返回的 preflight 文案含 `PRECHECK_PORT_CONFLICT` 与 `pid=`（Task 3 已保证）。

- [ ] **Step 4: 运行全量 UI 测试**

Run: `cd runAll/src && go test -run TestUI -count=1`  
Expected: PASS

- [ ] **Step 5: 提交**

Run: `cd runAll && git add src/status.html src/ui_test.go && git commit -m "feat(runAll): improve log copy and preflight feedback in status UI"`

---

### Task 6: 全量验证与价值流状态更新

**Files:**
- Modify: `value-stream.yaml`（可选：将 4 个 step 从 `planned` 改 `active` 并指向真实 test_file）
- Verify: `docs/superpowers/plans/2026-05-26-runall-log-copy-gitoauth-port-conflict-value-stream.md`

- [ ] **Step 1: 运行 runAll 全量测试**

Run: `cd runAll/src && go test ./... -count=1`  
Expected: PASS

- [ ] **Step 2: 手工验收（git-oauth 端口冲突）**

1. 手工占用 8002：`python3 -c "import socket; s=socket.socket(); s.bind(('0.0.0.0',8002)); input('hold')"`  
2. 打开 `http://localhost:9999/`，将 `git-oauth` 置为 stopped 后点「启动」  
3. Expected: preflight 清理或结构化失败（含 port/pid）；不应仅见 Python traceback  
4. 打开日志面板，点 Copy，粘贴到编辑器验证内容一致

- [ ] **Step 3: 更新 value-stream.yaml（实现完成后）**

将 `runall-log-copy-gitoauth-port-conflict-recovery` 下各 step 的 `test_file` 改为：
- `runAll/src/ui_test.go`（thin slice + feedback）
- `runAll/src/preflight_test.go`
- `runAll/src/runner_test.go`

Run: `cd valueStream && go test ./... -count=1`  
Expected: PASS

- [ ] **Step 4: 提交**

Run: `git add value-stream.yaml && git commit -m "chore(value-stream): activate runall log copy port conflict recovery steps"`

---

## Self-Review

| 规格/requirement | 对应 Task |
|------------------|-----------|
| Increment 1 日志复制 | Task 1 |
| Increment 2 端口冲突可诊断 | Task 3 + Task 4（preflight 结构化错误） |
| Increment 3 自动恢复 | Task 2 + Task 3 + Task 4 |
| Increment 4 UI 反馈 | Task 5 |
| NFR 一致性：手动/批量 preflight 统一 | Task 4 |
| NFR 安全：仅 foreign PID | Task 2 + Task 3（domain.FilterForeignPIDs） |
| 领域层已完成 | Task 0 |
| 无新后端 copy API | Task 1 纯前端 |

**Placeholder scan:** 无 TBD/TODO/“implement later”。

**Type consistency:** `ServicePreflightDomainService.EnsurePortsReadyForLaunch(serviceName, ports, autoCleanup bool)` 与 Task 3/4 调用一致；Repository 构造函数名与 infrastructure 文件一致。

---

## 验收标准（DoD）

- [ ] 日志面板有 Copy 按钮，可复制当前可见日志
- [ ] UI 手动启动 `git-oauth` 前执行 preflight
- [ ] 8002 被外部进程占用时，自动清理或返回含 `PRECHECK_PORT_CONFLICT` + pid 的结构化错误
- [ ] `go test ./...` 在 `runAll/src` 通过
- [ ] 不破坏现有 stop/restart/ownership 测试
