# 实施计划：runAll 端口探活与停服兜底

> 设计：`docs/superpowers/specs/2026-06-03-runall-port-based-stop-liveness-design.md`

## 任务清单

- [x] **T1** `resolveServicePorts` 增加 `EffectiveStartCommand()` 端口解析
- [x] **T2** Runner：`listenerPIDsForPort`、`hasActivePortListeners`、domain 端口服务
- [x] **T3** 修订 `IsServiceRunning`（端口优先于 stopped）
- [x] **T4** `finalizeServiceStop`；stopped+端口仍开时继续停服
- [x] **T5** detach / 无 tracked 进程时 `ensureServicePortsReleased`（SIGTERM→SIGKILL）
- [x] **T6** 测试：`stopped`+stub listener、`StopService` stopped 孤儿
- [x] **T7** `go test ./runAll/src/...` 通过

## 验证命令

```bash
cd runAll && go test ./src/... -count=1
```
