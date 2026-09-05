# 实施计划：金丝雀平滑重启

## Task 1: domain PlanCanaryRestartAll

- [ ] `IsCanaryRestartableStatus` + `LifecycleOperationCanaryRestart`
- [ ] `PlanCanaryRestartAll` 测试：healthy+stopped 都在、依赖序
- [ ] 实现 filter

## Task 2: tracelog.ListenAndServe

- [ ] Red: listen_test Shutdown 完成慢请求
- [ ] Linux: 双 bind SO_REUSEPORT
- [ ] `X-RunAll-Pid`；SIGTERM → Shutdown 25s

## Task 3: startAndCheck allowOverlapStart

- [ ] 端口占用且健康时仍 Start 新进程
- [ ] 保存 old *exec.Cmd，成功后只杀旧 PGID

## Task 4: RestartAll / PreciseRestart / UI

- [ ] RestartAllWithActor 循环 canary，不 StopAll
- [ ] restartService 非热替换走 canary
- [ ] SSE 单通道；确认文案

## Task 5: 迁移 Go mains + taskEvents REUSEPORT

- [ ] 各 `main.go` 换 ListenAndServe
- [ ] taskEvents ListenConfig 加 SO_REUSEPORT + Pid 头

## Task 6: 事件对照

- [ ] 意图文档已声明运维例外（无 MQ）
