# Intent: 全部重启只拉起 last-good 二进制（编译与重启分离）

## 背景与目标

2026-08-22 全部重启撞上 TDD 半写入的 `broker.DelayForAttempt`（`undefined`），fanout 消费者 readiness 超时。根因不是半截 ELF，而是：(1) `taskEvents/run.sh start` 隐式 `go build`；(2) stop 先于编译；(3) Agent 留下不可编译工作树。

目标：全部重启 / 普通启动只 exec 磁盘 last-good；编译仅出现在 build-all / 单服务编译 / 精准编译重启；精准路径先原子编译再切进程；Go 批次落地后模块必须能 `go build`。

## 范围与边界

- **范围内**：`RestartAll`/`StartAll`/`startService` 不调用 `runBuild`；`taskEvents/run.sh` `start_intent` 不再 `build_intent`；`restartService` 在非精准路径跳过编译；精准编译重启改为 compile-then-swap；约束 42 与代码对齐。
- **范围外**：Kafka 消费者在全量重启时空窗 DLT（OPT-20260822-002）；业务退避（失败经验 110，已修）。

## 约束与风险

- 不破坏 OPT-20260810-005 的「新二进制必须真正起来」：编译成功后仍 `forceFreshStart` + 端口扫尾。
- start 无二进制时失败信息必须指向 `build` / 精准编译重启，不得静默 go build。
- TDD 允许测试红，不允许包级 `undefined`。

## 验收标准

1. 全部重启不执行 `go build` / `build_command`。
2. `bash taskEvents/run.sh start <intent>` 在缺二进制时非零退出且不调用 `go build`。
3. 精准编译重启：编译失败则旧进程仍健康；成功则 PID 切换到新二进制。
4. 约束 42 正文为「编译 → 停止 → 启动」，与 `precise_restart.go` 一致。

## 实施计划

见 `docs/superpowers/specs/2026-08-22-restart-compile-separation-design.md`。

## 业务意图 → 事件对照

> 运维编排器进程生命周期，不产生业务领域事件。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| Intent: 全部重启只拉起 last-good 二进制 | — | — | — | — | 运维进程生命周期，不产生业务领域事件 |

## 变更记录

- 2026-08-22：新增。现场：`consumer/retry_wait.go:16:16: undefined: broker.DelayForAttempt`。
