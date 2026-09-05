# ADR-0035: runAll 编排器与托管服务生命周期解耦

- **Status:** accepted
- **Date:** 2026-08-23
- **Author:** cursor
- **Deciders:** goal-mode 自动决策（可用性优先）

---

## Context

`:9999` Status UI（runAll）一旦因 SIGTERM、误杀、热替换失败或进程崩溃退出，会连带停止 `conf/runAll.yaml` 中的大部分托管服务。业务进程并不依赖 9999 端口，但监督器把「自身退出」实现成了「整栈 Shutdown」。热替换路径（`/api/shutdown-self`）已经证明「只退编排器、adopt 存量服务」可行。

## Decision

We will treat the runAll UI process as an **optional supervisor**:

1. UI 模式进程退出（信号、监听丢失、shutdown-self、默认 ctx cancel）**不得**停止托管服务，除非显式 `RUNALL_SHUTDOWN_SERVICES=1`。
2. 托管服务以 **独立 session**（`Setsid`）+ 独立进程组启动，避免会话首领死亡引发 SIGHUP。
3. 新 runAll 实例必须 **adopt** 仍在监听的托管端口，禁止把它们当 orphan `SIGKILL`（延续 OPT-20260820-008 / `shouldSkipOrphanCleanup`）。
4. 整栈停止的唯一用户意图入口仍是 Status UI / API 的 stop-all、单服务 stop、restart-all。

## Alternatives Considered

### Alternative 1: 保持 SIGTERM 拆栈，只加强 setsid 文档

- **Pros:** 行为与「监督器退出=停服务」传统直觉一致
- **Cons:** 9999 挂掉等于生产面不可用
- **Why rejected:** 与可用性目标相反

### Alternative 2: 外部 systemd 托管每个服务

- **Pros:** 业界标准
- **Cons:** 与现有 runAll DAG/ownership/精准编译重启冲突，迁移成本高
- **Why rejected:** 超出本次故障闭环

## Consequences

### Positive

- 编排器崩溃或重启不再级联停业务
- 与 shutdown-self / 9999 空窗拉起同一心智模型

### Negative / Trade-offs

- 前台 `Ctrl+C` 默认不再拆栈；须用 stop-all 或 `RUNALL_SHUTDOWN_SERVICES=1`
- 残留服务需依赖 adopt + ownership.json

### Mitigations

- 日志明确「orchestrator exiting, managed services kept」
- 调试环境变量恢复旧拆栈
- 残留无 UI 的 runAll 仍用 SIGKILL 收割（禁止 SIGTERM 误拆栈）
- 托管服务 stdout/stderr **继承日志文件 FD**，禁止 `StdoutPipe`/`StderrPipe`：编排器退出关闭 pipe 读端会给仍在写日志的子进程发 SIGPIPE（2026-08-29 WeChat callback 502，`trace_id=cf7b5ea2918884602b81fd80eae1efb1`）
- `waitTerminationSignal`（`rt_sigtimedwait`）必须重试 `EINTR`，禁止把非 INT/TERM 中断当成编排器退出

## References

- 设计：`docs/superpowers/specs/2026-08-23-runall-orchestrator-independence-design.md`
- `runAll/ai.md`
- ADR-0027 重启与编译分离（正交）
