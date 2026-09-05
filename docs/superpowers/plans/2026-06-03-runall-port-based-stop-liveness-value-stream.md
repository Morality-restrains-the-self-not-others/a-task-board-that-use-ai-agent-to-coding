# 价值流：runAll 端口探活与停服兜底

> 源自设计：`docs/superpowers/specs/2026-06-03-runall-port-based-stop-liveness-design.md`

## Value Summary

开发者在 runAll UI（`:9999`）关闭服务后，系统能**如实识别**端口仍被占用（detach 孤儿进程），并通过 **stop_command + kill -9** 清掉残留，避免「界面已停、`ps` 仍在」与清库误判。

## Related Value Streams

| 流 | 关系 |
|----|------|
| **runall-cascade-lifecycle**（`value-stream.yaml`） | **扩展** — 在 `runall-stop-cascade-failed-downstream` 同类场景上增加端口级验收 |
| **2026-05-27-runall-cascade-failed-downstream-stop** | **依赖** — failed+PID 须纳入级联停服；本流保证停后端口释放 |
| **2026-05-31-dev-database-reset** | **扩展** — 清库前 `IsServiceRunning` / `RunningApplicationsExcept` 须识别 stopped+端口开 |
| **2026-06-02-aimonitor-managed-detach** | **参考** — 同类 detach 启动；本流在 runAll 层统一端口语义 |
| **2026-05-31-runall-explicit-lifecycle-commands** | **依赖** — 停服仍走 YAML `stop_command`，kill -9 仅作端口兜底 |

## End-to-End Flow

```text
[开发者点击「关闭」或 dev-db 清库触停]
  → runAll stopService / stopServiceForDevDatabaseClear
  → runStopCommandIfConfigured（如 bash run.sh stop）
  → stopProcess（包装进程）
  → lsof：端口仍 LISTEN？
       → SIGTERM 监听 PID → 仍监听？ → SIGKILL（-9）
  → 无 LISTEN → store=stopped；否则 failed

[开发者查看状态 / 清库门禁]
  → IsServiceRunning
  → tracked 进程？ → 配置端口 LISTEN？ → 否则 store+health
  → 真·已停 vs 僵尸运行
```

## Value Increments

### Increment 1：端口探活（Thin Slice）

**Value to user:** 清库与 `RunningApplicationsExcept` 能发现「UI 显示 stopped 但端口仍占用」的服务，不再静默漏检。

**Scope:**

- `hasActivePortListeners(svc)`（复用 `resolveServicePorts` + `listenerPIDs`）
- 修订 `IsServiceRunning`（端口优先于 `store.status==stopped`）
- Go 单元测试：`stopped` + mock listener → true

**Depends on:** 无

**验收:** `go test ./runAll/src/... -run IsServiceRunning` 通过；手工 `lsof` 与 `RunningApplicationsExcept` 一致。

---

### Increment 2：停服阶梯（Core Value）

**Value to user:** 点击「关闭」或清库停服后，`task-events-*` 等业务进程实际退出（`ps` / 端口无 LISTEN）。

**Scope:**

- `stopService` / `stopServiceForDevDatabaseClear`：去掉裸 `stopped` early return（当端口仍监听）
- 停服顺序：`stop_command` → `stopProcess` → `terminateListenersByPort`（TERM → **KILL -9**）
- 仍监听 → `status=failed` + 错误含 port/pid
- 扩展 `runner_test.go`（detach listener + stopped 再 stop）
- 可选 `view_test/runall-detach-port-stop.md`（手工步骤）

**Depends on:** Increment 1

**验收:**

1. 启动 `task-events-email-sent-1-send-email` → UI 关闭 → `lsof` 无 18022 LISTEN  
2. store=stopped 且故意留孤儿 → 再 stop → 进程消失  
3. `go test ./runAll/src/...` 通过  

---

### Increment 3：状态可观测（Enhancement）

**Value to user:** UI 服务行可区分「已停」与「端口仍占用（僵尸）」。

**Scope:**

- `buildStatusPayload` 增加 `listen_port_active`（或 `hint` 文案）
- `status.html` 可选展示 ⚠ 标记（非阻塞 Increment 2）

**Depends on:** Increment 1

---

### Future（不纳入本期）

- `/api/status` reconcile 探活超时（另 spec）
- `taskEvents/run.sh stop` 非零退出码双保险
- 全局「关闭全部应用」按钮

## YAML 写入计划（待确认目标文件）

拟在 **`value-stream.yaml`** 的 `runall-cascade-lifecycle` 下追加步骤（或扩展 `runall-explicit-lifecycle-commands`）：

| step name | status | test_file |
|-----------|--------|-----------|
| `runall-port-listener-liveness` | planned → active | `../../runAll/src/runner_dev_database_running_test.go` |
| `runall-stop-port-kill-fallback` | planned | `view_test/runall-detach-port-stop.md` |

**Fields（三段式）:**

```yaml
- name: task-events-email-sent-1-send-email.runtime.listen_port_active
- name: task-events-email-sent-1-send-email.runtime.lifecycle_status
- name: docker-redis.runtime.lifecycle_status   # dev-db 清库门禁对照
```

（`task-events-*` 任选一个 intent 作代表步骤即可。）
