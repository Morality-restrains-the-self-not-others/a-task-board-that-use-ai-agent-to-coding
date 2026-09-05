# 设计：runAll 编排器与托管服务生命周期解耦

- **Date:** 2026-08-23
- **Iteration:** runall-orchestrator-independence
- **Status:** accepted（goal-mode 自动采用）
- **ADR:** [ADR-0035](../../adr/0035-runall-orchestrator-independence.md)

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief` 成功；当前图 `Languages: javascript, typescript, python, bash`，**不含 Go `runAll/`**。  
`CRG unavailable for Go runAll: graph has no Go nodes.` 设计基于源码阅读：`main.go` 信号环、`runner.Run` 退出路径、`SysProcAttr{Setpgid:true}`。

## 当前架构理解

- 企业景观 / 应用集成基线：**v100 current**（订单电子发票）。
- runAll 是 `:9999` Status UI + 进程监督器；托管服务由 `start_command` 拉起，健康检查独立监听。
- 已有热替换路径：`/api/shutdown-self` 与 UI 监听丢失会 `skipShutdownServices=true`，新实例 adopt 端口。

## 问题（根因）

当 `http://10.2.150.68:9999/` 对应的 **runAll 进程退出**时，大部分托管服务随之退出。两条耦合：

1. **主动拆栈（主因）**：`SIGINT`/`SIGTERM` 只 `cancel()` 运行环，**不**设置 `skipShutdownServices`。`Runner.Run` UI 模式在 `ctx.Done()` 后默认调用 `Shutdown()`，对每个托管进程组 `SIGTERM` → 5s → `SIGKILL`。
2. **会话 SIGHUP（硬杀放大）**：`run.sh` 用 `setsid` 让 runAll 成为会话首领；子进程仅 `Setpgid:true`（新进程组、**同会话**）。runAll 被 `SIGKILL`/崩溃时，内核可能向会话内进程发 `SIGHUP`，无独立 session 的服务一并退出。

不是「9999 端口本身被依赖」：业务服务不把 runAll 当运行时依赖。级联来自监督器退出策略。

## 决策（采用方案）

**编排器进程 ≠ 业务进程组。** UI 模式 runAll 退出默认**保留**托管服务；新实例 adopt 已监听端口（既有 `adoptListeningServices` + `shouldSkipOrphanCleanup`）。

| 退出原因 | 托管服务 |
|----------|----------|
| SIGTERM / SIGINT / 运行环 cancel | **保持运行** |
| `/api/shutdown-self`、UI bind/监听丢失 | **保持运行**（已有） |
| 进程 `SIGKILL` / panic 无 defer | **保持运行**（靠 `Setsid` 隔离） |
| UI「全部停止」/ 单服务停止 / 全部重启 | **按现有 API 停止** |
| `RUNALL_SHUTDOWN_SERVICES=1` + 信号 | 恢复旧行为（前台调试拆栈） |

子进程 `SysProcAttr`：`Setsid: true` + `Setpgid: true`（新会话、无控制终端；`Kill(-pid)` 仍可用）。

## Alternatives Considered

### A. systemd KillMode=process + 不改 Shutdown

- **Pros:** 不改 Go。
- **Cons:** 本环境由 `run.sh`/`setsid` 拉起，不是 unit；改不了 SIGTERM 处理。
- **Why rejected:** 未覆盖主因。

### B. 仅 Setsid、保留 SIGTERM 拆栈

- **Pros:** 硬杀不再 SIGHUP。
- **Cons:** `kill <runAll>` / 误 SIGTERM 仍拆整栈（用户现场主路径）。
- **Why rejected:** 不解决「9999 挂了服务全挂」。

### C. 采用方案（默认保留 + Setsid + adopt）

- **Pros:** 与 shutdown-self / 空窗拉起一致；业务可用性不绑编排器。
- **Cons:** Ctrl-C 不再拆栈（用 stop-all；可用 env 恢复）。

## 非目标

- 不把业务服务改成 systemd 用户 unit。
- 不新增 Kafka 业务事件（运维生命周期，见意图例外）。
- 不把 `runner.go` 一次削到 500 行（遗留巨石；本增量抽新文件，禁止继续堆逻辑）。

## 改动面

- `runAll/src/domain/orchestrator_exit_policy.go` — 退出是否保留服务
- `runAll/src/managed_service_proc_attr.go` — Setsid/Setpgid
- `runAll/src/main.go` — 信号路径套用策略
- `runAll/src/runner.go` — UI wait 套用策略；启动子进程用 helper
- `runAll/ai.md`、ADR-0035、架构 v101
