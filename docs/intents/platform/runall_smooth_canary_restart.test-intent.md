# Test Intent: 全部重启 / 精准编译重启支持金丝雀平滑重启

## 覆盖的功能意图

`docs/intents/platform/runall_smooth_canary_restart.intent.md`

## 测例

| ID | 场景 | 期望 | 实现 |
|----|------|------|------|
| T1 | 领域：健康+停止服务的 canary 计划含两者、依赖序 | `PlanCanaryRestartAll` 顺序与 start-all 拓扑一致且包含 Healthy | `runAll/src/domain/service_cascade_orchestration_service_test.go` |
| T2 | tracelog：SIGTERM 等价 stop 后 Shutdown，慢请求完成 | ListenAndServe 返回 nil；handler 跑完 | `shareLib/tracelog/listen_test.go` |
| T3 | tracelog：Linux 同端口双 Listen 成功 | 第二 listener 不 EADDRINUSE | `shareLib/tracelog/listen_linux_test.go` |
| T4 | 重叠启动：端口已被旧进程监听时仍启动新进程 | `allowOverlapStart` 不 skip、不 terminate residual | `runAll/src/runner_canary_restart_test.go` |
| T5 | 重叠失败回退：新进程 bind 失败则 drain 旧再 start | 最终新 PID 健康 | 同上 |
| T6 | RestartAll 不调用 StopAll | 健康探针在滚动中途仍 200 | `runAll/src/runner_restart_all_test.go` / `ui_restart_all_test.go` |
| T7 | UI 确认文案含金丝雀/平滑 | 组装后的 status.html 含新文案 | `ui_restart_all_test.go` |

## 非目标

- 不测真实 APISIX 生产流量切换（由重叠端口 + 健康检查间接保证）。
- 不测 Docker 基础设施双实例 MySQL。
