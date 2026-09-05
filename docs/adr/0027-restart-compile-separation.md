# ADR-0027: 进程重启与源码编译分离；编译失败保留 last-good

- **Status:** accepted
- **Date:** 2026-08-22
- **Author:** cursor
- **Deciders:** 工程团队（/1-brainstorming 总体设计审批：approve_full）

---

## Context

2026-08-22 全部重启在 start 阶段编译 `task-events` fanout 时撞上 TDD 半写入：`undefined: broker.DelayForAttempt`。旧进程已在 stop 阶段被杀掉，`go build` 失败（`set -e`）导致 last-good 二进制未被 exec，服务空窗并放大 Kafka DLT（失败经验 110）。

三条独立缺陷叠在一起：

1. **重启路径隐式编译**：`RestartAll` 本身不跑 `build_command`，但 `taskEvents/run.sh start` 总是 `go build` 工作树。
2. **kill-first（OPT-20260810-005）**：需要新二进制的路径先停再编，编译失败即停机。约束 42 仍写「编译 → 停止 → 启动」，与代码漂移。
3. **不可编译工作树**：Agent 先写引用、后写定义；并发的 runAll 把类型检查器当成了编排器。

`go build -o` 对 ELF 已是临时文件 + rename，**不是**半截二进制问题。

这是进程编排的重要设计约束，须单独 ADR，不新增架构视图组件。

## Decision

We will separate **process restart** from **source compile**.

1. **Restart/start/stop（含全部重启）只 exec 磁盘 last-good 二进制**，不得调用 `build_command`，也不得在 `start_command` 里 `go build` / `./build.sh`。
2. **Compile-only**（build-all / build-group / 单服务编译）只写磁盘，不杀进程。产物经临时文件 `rename` 到最终路径；失败则 last-good 不变。
3. **精准编译重启**改为 **compile-then-swap**：旧进程保持服务直到新二进制 rename 成功，再 stop + `forceFreshStart`。编译失败 → 旧进程继续、登记保留。
4. **Go 批次可编译门**：TDD 允许测试失败，不允许模块 `go build` 失败（`undefined:`）。同一批写入必须包含新符号的定义与引用。

OPT-20260810-005 的端口扫尾与「新二进制必须真正起来」仍然有效，但**不再**在编译失败路径上先杀进程。

## Alternatives Considered

### Alternative 1: 仅把 go build 改成显式 tmp+mv

- **Pros:** 改动面小
- **Cons:** 事故不是 torn ELF；先杀再编失败仍空窗
- **Why rejected:** 不满足「重启只拉二进制」

### Alternative 2: 保持 kill-first，只加长 readiness

- **Pros:** 不改启停顺序
- **Cons:** 脏树编译失败时永远起不来
- **Why rejected:** 与现场 stderr 不符

### Alternative 3: runAll 只编译 git HEAD，忽略脏工作树

- **Pros:** 避开 TDD 半写入
- **Cons:** 精准编译重启就是为了脏 WIP
- **Why rejected:** 根因是重启不该编译，而非禁止编脏树

## Consequences

### Positive

- 全部重启不再把 Agent 半成品编进生产空窗
- 编译失败时服务继续用 last-good
- 约束 42 与实现重新对齐

### Negative / Trade-offs

- 点「全部重启」不会带上未精准编译的代码；开发者可能误以为已生效
- 纠正 kill-first 后须防止旧进程残留（依赖编译成功后的 `forceFreshStart`）
- 冷启动无 bin 时 start 会失败，必须先 build
- **部署机（ADR-0052）**：无源码树时升级是 **download-then-swap**（`deploy-sync` + 钉版本）；`DEPLOY_MODE=1` 时 `resolveBuildCommand` 恒为空。精准编译重启仍只属于源码工作树。

### Mitigations

- UI/文档标明：重启 ≠ 编译
- `start` 缺二进制时错误指向 build / 精准编译重启
- 审计所有 `start_command` 是否夹带编译（举一反三）

## References

- 设计: `docs/superpowers/specs/2026-08-22-restart-compile-separation-design.md`
- 意图: `docs/intents/platform/runall_restart_compile_separation.intent.md`
- 约束 42: `.ai/01_project_constraints/42_precise_restart_service_registration.md`（实现时改为 compile-then-swap）
- 相关: OPT-20260810-005（kill-first，本 ADR 在编译路径上部分取代）；OPT-20260822-002（重启时暂停消费者，正交）
- 失败经验 110: `.ai/09_failure_experience/02_runtime_errors/110_task_status_changed_dlt_cloud_refused_retry_backoff_reset.md`
- 失败经验 111: `.ai/09_failure_experience/02_runtime_errors/111_task_events_restart_all_undefined_clearbackoff.md`
- [ADR-0052](0052-binary-deploy-config-repo.md) 配置仓与二进制部署；部署机 download-then-swap
