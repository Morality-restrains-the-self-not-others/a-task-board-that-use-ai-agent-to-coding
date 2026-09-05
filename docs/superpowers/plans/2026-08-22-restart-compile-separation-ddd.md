# DDD：重启与编译分离（运维限界上下文）

## Bounded Context

**PlatformOps / runAll** — 托管进程生命周期，不是租户业务域。

## 概念

| 概念 | 类型 | 说明 |
|------|------|------|
| LastGoodBinary | Value Object | 磁盘上一次成功 rename 的 ELF |
| RestartIntent | Command | 只 stop+start，禁止编译 |
| PreciseRestartIntent | Command | compile-then-swap |
| RegistrationFile | Entity | `.runall/precise_restart_services.txt` |

## 业务意图 → 事件

运维例外，无 MQ。见 `docs/intents/platform/runall_restart_compile_separation.intent.md`。

## 聚合不变量

1. Restart 不得调用 build 端口
2. Compile 失败不得杀死仍在服务的进程
3. PreciseRestart 错误返回时登记项保留，进程状态与事实一致（仍在听端口则为 healthy）
