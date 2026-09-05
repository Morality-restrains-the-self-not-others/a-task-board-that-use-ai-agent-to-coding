# 功能意图：runAll 挂掉不得拆掉托管服务

## 背景与目标

`:9999` 编排器退出时，托管业务服务必须继续提供能力；新 runAll 拉起后接管（adopt）已监听进程。

## 范围与边界

- **范围内：** UI 模式信号退出、监听丢失、shutdown-self、子进程 session 隔离、adopt/跳过 orphan kill。
- **范围外：** 用户点击「全部停止」/单服务停止；`RUNALL_SHUTDOWN_SERVICES=1` 调试拆栈。

## 约束与风险

- 子进程必须脱离 runAll 会话，避免 SIGHUP。
- 新实例禁止把仍健康的监听端口当孤儿 SIGKILL。
- 本意图为 **运维生命周期**，无租户业务状态变更。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | MQ类型/契约 |
|---------|--------|--------|--------|-------------|
| 无（编排器退出保留服务） | — | — | — | 无对应事件。理由：纯查询/运维进程策略，不改变租户域事实；不经 Kafka。结构化日志 `orchestrator_exit_keeps_services` 可观测。 |

## 验收标准

1. UI 模式 `ctx` 取消或 SIGTERM 后，已启动的托管进程仍存活。
2. 托管 `SysProcAttr.Setsid==true`，session id ≠ runAll。
3. `RUNALL_SHUTDOWN_SERVICES=1` 时信号路径仍可 Shutdown 托管服务。
4. 单测覆盖策略函数与「cancel 不杀进程」。

## 实施计划

见 `docs/superpowers/plans/2026-08-23-runall-orchestrator-independence-plan.md`。
