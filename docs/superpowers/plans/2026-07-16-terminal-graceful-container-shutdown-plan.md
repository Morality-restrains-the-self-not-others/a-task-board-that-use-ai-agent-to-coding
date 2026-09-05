# 实施计划：terminal-graceful-container-shutdown

- 日期：2026-07-16
- 设计：`docs/superpowers/specs/2026-07-16-terminal-graceful-container-shutdown-design.md`

## Tasks

- [x] **T1** onlineServiceJS：`POST /api/task-lifecycle/shutdown` + 收尾 + 单测
- [x] **T2** taskCloudService：`request-machine-release` + SoleContainerGate + 单测
- [x] **T3** taskAgentSupport：登记 `request-machine-release`
- [x] **T4** taskEvents：Notifier；notify 成功发 `TASK_GRACEFUL_SHUTDOWN_AWAIT`
- [x] **T5** taskEvents：await handler + intent 注册 :18047
- [x] **T6** taskContainerGateway：L0 `container-task-lifecycle-shutdown`
- [x] **T7** 更新 machine_container.md、意图、DOMAIN_EVENTS、runAll
- [x] **T8** 回归既有 taskstatuschanged 单测

## 验证命令

```bash
cd taskEvents && go test ./internal/handlers/taskstatuschanged/... ./internal/handlers/taskgracefulshutdownawait/...
cd taskCloudService && go test ./src/ -count=1 -run 'RequestMachineRelease|Terminal|Sole'
cd trae-agent/onlineServiceJS && npm test -- --testPathPattern=taskLifecycle
```
