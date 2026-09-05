# NFR 澄清：runAll 端口探活与停服兜底

> 设计：`docs/superpowers/specs/2026-06-03-runall-port-based-stop-liveness-design.md`  
> 价值流：`docs/superpowers/plans/2026-06-03-runall-port-based-stop-liveness-value-stream.md`

## 增量范围

Increment 1–2（端口探活 + stop_command → kill -9），本地开发 orchestrator，无生产多租户。

## NFR 支撑等级（默认 L2，本地工具 L1）

| 类别 | 等级 | 说明 |
|------|------|------|
| 性能 | L1 | 单次 `lsof` + 停服等待 ≤500ms；不优化批量并行 |
| 可用性 | L2 | 停服失败标 `failed` 并带 port/pid，开发者可重试 |
| 数据一致性 | 不适用 | 无持久化状态机，仅内存 store |
| 安全性 | L2 | 仅杀本服务配置端口上的 PID；不误杀无端口服务 |
| 可观测性 | L2 | `[svc] stop:` 日志含端口与 PID |
| 可测试性 | L2 | `listenerPIDsFn` 可注入；不依赖真 lsof 的单元测试 |
| 容错 | L2 | SIGTERM → 250ms → SIGKILL；与 preflight 一致 |

## 质量场景

| ID | 刺激 | 响应 |
|----|------|------|
| QS-1 | store=stopped，18022 仍 LISTEN | `IsServiceRunning` 在 100ms 内为 true |
| QS-2 | 开发者 POST /api/stop，stop_command 无效 | 250ms 内端口无 LISTEN 或 status=failed |
| QS-3 | dev-db 清库前 18 个 task-events 僵尸 | `RunningApplicationsExcept` 列出全部，清库循环触发 stop |
| QS-4 | 端口被无关进程占用 | 可能误报 running；接受（task-events 独占端口） |

## 领域模型影响

| NFR | 建模影响 |
|-----|----------|
| QS-1 | `PortListenerProbeRepository` 纳入 `IsServiceRunning` 路径 |
| QS-2 | 复用 `ForeignProcessTerminationRepository`（SIGTERM→SIGKILL）于停服阶梯 |
| QS-3 | `stopService` 在 stopped+端口开 时跳过 early return |

## 明确不做

- 跨主机远程杀进程
- `/api/status` reconcile 超时（另项）
- 修改 `taskEvents/run.sh` 退出码（二期）
