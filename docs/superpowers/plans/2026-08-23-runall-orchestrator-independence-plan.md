# 实施计划：runAll 编排器独立

- **Date:** 2026-08-23

## 任务

- [x] D1 领域策略 `KeepManagedServicesOnOrchestratorExit` + 单测 T1–T3
- [x] D2 `managedServiceSysProcAttr`（仅 Setsid；叠加 Setpgid 会 EPERM）+ T5
- [x] D3 `Runner.Run` UI wait 应用策略；信号处理器应用策略；日志
- [x] D4 T4：UI Run cancel 后进程仍存活 + Cleanup
- [x] D5 修复既有 `TestRun_UIMode_*` 的进程泄漏 Cleanup
- [x] D6 更新 `runAll/ai.md` 与 Ctrl+C 日志文案
- [x] D7 架构 v101 四件套 + VERSION_HISTORY

无 MQ publish 任务：意图已书面例外（运维策略）。
