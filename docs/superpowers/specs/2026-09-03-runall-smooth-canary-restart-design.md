# 金丝雀平滑重启：全部重启与精准编译重启

- **日期:** 2026-09-03
- **状态:** accepted（`/goal` 自动采用）
- **作者:** cursor
- **ADR:** [ADR-0058](../../adr/0058-runall-smooth-canary-restart.md)
- **架构视图:** v129 application-integration + enterprise-landscape（四件套）

## 当前架构理解

根据 current 架构（v128 ✅，2026-09-02 14:20）：

- 共有 2 个 current 视图：enterprise-landscape、application-integration
- 应用层：runAll `:9999` 编排 `conf/runAll.yaml` 托管进程；业务 Go 服务 + taskEvents 消费者 + APISIX
- 技术层：MySQL / Redis / Kafka；进程监听 `0.0.0.0`
- 上次交付：v128 阿里云 GitLab pending_node

📋 架构版本历史（最近）：v128 current → 本次 v129 target（金丝雀滚动重启）

## 问题

`button#restart-all-btn`（全部重启）与 `button#precise-restart-btn`（精准编译重启）在停旧启新之间切断流量。Go 服务几乎都是裸 `ListenAndServe`，SIGTERM 不排空。

## 🕸️ Code Review Graph 分析

CRG `update --brief` 成功（软依赖）。CodeGraph 索引存在。关键符号：

| 符号 | 位置 | 影响 |
|------|------|------|
| `RestartAllWithActor` | `runAll/src/runner_restart_all.go` | 改为 PlanCanaryRestartAll 循环 |
| `restartService` | `runAll/src/runner.go` | swap 半段改为 overlap |
| `startAndCheck` | `runAll/src/runner.go` | `allowOverlapStart` 禁止 skip/terminate |
| `stopProcess` | `runAll/src/runner.go` | 金丝雀只杀旧 PGID |
| `http.ListenAndServe` | 各服务 `main.go` | 换 `tracelog.ListenAndServe` |

## 选定方案

**滚动金丝雀（SO_REUSEPORT 重叠 + Shutdown 排空），失败回退 drain-then-start。**

```
compile? (仅精准路径, ADR-0027)
    ↓ success
记录 oldPID / oldPGID / oldListeners
    ↓
startAndCheck(allowOverlapStart=true)  // 不杀旧监听
    ↓ 新 PID 进入 listener 集合 且健康
SIGTERM(-oldPGID) → wait drain (25s) → 必要时 SIGKILL 旧组
    ↓ overlap 失败
SIGTERM 旧组排空 → startAndCheck(forceFreshStart)
```

### Go HTTP（源码驱动）

- 监听：[net.ListenConfig](https://pkg.go.dev/net#ListenConfig) `Control` 设 `SO_REUSEPORT`（Linux）+ `SO_REUSEADDR`
- 排空：[http.Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)（先关 listener，再等 idle；用带 25s deadline 的 context）
- 探针：响应头 `X-RunAll-Pid: <pid>`

### 编排

- `PlanCanaryRestartAll`：与 start-all 相同拓扑，包含 Healthy（`IsCanaryRestartableStatus`）
- RestartAll **不** StopAll；进度单通道 `operation=restart`（不再 stop→start 两段 SSE）
- 热替换 `skip_stop_on_restart` 不变

### 废弃

- 无 API 废弃。RestartAll 语义从「整栈先停」变为「滚动金丝雀」——UI 文案同步。

## Python 新增接口清单与 Go 替代评估

无新增 Python HTTP 接口（`not_applicable`）。

## 改动文件清单（实现）

- `shareLib/tracelog/listen*.go`
- `runAll/src/domain/service_cascade_orchestration_service.go` 等
- `runAll/src/runner_canary_restart.go`、`runner_restart_all.go`、`dag.go`、`ui_restart_all.go`、`status_ui/js/03.js` `05.js`
- 各 Go 服务 `main.go`；`taskEvents/consumer/runner.go`（SO_REUSEPORT）

## 架构变更影响

新增应用组件：`tracelog.ListenAndServe`（共享库）；修改 runAll 生命周期关系。见 v129 四件套。
