# 实施计划：ADR-0027 重启与编译分离

## Tasks

- [x] **T1** 红灯：`TestRestartAll_DoesNotRunBuildCommand`、`TestRestartService_IgnoresBuildCommand`、`TestRestartService_CompileThenSwap_BuildFailureKeepsProcess`
- [x] **T2** `restartService`：默认不编译；`withCompileThenSwap(ctx)` 先编后切；失败恢复监控与 previousStatus
- [x] **T3** `PreciseRestart` 传入 compile-then-swap ctx
- [x] **T4** 改写 kill-first 构建失败测例
- [x] **T5** `taskEvents/run.sh` start 去掉 `build_intent`；缺 bin 明确失败；build 用 tmp+mv
- [x] **T6** 约束 42、runAll/ai.md、失败经验 111
- [x] **T7** 单测全绿；登记精准重启 runAll；提交推送

无新领域事件任务（运维例外）。
